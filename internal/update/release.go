package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// LatestURL is the public GitHub Releases endpoint. It does not use a token.
	LatestURL = "https://api.github.com/repos/MaybeJustLikeThis/tarot-agent/releases/latest"
	// InstallPath is the module path passed to go install.
	InstallPath = "github.com/MaybeJustLikeThis/tarot-agent/cmd/tarot-agent"
	// CheckTimeout is the startup and check-only budget.
	CheckTimeout = 3 * time.Second
	// UpdateTimeout bounds go install and release downloads.
	UpdateTimeout = 2 * time.Minute
	maxDownload   = 64 << 20
)

// Asset is one file attached to a GitHub release.
type Asset struct {
	Name string
	URL  string
}

// Release is the latest published GitHub release.
type Release struct {
	Tag    string
	URL    string
	Assets []Asset
}

// Result is a completed check. Skipped means the local version is not a release.
// Unparsed means the remote tag is not major.minor.patch.
type Result struct {
	Current  string
	Latest   string
	URL      string
	Newer    bool
	Skipped  bool
	Unparsed bool
}

// Check compares current with the latest GitHub release.
// A dev or empty current version does not call the network.
// An unparseable remote tag is returned without an error.
func Check(ctx context.Context, client *http.Client, current string) (*Result, error) {
	if _, ok := ParseSemver(current); !ok {
		return &Result{Current: current, Skipped: true}, nil
	}
	rel, err := FetchLatest(ctx, client)
	if err != nil {
		return nil, err
	}
	res := &Result{Current: current, Latest: rel.Tag, URL: rel.URL}
	if _, ok := ParseSemver(rel.Tag); !ok {
		res.Unparsed = true
		return res, nil
	}
	res.Newer = IsNewer(rel.Tag, current)
	return res, nil
}

// FetchLatest reads tag_name and html_url from the public releases API.
func FetchLatest(ctx context.Context, client *http.Client) (Release, error) {
	if client == nil {
		client = &http.Client{Timeout: CheckTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "tarot-agent")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Release{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub 返回 %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Release{}, fmt.Errorf("解析发布信息失败")
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return Release{}, fmt.Errorf("响应里没有 tag_name")
	}
	rel := Release{Tag: strings.TrimSpace(payload.TagName), URL: strings.TrimSpace(payload.HTMLURL)}
	for _, a := range payload.Assets {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			continue
		}
		rel.Assets = append(rel.Assets, Asset{Name: name, URL: strings.TrimSpace(a.URL)})
	}
	return rel, nil
}

// FormatReport is the --check-update text. It does not include a trailing blank nag.
func FormatReport(r *Result) string {
	if r == nil {
		return "检查更新失败：没有结果\n"
	}
	cur := strings.TrimSpace(r.Current)
	if cur == "" {
		cur = "（空）"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "当前版本：%s\n", cur)
	if r.Skipped {
		b.WriteString("开发版本不与发布版比较，不做更新提示。\n")
		return b.String()
	}
	fmt.Fprintf(&b, "最新版本：%s\n", r.Latest)
	switch {
	case r.Unparsed:
		b.WriteString("远端版本无法解析，已跳过。\n")
	case r.Newer:
		b.WriteString("有更新。\n\n")
		b.WriteString(InstallGuide(r.URL))
	default:
		b.WriteString("已是最新版本。\n")
	}
	return b.String()
}

// InstallGuide is the manual install text. The program does not replace itself here.
func InstallGuide(url string) string {
	var b strings.Builder
	b.WriteString("运行 tarot-agent update 更新。\n")
	fmt.Fprintf(&b, "或：go install %s@latest\n", InstallPath)
	if strings.TrimSpace(url) != "" {
		fmt.Fprintf(&b, "或下载预编译包：%s\n", strings.TrimSpace(url))
	} else {
		b.WriteString("或到 GitHub Releases 下载预编译包。\n")
	}
	return b.String()
}

// ShortNotice is the one-time TUI hint. It stays on two short lines.
// The TUI never downloads or replaces the binary.
func ShortNotice(current, latest string) string {
	cur := strings.TrimSpace(current)
	if cur == "" {
		cur = "（空）"
	}
	return fmt.Sprintf("%s → %s 可更新\n运行 tarot-agent update 更新", cur, strings.TrimSpace(latest))
}
