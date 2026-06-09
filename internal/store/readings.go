package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/voocel/tarot-agent/internal/domain"
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
