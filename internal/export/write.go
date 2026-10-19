package export

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
	"github.com/MaybeJustLikeThis/tarot-agent/internal/store"
)

// ErrExists is returned when the destination file is already there and
// force is false. The file is left unchanged.
var ErrExists = errors.New("文件已存在")

// ErrNotFound is returned when the requested reading id is not in the file.
var ErrNotFound = errors.New("找不到占卜记录")

// ErrEmpty is returned when there is nothing to export.
var ErrEmpty = errors.New("还没有可导出的占卜记录")

// Options selects one reading and where its text goes.
// Query empty or "latest" is the newest line. OutputPath empty writes Stdout.
type Options struct {
	Readings   *store.ReadingStore
	Query      string
	Format     string
	OutputPath string
	Force      bool
	Stdout     io.Writer
}

// Execute loads one reading and writes Markdown or plain text.
func Execute(opt Options) error {
	if opt.Readings == nil {
		return fmt.Errorf("没有占卜记录文件")
	}
	readings, err := opt.Readings.List(0)
	if err != nil {
		return fmt.Errorf("读取占卜记录: %w", err)
	}
	reading, err := Find(readings, opt.Query)
	if err != nil {
		return err
	}
	format := opt.Format
	if strings.TrimSpace(format) == "" {
		format = FormatMarkdown
	}
	text, err := Render(reading, format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opt.OutputPath) == "" {
		out := opt.Stdout
		if out == nil {
			out = os.Stdout
		}
		_, err := io.WriteString(out, text)
		return err
	}
	if err := WriteFile(opt.OutputPath, []byte(text), opt.Force); err != nil {
		if errors.Is(err, ErrExists) {
			return fmt.Errorf("%w：%s（加 --force 覆盖）", ErrExists, opt.OutputPath)
		}
		return err
	}
	return nil
}

// Find returns the newest reading when query is empty or "latest".
func Find(readings []domain.Reading, query string) (domain.Reading, error) {
	q := strings.TrimSpace(query)
	if len(readings) == 0 {
		return domain.Reading{}, ErrEmpty
	}
	if q == "" || strings.EqualFold(q, "latest") {
		return readings[len(readings)-1], nil
	}
	for i := len(readings) - 1; i >= 0; i-- {
		if readings[i].ID == q {
			return readings[i], nil
		}
	}
	return domain.Reading{}, fmt.Errorf("%w %s", ErrNotFound, q)
}

// DefaultExportPath is ~/.tarot-agent/exports/<id>.md.
func DefaultExportPath(id string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".tarot-agent", "exports", SafeID(id)+".md"), nil
}

// SaveMarkdown writes the reading to the default export path, replacing a
// previous export of the same id.
func SaveMarkdown(r domain.Reading) (string, error) {
	if strings.TrimSpace(r.ID) == "" {
		return "", fmt.Errorf("这条记录没有 ID，不能导出")
	}
	path, err := DefaultExportPath(r.ID)
	if err != nil {
		return "", err
	}
	text, err := Render(r, FormatMarkdown)
	if err != nil {
		return "", err
	}
	if err := WriteFile(path, []byte(text), true); err != nil {
		return "", err
	}
	return path, nil
}

// SafeID keeps a reading id inside a single filename.
func SafeID(id string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	s := strings.Trim(b.String(), "._")
	if s == "" {
		return "reading"
	}
	return s
}

// WriteFile creates or replaces path. New files and directories use 0600 and
// 0700. Without force, an existing file returns ErrExists and is not changed.
func WriteFile(path string, data []byte, force bool) error {
	if !force {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("%w", ErrExists)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("检查 %s: %w", path, err)
		}
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查目录 %s: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("创建目录 %s: %w", dir, err)
		}
		// Only the directory this call created. An existing parent keeps its mode.
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("设置目录权限 %s: %w", dir, err)
		}
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		if !force && os.IsExist(err) {
			return fmt.Errorf("%w", ErrExists)
		}
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("设置文件权限 %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("关闭 %s: %w", path, err)
	}
	return nil
}
