package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
)

// ReadingStore persists tarot reading records in JSONL format.
type ReadingStore struct {
	mu   sync.Mutex
	path string
}

// NewReadingStore creates a ReadingStore that writes to the given JSONL file path.
// The parent directory is created on first write if it does not exist.
func NewReadingStore(path string) *ReadingStore {
	return &ReadingStore{path: path}
}

// DefaultReadingsPath returns the default path for the readings JSONL file.
func DefaultReadingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".tarot-agent", "data", "readings.jsonl"), nil
}

// Append writes a reading record as a single JSON line to the JSONL file.
func (rs *ReadingStore) Append(reading *domain.Reading) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	dir := filepath.Dir(rs.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create readings directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(rs.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open readings file %s: %w", rs.path, err)
	}
	defer f.Close()

	data, err := json.Marshal(reading)
	if err != nil {
		return fmt.Errorf("marshal reading: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write reading: %w", err)
	}

	return nil
}

// List returns the most recent N readings from the JSONL file.
// If limit <= 0, all readings are returned.
func (rs *ReadingStore) List(limit int) ([]domain.Reading, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	f, err := os.Open(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open readings file %s: %w", rs.path, err)
	}
	defer f.Close()

	var all []domain.Reading
	scanner := bufio.NewScanner(f)
	// A full reading (interpretation plus ten cards) can exceed the default 64KiB token limit.
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r domain.Reading
		if err := json.Unmarshal(line, &r); err != nil {
			// Skip malformed lines rather than failing the entire read.
			continue
		}
		all = append(all, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan readings file: %w", err)
	}

	if limit <= 0 || limit >= len(all) {
		return all, nil
	}

	// Return the last N entries (most recent).
	return all[len(all)-limit:], nil
}

// UpdateFollowUps replaces the follow-up turns of the reading with id.
// Other fields on that line are kept. Lines that do not parse are copied
// through unchanged. The file mode stays 0600.
func (rs *ReadingStore) UpdateFollowUps(id string, turns []domain.FollowUpTurn) error {
	if id == "" {
		return fmt.Errorf("update reading: empty id")
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	data, err := os.ReadFile(rs.path)
	if err != nil {
		return fmt.Errorf("open readings file %s: %w", rs.path, err)
	}

	lines := bytes.Split(data, []byte("\n"))
	// Split keeps a trailing empty piece when the file ends with a newline.
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	var buf bytes.Buffer
	found := false
	for _, line := range lines {
		replaced := false
		if len(bytes.TrimSpace(line)) > 0 {
			var reading domain.Reading
			if json.Unmarshal(line, &reading) == nil && reading.ID == id {
				reading.FollowUps = cloneFollowUps(turns)
				encoded, err := json.Marshal(&reading)
				if err != nil {
					return fmt.Errorf("marshal reading: %w", err)
				}
				buf.Write(encoded)
				replaced = true
				found = true
			}
		}
		if !replaced {
			buf.Write(line)
		}
		buf.WriteByte('\n')
	}
	if !found {
		return fmt.Errorf("update reading %s: not found", id)
	}

	dir := filepath.Dir(rs.path)
	tmp, err := os.CreateTemp(dir, ".readings-*.tmp")
	if err != nil {
		return fmt.Errorf("create readings temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod readings temp file: %w", err)
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write readings temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close readings temp file: %w", err)
	}
	if err := os.Rename(tmpName, rs.path); err != nil {
		return fmt.Errorf("replace readings file %s: %w", rs.path, err)
	}
	cleanup = false
	return nil
}

func cloneFollowUps(turns []domain.FollowUpTurn) []domain.FollowUpTurn {
	if len(turns) == 0 {
		return nil
	}
	out := make([]domain.FollowUpTurn, len(turns))
	copy(out, turns)
	return out
}
