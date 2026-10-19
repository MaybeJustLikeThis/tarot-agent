package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestFetchLatestUsesPublicAPI(t *testing.T) {
	var got *http.Request
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got = r
		return jsonResponse(200, `{"tag_name":"v0.6.0","html_url":"https://github.com/MaybeJustLikeThis/tarot-agent/releases/tag/v0.6.0"}`), nil
	})}
	rel, err := FetchLatest(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v0.6.0" || !strings.Contains(rel.URL, "/releases/tag/v0.6.0") {
		t.Fatalf("release %+v", rel)
	}
	if got.URL.String() != LatestURL {
		t.Fatalf("url %s", got.URL)
	}
	if got.Header.Get("Accept") != "application/vnd.github+json" {
		t.Fatalf("accept %q", got.Header.Get("Accept"))
	}
	if got.Header.Get("Authorization") != "" {
		t.Fatal("public request must not send a token")
	}
	if got.Header.Get("User-Agent") == "" {
		t.Fatal("GitHub requires a User-Agent")
	}
}

func TestCheckNewerOlderAndBadTag(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"tag_name":"v0.6.0","html_url":"https://example.test/r"}`), nil
	})}
	res, err := Check(context.Background(), client, "0.5.0")
	if err != nil || !res.Newer || res.Latest != "v0.6.0" {
		t.Fatalf("newer: %+v err %v", res, err)
	}
	res, err = Check(context.Background(), client, "v0.6.0")
	if err != nil || res.Newer {
		t.Fatalf("equal should not nag: %+v err %v", res, err)
	}
	res, err = Check(context.Background(), client, "1.0.0")
	if err != nil || res.Newer {
		t.Fatalf("local newer: %+v err %v", res, err)
	}

	called := false
	skipClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("should not be called")
	})}
	res, err = Check(context.Background(), skipClient, "dev")
	if err != nil || !res.Skipped || called {
		t.Fatalf("dev: %+v called %v err %v", res, called, err)
	}

	bad := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"tag_name":"nightly","html_url":"https://example.test/r"}`), nil
	})}
	res, err = Check(context.Background(), bad, "0.5.0")
	if err != nil || !res.Unparsed || res.Newer {
		t.Fatalf("bad tag: %+v err %v", res, err)
	}
}

func TestCheckFailureIsAnError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(403, `{"message":"rate limit"}`), nil
	})}
	_, err := Check(context.Background(), client, "0.5.0")
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	slow := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	_, err = Check(ctx, slow, "0.5.0")
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestFormatReport(t *testing.T) {
	text := FormatReport(&Result{Current: "0.5.0", Latest: "v0.6.0", URL: "https://example.test/r", Newer: true})
	for _, want := range []string{"当前版本：0.5.0", "最新版本：v0.6.0", "有更新", "go install " + InstallPath + "@latest", "https://example.test/r"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
	same := FormatReport(&Result{Current: "0.6.0", Latest: "v0.6.0"})
	if !strings.Contains(same, "已是最新版本") {
		t.Fatalf("same:\n%s", same)
	}
	dev := FormatReport(&Result{Current: "dev", Skipped: true})
	if !strings.Contains(dev, "不做更新提示") || strings.Contains(dev, "有更新") {
		t.Fatalf("dev:\n%s", dev)
	}
	notice := ShortNotice("0.5.0", "v0.6.0")
	if !strings.Contains(notice, "0.5.0 → v0.6.0") || strings.Count(notice, "\n") != 1 {
		t.Fatalf("notice %q", notice)
	}
}
