package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestExecuteAlreadyLatestDoesNotInstall(t *testing.T) {
	client := releaseClient(t, "v0.6.0", nil)
	var out strings.Builder
	err := Execute(context.Background(), ExecuteOptions{
		Current: "0.6.0",
		Stdout:  &out,
		Stderr:  io.Discard,
		Client:  client,
		LookGo: func() (string, error) {
			t.Fatal("already latest should not look for go")
			return "", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "已是最新版本") {
		t.Fatalf("out:\n%s", out.String())
	}

	out.Reset()
	err = Execute(context.Background(), ExecuteOptions{
		Current: "1.0.0",
		Stdout:  &out,
		Stderr:  io.Discard,
		Client:  releaseClient(t, "v0.6.0", nil),
		LookGo:  func() (string, error) { t.Fatal("local newer should not install"); return "", nil },
	})
	if err != nil || !strings.Contains(out.String(), "已是最新版本") {
		t.Fatalf("err %v out %s", err, out.String())
	}
}

func TestExecuteNonInteractiveReportsAvailable(t *testing.T) {
	var out strings.Builder
	err := Execute(context.Background(), ExecuteOptions{
		Current:    "0.5.0",
		Stdout:     &out,
		Stderr:     io.Discard,
		IsTerminal: false,
		Client:     releaseClient(t, "v0.6.0", nil),
		LookGo:     func() (string, error) { t.Fatal("should not install"); return "", nil },
	})
	if !errors.Is(err, ErrAvailable) {
		t.Fatalf("err %v", err)
	}
	text := out.String()
	for _, want := range []string{"0.5.0 → v0.6.0", "https://example.test/r", "tarot-agent update -y", "要现在更新到"} {
		if want == "要现在更新到" {
			if strings.Contains(text, want) {
				t.Fatalf("non-interactive run asked a question:\n%s", text)
			}
			continue
		}
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
}

func TestExecutePromptAndGoInstall(t *testing.T) {
	var installed string
	opt := ExecuteOptions{
		Current:    "dev",
		Yes:        false,
		IsTerminal: true,
		Stderr:     io.Discard,
		Client:     releaseClient(t, "v0.6.0", nil),
		LookGo:     func() (string, error) { return "/usr/bin/go", nil },
		GoInstall: func(ctx context.Context, moduleAt string) error {
			installed = moduleAt
			return nil
		},
		ReportGo: func(context.Context) (string, string) {
			return "/home/user/go/bin/tarot-agent", "tarot-agent dev"
		},
		Place: func(string, string, []byte) (string, error) {
			t.Fatal("go install should not download a release binary")
			return "", nil
		},
	}

	var declined strings.Builder
	opt.Stdout = &declined
	opt.Stdin = strings.NewReader("n\n")
	if err := Execute(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if installed != "" || !strings.Contains(declined.String(), "已取消") || !strings.Contains(declined.String(), "要现在更新到 v0.6.0 吗？[y/N]") {
		t.Fatalf("installed %q out %s", installed, declined.String())
	}

	var accepted strings.Builder
	opt.Stdout = &accepted
	opt.Stdin = strings.NewReader("y\n")
	if err := Execute(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if installed != InstallPath+"@v0.6.0" {
		t.Fatalf("installed %q", installed)
	}
	text := accepted.String()
	for _, want := range []string{"which: /home/user/go/bin/tarot-agent", "tarot-agent dev", "代码已是 v0.6.0"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
}

func TestExecuteReleaseAssetAndPermission(t *testing.T) {
	archive := tarGz(t, map[string]string{"tarot-agent": "new-binary"})
	name, err := AssetName("linux", "amd64", "v0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive)
	good := hex.EncodeToString(sum[:]) + "  " + name + "\n"

	var got []byte
	_, out := runAssetUpdate(t, archive, good, name, "/usr/local/bin/tarot-agent", func(goos, dest string, data []byte) (string, error) {
		if goos != "linux" || dest != "/usr/local/bin/tarot-agent" {
			t.Fatalf("place %s %s", goos, dest)
		}
		got = append([]byte(nil), data...)
		return "", nil
	})
	if string(got) != "new-binary" {
		t.Fatalf("placed %q", got)
	}
	if !strings.Contains(out, "已替换 /usr/local/bin/tarot-agent") {
		t.Fatalf("out %s", out)
	}

	_, errText := runAssetUpdateErr(t, archive, "deadbeef  "+name+"\n", name, "/usr/local/bin/tarot-agent", func(string, string, []byte) (string, error) {
		t.Fatal("checksum mismatch must not write")
		return "", nil
	})
	if !strings.Contains(errText, "校验失败") {
		t.Fatalf("err %s", errText)
	}

	_, errText = runAssetUpdateErr(t, archive, good, name, "/usr/local/bin/tarot-agent", func(string, string, []byte) (string, error) {
		return "", os.ErrPermission
	})
	for _, want := range []string{"无法写入 /usr/local/bin/tarot-agent", "curl -fL", name, "mv tarot-agent", "/usr/local/bin/tarot-agent"} {
		if !strings.Contains(errText, want) {
			t.Errorf("permission message missing %q\n%s", want, errText)
		}
	}
}

func TestInstallOrder(t *testing.T) {
	archive := tarGz(t, map[string]string{"tarot-agent": "from-release"})
	name, err := AssetName("linux", "amd64", "v0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive)
	good := hex.EncodeToString(sum[:]) + "  " + name + "\n"

	t.Run("release replaces the running binary and skips go", func(t *testing.T) {
		opt := assetOptions(t, archive, good, name, "/usr/local/bin/tarot-agent", func(string, string, []byte) (string, error) {
			return "", nil
		})
		opt.Current = "v0.5.0"
		opt.LookGo = func() (string, error) {
			t.Fatal("release build should not look for go when replace works")
			return "", nil
		}
		if err := Execute(context.Background(), opt); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("release falls back to go install after replace fails", func(t *testing.T) {
		var order []string
		var stderr strings.Builder
		opt := assetOptions(t, archive, good, name, "/usr/local/bin/tarot-agent", func(string, string, []byte) (string, error) {
			order = append(order, "place")
			return "", os.ErrPermission
		})
		opt.Current = "0.5.0"
		opt.Stderr = &stderr
		opt.LookGo = func() (string, error) { return "/usr/bin/go", nil }
		opt.GoInstall = func(context.Context, string) error {
			order = append(order, "go")
			return nil
		}
		opt.ReportGo = func(context.Context) (string, string) {
			return "/home/user/go/bin/tarot-agent", "tarot-agent dev"
		}
		if err := Execute(context.Background(), opt); err != nil {
			t.Fatal(err)
		}
		if strings.Join(order, ",") != "place,go" {
			t.Fatalf("order %v", order)
		}
		if !strings.Contains(stderr.String(), "替换当前程序失败，改用 go install") {
			t.Fatalf("stderr %s", stderr.String())
		}
	})

	for _, current := range []string{"dev", "", "nightly"} {
		t.Run("nonrelease "+current+" prefers go install", func(t *testing.T) {
			var order []string
			opt := assetOptions(t, archive, good, name, "/tmp/tarot-agent", func(string, string, []byte) (string, error) {
				order = append(order, "place")
				return "", nil
			})
			opt.Current = current
			opt.LookGo = func() (string, error) { return "/usr/bin/go", nil }
			opt.GoInstall = func(context.Context, string) error {
				order = append(order, "go")
				return nil
			}
			opt.ReportGo = func(context.Context) (string, string) { return "/home/user/go/bin/tarot-agent", "tarot-agent dev" }
			if err := Execute(context.Background(), opt); err != nil {
				t.Fatal(err)
			}
			if strings.Join(order, ",") != "go" {
				t.Fatalf("current %q order %v", current, order)
			}
		})
	}

	t.Run("dev falls back to the release asset", func(t *testing.T) {
		var order []string
		var stderr strings.Builder
		opt := assetOptions(t, archive, good, name, "/tmp/tarot-agent", func(_, _ string, data []byte) (string, error) {
			order = append(order, "place")
			if string(data) != "from-release" {
				t.Fatalf("placed %q", data)
			}
			return "", nil
		})
		opt.Current = "dev"
		opt.Stderr = &stderr
		opt.LookGo = func() (string, error) { return "/usr/bin/go", nil }
		opt.GoInstall = func(context.Context, string) error {
			order = append(order, "go")
			return errors.New("module lookup failed")
		}
		if err := Execute(context.Background(), opt); err != nil {
			t.Fatal(err)
		}
		if strings.Join(order, ",") != "go,place" {
			t.Fatalf("order %v", order)
		}
		if !strings.Contains(stderr.String(), "改用预编译包") {
			t.Fatalf("stderr %s", stderr.String())
		}
	})
}

func TestExecuteCheckOnlyDoesNotInstall(t *testing.T) {
	var out strings.Builder
	err := Execute(context.Background(), ExecuteOptions{
		Current:   "0.5.0",
		CheckOnly: true,
		Yes:       true,
		Stdout:    &out,
		Stderr:    io.Discard,
		Client:    releaseClient(t, "v0.6.0", nil),
		LookGo:    func() (string, error) { t.Fatal("check only"); return "", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "有更新") || !strings.Contains(out.String(), "tarot-agent update") {
		t.Fatalf("out %s", out.String())
	}
}

func TestDownloadUsesHTTPTest(t *testing.T) {
	var sawAuth, sawAgent bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") != ""
		sawAgent = r.Header.Get("User-Agent") != ""
		w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	body, err := download(context.Background(), srv.Client(), srv.URL+"/pkg")
	if err != nil || string(body) != "ok" || sawAuth || !sawAgent {
		t.Fatalf("body %q err %v auth %v agent %v", body, err, sawAuth, sawAgent)
	}
}

func releaseClient(t *testing.T, tag string, assets []Asset) *http.Client {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"tag_name":"` + tag + `","html_url":"https://example.test/r","assets":[`)
	for i, a := range assets {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"name":"` + a.Name + `","browser_download_url":"` + a.URL + `"}`)
	}
	b.WriteString(`]}`)
	payload := b.String()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != LatestURL {
			return jsonResponse(404, "not the release api"), nil
		}
		return jsonResponse(200, payload), nil
	})}
}

func assetOptions(t *testing.T, archive []byte, checksum, name, dest string, place func(string, string, []byte) (string, error)) ExecuteOptions {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+name):
			w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			io.WriteString(w, checksum)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	api := releaseClient(t, "v0.6.0", []Asset{
		{Name: name, URL: srv.URL + "/" + name},
		{Name: checksumAsset, URL: srv.URL + "/checksums.txt"},
	})
	base := api.Transport
	next := srv.Client().Transport
	if next == nil {
		next = http.DefaultTransport
	}
	api.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == LatestURL {
			return base.RoundTrip(r)
		}
		return next.RoundTrip(r)
	})
	var out strings.Builder
	return ExecuteOptions{
		Current:    "0.5.0",
		Yes:        true,
		Stdout:     &out,
		Stderr:     io.Discard,
		Client:     api,
		GOOS:       "linux",
		GOARCH:     "amd64",
		LookGo:     func() (string, error) { return "", errors.New("no go") },
		Executable: func() (string, error) { return dest, nil },
		Place:      place,
	}
}

func runAssetUpdate(t *testing.T, archive []byte, checksum, name, dest string, place func(string, string, []byte) (string, error)) (ExecuteOptions, string) {
	t.Helper()
	opt := assetOptions(t, archive, checksum, name, dest, place)
	if err := Execute(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	return opt, opt.Stdout.(*strings.Builder).String()
}

func runAssetUpdateErr(t *testing.T, archive []byte, checksum, name, dest string, place func(string, string, []byte) (string, error)) (ExecuteOptions, string) {
	t.Helper()
	opt := assetOptions(t, archive, checksum, name, dest, place)
	err := Execute(context.Background(), opt)
	if err == nil {
		t.Fatal("expected error")
	}
	return opt, err.Error()
}
