package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// welcomeFile remembers that the first-run product guide was dismissed.
// It is separate from config.json so an old config file can stay untouched.
const welcomeFile = "welcome.seen"

// WelcomeSeen reports whether the guide has already been dismissed.
// A missing file, including an old install that has never had one, is not seen.
// Read problems are treated as not seen so startup can continue.
func WelcomeSeen() bool {
	path, err := welcomePath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return true
}

// MarkWelcomeSeen writes the marker. The file is 0600. A new config directory
// is 0700; an existing directory keeps its mode.
func MarkWelcomeSeen() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查目录 %s: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("创建目录 %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("设置目录权限 %s: %w", dir, err)
		}
	}
	path := filepath.Join(dir, welcomeFile)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	_, writeErr := f.Write([]byte("seen\n"))
	chmodErr := f.Chmod(0o600)
	closeErr := f.Close()
	return errors.Join(writeErr, chmodErr, closeErr)
}

func welcomePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, welcomeFile), nil
}
