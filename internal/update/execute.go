package update

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrAvailable means a newer release exists and this run did not install it
// because there is no terminal and -y was not set. Callers should exit 3.
var ErrAvailable = errors.New("有更新可用")

// ExecuteOptions configures one `tarot-agent update` run.
// Nil hooks use the real go tool, the current executable, and PlaceBinary.
type ExecuteOptions struct {
	Current    string
	CheckOnly  bool
	Yes        bool
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	IsTerminal bool
	Client     *http.Client
	GOOS       string
	GOARCH     string

	LookGo     func() (string, error)
	GoInstall  func(ctx context.Context, moduleAt string) error
	ReportGo   func(ctx context.Context) (path, version string)
	Executable func() (string, error)
	Place      func(goos, dest string, data []byte) (string, error)
}

// Execute checks the latest release and, unless CheckOnly is set, installs it.
// TAROT_AGENT_NO_UPDATE does not apply here; callers must not consult it.
func Execute(ctx context.Context, opt ExecuteOptions) error {
	opt = opt.withDefaults()
	if opt.CheckOnly {
		res, err := Check(ctx, opt.Client, opt.Current)
		if err != nil {
			return err
		}
		fmt.Fprint(opt.Stdout, FormatReport(res))
		return nil
	}

	rel, err := FetchLatest(ctx, opt.Client)
	if err != nil {
		return err
	}
	if _, ok := ParseSemver(rel.Tag); !ok {
		fmt.Fprint(opt.Stdout, FormatReport(&Result{Current: opt.Current, Latest: rel.Tag, URL: rel.URL, Unparsed: true}))
		return nil
	}
	if _, ok := ParseSemver(opt.Current); ok && !IsNewer(rel.Tag, opt.Current) {
		fmt.Fprint(opt.Stdout, FormatReport(&Result{Current: opt.Current, Latest: rel.Tag, URL: rel.URL}))
		return nil
	}

	writeOffer(opt.Stdout, opt.Current, rel)
	if !opt.Yes && !opt.IsTerminal {
		fmt.Fprint(opt.Stdout, "\n当前不是交互终端，没有安装。要更新请运行 tarot-agent update -y\n")
		fmt.Fprint(opt.Stdout, InstallGuide(rel.URL))
		return ErrAvailable
	}
	if !opt.Yes {
		ok, err := confirm(opt.Stdin, opt.Stdout, rel.Tag)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(opt.Stdout, "已取消。")
			return nil
		}
	}
	return install(ctx, opt, rel)
}

func (opt ExecuteOptions) withDefaults() ExecuteOptions {
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Stderr == nil {
		opt.Stderr = os.Stderr
	}
	if opt.Stdin == nil {
		opt.Stdin = os.Stdin
	}
	if opt.Client == nil {
		opt.Client = &http.Client{Timeout: UpdateTimeout}
	}
	if opt.GOOS == "" {
		opt.GOOS = runtime.GOOS
	}
	if opt.GOARCH == "" {
		opt.GOARCH = runtime.GOARCH
	}
	if opt.LookGo == nil {
		opt.LookGo = func() (string, error) { return exec.LookPath("go") }
	}
	if opt.GoInstall == nil {
		opt.GoInstall = defaultGoInstall
	}
	if opt.ReportGo == nil {
		opt.ReportGo = defaultReportGo
	}
	if opt.Executable == nil {
		opt.Executable = defaultExecutable
	}
	if opt.Place == nil {
		opt.Place = PlaceBinary
	}
	return opt
}

func writeOffer(w io.Writer, current string, rel Release) {
	cur := strings.TrimSpace(current)
	if cur == "" {
		cur = "（空）"
	}
	fmt.Fprintf(w, "当前版本：%s\n最新版本：%s\n", cur, rel.Tag)
	if _, ok := ParseSemver(current); ok {
		fmt.Fprintf(w, "%s → %s\n", cur, rel.Tag)
	} else {
		fmt.Fprintf(w, "开发版本将安装最新发布 %s。\n", rel.Tag)
	}
	if rel.URL != "" {
		fmt.Fprintf(w, "发布页：%s\n", rel.URL)
	}
}

func confirm(in io.Reader, out io.Writer, tag string) (bool, error) {
	fmt.Fprintf(out, "要现在更新到 %s 吗？[y/N] ", tag)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
}

// errNoGo means the go tool is not on PATH. It is not a failed install.
var errNoGo = errors.New("没有 go")

// install picks a path from the build the user is actually running.
// A release version replaces that executable. dev and other non-versions
// were installed with go, so they update with go install first.
func install(ctx context.Context, opt ExecuteOptions, rel Release) error {
	if _, ok := ParseSemver(opt.Current); ok {
		err := installAsset(ctx, opt, rel)
		if err == nil {
			return nil
		}
		fmt.Fprintf(opt.Stderr, "替换当前程序失败，改用 go install：%s\n", firstLine(err))
		if gerr := tryGoInstall(ctx, opt, rel); gerr == nil {
			return nil
		} else if !errors.Is(gerr, errNoGo) {
			return fmt.Errorf("%w\ngo install 也失败了：%v", err, gerr)
		}
		return err
	}

	if err := tryGoInstall(ctx, opt, rel); err == nil {
		return nil
	} else if !errors.Is(err, errNoGo) {
		fmt.Fprintf(opt.Stderr, "go install 失败，改用预编译包：%v\n", err)
	}
	return installAsset(ctx, opt, rel)
}

