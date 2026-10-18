package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// maxLogBytes is the size at which tarot.log rotates to tarot.log.1.
// One extra record may land past the cap; the next write rotates.
const maxLogBytes int64 = 1 << 20

// InitLogger sends structured logs to logFile.
// If the path is empty or the file cannot be opened, logs are discarded.
// They are not written to stderr, which the TUI uses for the screen.
func InitLogger(level string, logFile string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	w := io.Writer(io.Discard)
	if logFile != "" {
		if f, err := openRotatingLog(logFile, maxLogBytes); err == nil {
			w = f
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: l})))
}

// rotatingFile appends to path and keeps a single backup at path+".1".
type rotatingFile struct {
	mu   sync.Mutex
	path string
	max  int64
	f    *os.File
	size int64
}

func openRotatingLog(path string, max int64) (*rotatingFile, error) {
	if path == "" {
		return nil, fmt.Errorf("empty log path")
	}
	if max <= 0 {
		return nil, fmt.Errorf("log size limit must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	rf := &rotatingFile{path: path, max: max}
	if err := rf.open(); err != nil {
		return nil, err
	}
	if rf.size >= rf.max {
		if err := rf.rotate(); err != nil && rf.f == nil {
			return nil, err
		}
	}
	return rf, nil
}

func (r *rotatingFile) open() error {
	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	r.f = f
	r.size = info.Size()
	return nil
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, fmt.Errorf("log file %s is closed", r.path)
	}
	if r.size > 0 && r.size+int64(len(p)) > r.max {
		if err := r.rotate(); err != nil && r.f == nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// rotate keeps writing to the current file when the backup cannot be created.
func (r *rotatingFile) rotate() error {
	if r.f != nil {
		if err := r.f.Close(); err != nil {
			return err
		}
		r.f = nil
		r.size = 0
	}
	backup := r.path + ".1"
	_ = os.Remove(backup)
	if err := os.Rename(r.path, backup); err != nil && !os.IsNotExist(err) {
		if openErr := r.open(); openErr != nil {
			return fmt.Errorf("reopen log after failed rotate: %w", openErr)
		}
		return nil
	}
	return r.open()
}
