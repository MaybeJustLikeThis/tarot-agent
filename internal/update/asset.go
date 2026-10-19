package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const checksumAsset = "checksums.txt"

// AssetName is the goreleaser archive for this OS and arch.
// Tag may include a leading v. Windows archives are zip; the others are tar.gz.
func AssetName(goos, goarch, tag string) (string, error) {
	ver, ok := ParseSemver(tag)
	if !ok {
		return "", fmt.Errorf("无法从 %q 得到发布包名", tag)
	}
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", fmt.Errorf("不支持的系统 %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("不支持的架构 %s", goarch)
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	version := fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
	return fmt.Sprintf("tarot-agent_%s_%s_%s%s", version, goos, goarch, ext), nil
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "tarot-agent.exe"
	}
	return "tarot-agent"
}

// VerifyChecksum matches filename to a goreleaser checksums.txt line: "<sha256>  <name>".
func VerifyChecksum(checksums, filename string, data []byte) error {
	want, ok := checksumFor(checksums, filename)
	if !ok {
		return fmt.Errorf("checksums.txt 里没有 %s", filename)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("校验失败：%s 的 sha256 与 checksums.txt 不一致", filename)
	}
	return nil
}

func checksumFor(checksums, filename string) (string, bool) {
	for _, line := range strings.Split(checksums, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[len(fields)-1] == filename {
			return fields[0], true
		}
	}
	return "", false
}

func findAsset(assets []Asset, name string) (Asset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// ExtractBinary returns the tarot-agent binary inside a release archive.
func ExtractBinary(archiveName string, data []byte, goos string) ([]byte, error) {
	want := binaryName(goos)
	switch {
	case strings.HasSuffix(archiveName, ".zip"):
		return extractZip(data, want)
	case strings.HasSuffix(archiveName, ".tar.gz"), strings.HasSuffix(archiveName, ".tgz"):
		return extractTarGz(data, want)
	default:
		return nil, fmt.Errorf("无法解压 %s", archiveName)
	}
}

func extractTarGz(data []byte, want string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("无法解压发布包")
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var found []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("无法读取发布包")
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		if !isBinaryPath(hdr.Name, want) {
			if _, err := io.Copy(io.Discard, io.LimitReader(tr, hdr.Size)); err != nil {
				return nil, err
			}
			continue
		}
		if hdr.Size < 0 || hdr.Size > maxDownload {
			return nil, fmt.Errorf("压缩包里的程序过大")
		}
		body, err := io.ReadAll(io.LimitReader(tr, hdr.Size))
		if err != nil {
			return nil, err
		}
		found = body
	}
	if found == nil {
		return nil, fmt.Errorf("压缩包里没有 %s", want)
	}
	return found, nil
}

func extractZip(data []byte, want string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("无法解压发布包")
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !isBinaryPath(f.Name, want) {
			continue
		}
		if f.UncompressedSize64 > maxDownload {
			return nil, fmt.Errorf("压缩包里的程序过大")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, int64(f.UncompressedSize64)))
		closeErr := rc.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		return body, nil
	}
	return nil, fmt.Errorf("压缩包里没有 %s", want)
}

func isBinaryPath(name, want string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	name = strings.TrimPrefix(name, "./")
	base := filepath.Base(filepath.ToSlash(name))
	return base == want
}

// ManualCommands is the curl, extract, and move sequence printed when the
// program cannot replace itself.
func ManualCommands(archiveName, archiveURL, checksumURL, dest, goos string) string {
	var b strings.Builder
	b.WriteString("可以手动执行：\n")
	fmt.Fprintf(&b, "curl -fL -o %s %q\n", archiveName, archiveURL)
	if checksumURL != "" {
		fmt.Fprintf(&b, "curl -fL -o checksums.txt %q\n", checksumURL)
		b.WriteString("对照 checksums.txt 里该文件的 sha256\n")
	}
	if goos == "windows" {
		fmt.Fprintf(&b, "tar -xf %s\n", archiveName)
		fmt.Fprintf(&b, "退出正在运行的程序后，把 tarot-agent.exe 移到 %q\n", dest)
		fmt.Fprintf(&b, "如果无法覆盖，先移到 %q\n", dest+".new")
		return b.String()
	}
	fmt.Fprintf(&b, "tar -xzf %s\n", archiveName)
	b.WriteString("chmod +x tarot-agent\n")
	fmt.Fprintf(&b, "mv tarot-agent %q\n", dest)
	return b.String()
}