func firstLine(err error) string {
	s := err.Error()
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func tryGoInstall(ctx context.Context, opt ExecuteOptions, rel Release) error {
	path, err := opt.LookGo()
	if err != nil || path == "" {
		return errNoGo
	}
	spec := InstallPath + "@" + rel.Tag
	if err := opt.GoInstall(ctx, spec); err != nil {
		return err
	}
	which, ver := opt.ReportGo(ctx)
	fmt.Fprintf(opt.Stdout, "已执行 go install %s\n", spec)
	if which == "" {
		fmt.Fprintln(opt.Stdout, "which: 没有在 PATH 里找到 tarot-agent")
	} else {
		fmt.Fprintf(opt.Stdout, "which: %s\n", which)
	}
	if ver != "" {
		fmt.Fprintf(opt.Stdout, "tarot-agent --version: %s\n", ver)
	}
	fmt.Fprintf(opt.Stdout, "go install 没有打进发布版本号时，--version 仍可能显示 dev，但代码已是 %s。\n", rel.Tag)
	return nil
}

func installAsset(ctx context.Context, opt ExecuteOptions, rel Release) error {
	name, err := AssetName(opt.GOOS, opt.GOARCH, rel.Tag)
	page := rel.URL
	if err != nil {
		if page != "" {
			return fmt.Errorf("%w\n发布页：%s", err, page)
		}
		return err
	}
	asset, ok := findAsset(rel.Assets, name)
	if !ok || asset.URL == "" {
		return fmt.Errorf("发布里没有 %s\n发布页：%s", name, page)
	}
	checksumURL := ""
	if sum, ok := findAsset(rel.Assets, checksumAsset); ok {
		checksumURL = sum.URL
	}
	dest, err := opt.Executable()
	if err != nil {
		return fmt.Errorf("找不到当前程序：%w\n%s", err, ManualCommands(name, asset.URL, checksumURL, "当前可执行文件", opt.GOOS))
	}

	body, err := download(ctx, opt.Client, asset.URL)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, ManualCommands(name, asset.URL, checksumURL, dest, opt.GOOS))
	}
	if checksumURL == "" {
		fmt.Fprintln(opt.Stderr, "发布包没有 checksums.txt，已跳过校验。")
	} else {
		sumBody, err := download(ctx, opt.Client, checksumURL)
		if err != nil {
			return fmt.Errorf("下载 checksums.txt 失败：%w", err)
		}
		if err := VerifyChecksum(string(sumBody), name, body); err != nil {
			return fmt.Errorf("%w\n发布页：%s", err, page)
		}
	}
	bin, err := ExtractBinary(name, body, opt.GOOS)
	if err != nil {
		return err
	}
	note, err := opt.Place(opt.GOOS, dest, bin)
	if err != nil {
		return fmt.Errorf("无法写入 %s：%w\n%s", dest, err, ManualCommands(name, asset.URL, checksumURL, dest, opt.GOOS))
	}
	if note != "" {
		fmt.Fprint(opt.Stdout, note)
		return nil
	}
	fmt.Fprintf(opt.Stdout, "已替换 %s\n", dest)
	return nil
}

func download(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, fmt.Errorf("不支持的下载地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tarot-agent")
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败：%d", resp.StatusCode)
	}
	if len(body) > maxDownload {
		return nil, fmt.Errorf("下载内容超过限制")
	}
	return body, nil
}

func defaultGoInstall(ctx context.Context, moduleAt string) error {
	cmd := exec.CommandContext(ctx, "go", "install", moduleAt)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultReportGo(ctx context.Context) (string, string) {
	which, _ := exec.LookPath("tarot-agent")
	installed := installedByGo(ctx)
	path := installed
	if path == "" {
		path = which
	}
	if path == "" {
		return "", ""
	}
	if which != "" && installed != "" && which != installed {
		path = which + "（新文件在 " + installed + "）"
	}
	versionPath := installed
	if versionPath == "" {
		versionPath = which
	}
	out, err := exec.CommandContext(ctx, versionPath, "--version").Output()
	if err != nil {
		return path, strings.TrimSpace(err.Error())
	}
	return path, strings.TrimSpace(string(out))
}

// installedByGo is the binary go install just wrote, which may not be first on PATH.
func installedByGo(ctx context.Context) string {
	dir := strings.TrimSpace(commandOutput(ctx, "go", "env", "GOBIN"))
	if dir == "" {
		gopath := strings.TrimSpace(commandOutput(ctx, "go", "env", "GOPATH"))
		if gopath == "" {
			return ""
		}
		dir = filepath.Join(strings.Split(gopath, string(os.PathListSeparator))[0], "bin")
	}
	candidate := filepath.Join(dir, binaryName(runtime.GOOS))
	st, err := os.Stat(candidate)
	if err != nil || st.IsDir() {
		return ""
	}
	return candidate
}

func commandOutput(ctx context.Context, name string, args ...string) string {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func defaultExecutable() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	return p, nil
}
