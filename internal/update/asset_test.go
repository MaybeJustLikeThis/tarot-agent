package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch, tag, want string
	}{
		{"linux", "amd64", "v0.5.0", "tarot-agent_0.5.0_linux_amd64.tar.gz"},
		{"linux", "arm64", "0.5.0", "tarot-agent_0.5.0_linux_arm64.tar.gz"},
		{"darwin", "amd64", "v0.6.0", "tarot-agent_0.6.0_darwin_amd64.tar.gz"},
		{"darwin", "arm64", "V0.6.0", "tarot-agent_0.6.0_darwin_arm64.tar.gz"},
		{"windows", "amd64", "v0.5.0", "tarot-agent_0.5.0_windows_amd64.zip"},
		{"windows", "arm64", "v0.5.0", "tarot-agent_0.5.0_windows_arm64.zip"},
	}
	for _, tc := range cases {
		got, err := AssetName(tc.goos, tc.goarch, tc.tag)
		if err != nil || got != tc.want {
			t.Errorf("AssetName(%s, %s, %s) = %q, %v; want %s", tc.goos, tc.goarch, tc.tag, got, err, tc.want)
		}
	}
	for _, tc := range []struct{ goos, goarch, tag string }{
		{"linux", "386", "v0.5.0"},
		{"freebsd", "amd64", "v0.5.0"},
		{"linux", "amd64", "dev"},
		{"linux", "amd64", "v1.2"},
	} {
		if _, err := AssetName(tc.goos, tc.goarch, tc.tag); err == nil {
			t.Errorf("AssetName(%s, %s, %s) should fail", tc.goos, tc.goarch, tc.tag)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	body := []byte("tarot-bytes")
	sum := sha256.Sum256(body)
	name := "tarot-agent_0.5.0_linux_amd64.tar.gz"
	text := hex.EncodeToString(sum[:]) + "  " + name + "\r\n" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.tar.gz\n"
	if err := VerifyChecksum(text, name, body); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum(text, name, []byte("nope")); err == nil || !strings.Contains(err.Error(), "校验失败") {
		t.Fatalf("mismatch: %v", err)
	}
	if err := VerifyChecksum(text, "missing.tar.gz", body); err == nil || !strings.Contains(err.Error(), "没有") {
		t.Fatalf("missing: %v", err)
	}
}

func TestExtractBinary(t *testing.T) {
	payload := []byte("#!/tarot")
	gz := tarGz(t, map[string]string{
		"README.md":        "nope",
		"dist/tarot-agent": string(payload),
		"../tarot-agent":   "escape",
	})
	got, err := ExtractBinary("tarot-agent_0.6.0_linux_amd64.tar.gz", gz, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("extracted %q", got)
	}

	z := zipBytes(t, map[string]string{"tarot-agent.exe": "win"})
	got, err = ExtractBinary("tarot-agent_0.6.0_windows_amd64.zip", z, "windows")
	if err != nil || string(got) != "win" {
		t.Fatalf("zip %q err %v", got, err)
	}

	escape := tarGz(t, map[string]string{"../tarot-agent": "escape"})
	if _, err := ExtractBinary("x.tar.gz", escape, "linux"); err == nil {
		t.Fatal("path traversal should not yield a binary")
	}
}

func TestManualCommandsMentionCurlAndDest(t *testing.T) {
	text := ManualCommands(
		"tarot-agent_0.6.0_linux_amd64.tar.gz",
		"https://example.test/pkg.tar.gz",
		"https://example.test/checksums.txt",
		"/usr/local/bin/tarot-agent",
		"linux",
	)
	for _, want := range []string{"curl -fL", "tar -xzf", "chmod +x", "mv tarot-agent", "/usr/local/bin/tarot-agent", "checksums.txt"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
	win := ManualCommands("tarot-agent_0.6.0_windows_amd64.zip", "https://example.test/a.zip", "", `C:\tarot-agent.exe`, "windows")
	if !strings.Contains(win, "tar -xf") || !strings.Contains(win, ".new") {
		t.Fatalf("windows steps:\n%s", win)
	}
}

func TestPlaceBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "tarot-agent")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	note, err := PlaceBinary("linux", dest, []byte("new"))
	if err != nil || note != "" {
		t.Fatalf("note %q err %v", note, err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "new" {
		t.Fatalf("replaced %q err %v", got, err)
	}

	win := filepath.Join(dir, "tarot-agent.exe")
	if err := os.WriteFile(win, []byte("running"), 0o755); err != nil {
		t.Fatal(err)
	}
	note, err = PlaceBinary("windows", win, []byte("staged"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, ".new") || !strings.Contains(note, "不能覆盖") {
		t.Fatalf("note %q", note)
	}
	still, _ := os.ReadFile(win)
	side, err := os.ReadFile(win + ".new")
	if string(still) != "running" || string(side) != "staged" || err != nil {
		t.Fatalf("still %q side %q err %v", still, side, err)
	}

	if _, err := PlaceBinary("linux", "  ", []byte("x")); err == nil {
		t.Fatal("empty dest should fail")
	}
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
