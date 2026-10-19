package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlaceBinary writes data into place.
// On Windows the running executable cannot be replaced, so the bytes go to dest+".new".
// The returned note is empty when dest itself was replaced.
func PlaceBinary(goos, dest string, data []byte) (string, error) {
	if strings.TrimSpace(dest) == "" {
		return "", fmt.Errorf("没有可执行文件路径")
	}
	if goos == "windows" {
		side := dest + ".new"
		if err := os.WriteFile(side, data, 0o755); err != nil {
			return "", err
		}
		return fmt.Sprintf("Windows 不能覆盖正在运行的程序。新文件已写到 %s，退出后用它替换 %s。\n", side, dest), nil
	}
	if err := replaceFile(dest, data); err != nil {
		return "", err
	}
	return "", nil
}

func replaceFile(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	f, err := os.CreateTemp(dir, ".tarot-agent-new-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0o755); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
