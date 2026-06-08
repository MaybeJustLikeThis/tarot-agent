package store

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/voocel/tarot-agent/internal/domain"
)

//go:embed assets/cards/major_arcana.json
var majorArcanaJSON []byte

//go:embed assets/cards/minor_wands.json
var minorWandsJSON []byte

//go:embed assets/cards/minor_cups.json
var minorCupsJSON []byte

//go:embed assets/cards/minor_swords.json
var minorSwordsJSON []byte

//go:embed assets/cards/minor_pentacles.json
var minorPentaclesJSON []byte

//go:embed assets/spreads/spreads.json
var spreadsJSON []byte

//go:embed assets/knowledge/elements.json
var elementsJSON []byte

// MajorArcanaJSON returns the embedded major arcana JSON bytes.
func MajorArcanaJSON() []byte { return majorArcanaJSON }

// MinorWandsJSON returns the embedded minor wands JSON bytes.
func MinorWandsJSON() []byte { return minorWandsJSON }

// MinorCupsJSON returns the embedded minor cups JSON bytes.
func MinorCupsJSON() []byte { return minorCupsJSON }

// MinorSwordsJSON returns the embedded minor swords JSON bytes.
func MinorSwordsJSON() []byte { return minorSwordsJSON }

// MinorPentaclesJSON returns the embedded minor pentacles JSON bytes.
func MinorPentaclesJSON() []byte { return minorPentaclesJSON }

// SpreadsJSON returns the embedded spreads JSON bytes.
func SpreadsJSON() []byte { return spreadsJSON }

// ElementsJSON returns the embedded elements knowledge JSON bytes.
func ElementsJSON() []byte { return elementsJSON }

// IO provides atomic file read/write operations.
type IO struct {
	mu sync.RWMutex
}

// NewIO creates a new IO instance.
func NewIO() *IO {
	return &IO{}
}

// ReadJSON reads a JSON file into the given target.
func (io *IO) ReadJSON(path string, target any) error {
	io.mu.RLock()
	defer io.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal json from %s: %w", path, err)
	}
	return nil
}

// WriteJSON atomically writes data as JSON to a file using tmp+rename.
func (io *IO) WriteJSON(path string, data any) error {
	io.mu.Lock()
	defer io.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("write tmp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}
	return nil
}

// SuitInfo describes a tarot suit's element and domain.
type SuitInfo struct {
	Element string `json:"element"`
	Domain  string `json:"domain"`
}

// ElementKnowledge holds global tarot knowledge about elements, suits, and numerology.
type ElementKnowledge struct {
	Interactions map[string]string  `json:"interactions"`
	SuitMeanings map[string]SuitInfo `json:"suit_meanings"`
	Numerology   map[string]string  `json:"numerology"`
}

// NewElementKnowledge loads element knowledge from JSON data.
func NewElementKnowledge(jsonData []byte) (*ElementKnowledge, error) {
	var ek ElementKnowledge
	if err := json.Unmarshal(jsonData, &ek); err != nil {
		return nil, fmt.Errorf("load element knowledge: %w", err)
	}
	return &ek, nil
}

// CardStore provides read-only access to tarot card data.
type CardStore struct {
	mu    sync.RWMutex
	cards map[string]domain.Card
}

// NewCardStore creates a CardStore and loads cards from multiple JSON data sources.
func NewCardStore(jsonSources ...[]byte) (*CardStore, error) {
	cs := &CardStore{
		cards: make(map[string]domain.Card),
	}
	for _, data := range jsonSources {
		var cards []domain.Card
		if err := json.Unmarshal(data, &cards); err != nil {
			return nil, fmt.Errorf("load cards: %w", err)
		}
		for _, c := range cards {
			cs.cards[c.ID] = c
		}
	}
	return cs, nil
}

// GetByID returns a card by its ID.
func (cs *CardStore) GetByID(id string) (domain.Card, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	card, ok := cs.cards[id]
	if !ok {
		return domain.Card{}, fmt.Errorf("card with id %q not found", id)
	}
	return card, nil
}

// GetAll returns all cards.
func (cs *CardStore) GetAll() []domain.Card {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cards := make([]domain.Card, 0, len(cs.cards))
	for _, c := range cs.cards {
		cards = append(cards, c)
	}
	return cards
}

// Count returns the total number of cards.
func (cs *CardStore) Count() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.cards)
}

// SpreadStore provides read-only access to spread definitions.
type SpreadStore struct {
	mu      sync.RWMutex
	spreads map[string]domain.Spread
}

// NewSpreadStore creates a SpreadStore and loads spreads from JSON data.
func NewSpreadStore(jsonData []byte) (*SpreadStore, error) {
	var spreads []domain.Spread
	if err := json.Unmarshal(jsonData, &spreads); err != nil {
		return nil, fmt.Errorf("load spreads: %w", err)
	}

	ss := &SpreadStore{
		spreads: make(map[string]domain.Spread, len(spreads)),
	}
	for _, s := range spreads {
		ss.spreads[s.ID] = s
	}
	return ss, nil
}

// GetByID returns a spread by its ID.
func (ss *SpreadStore) GetByID(id string) (domain.Spread, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	spread, ok := ss.spreads[id]
	if !ok {
		return domain.Spread{}, fmt.Errorf("spread with id %q not found", id)
	}
	return spread, nil
}

// GetAll returns all spreads.
func (ss *SpreadStore) GetAll() []domain.Spread {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	spreads := make([]domain.Spread, 0, len(ss.spreads))
	for _, s := range ss.spreads {
		spreads = append(spreads, s)
	}
	return spreads
}

// Store aggregates all sub-stores.
type Store struct {
	io       *IO
	Cards    *CardStore
	Spreads  *SpreadStore
	Readings *ReadingStore
	Elements *ElementKnowledge
}

// New creates and initializes all sub-stores from embedded data.
func New() (*Store, error) {
	io := NewIO()

	cards, err := NewCardStore(
		majorArcanaJSON,
		minorWandsJSON,
		minorCupsJSON,
		minorSwordsJSON,
		minorPentaclesJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("init card store: %w", err)
	}

	spreads, err := NewSpreadStore(spreadsJSON)
	if err != nil {
		return nil, fmt.Errorf("init spread store: %w", err)
	}

	elements, err := NewElementKnowledge(elementsJSON)
	if err != nil {
		return nil, fmt.Errorf("init element knowledge: %w", err)
	}

	readingsPath, err := DefaultReadingsPath()
	if err != nil {
		return nil, fmt.Errorf("get readings path: %w", err)
	}

	return &Store{
		io:       io,
		Cards:    cards,
		Spreads:  spreads,
		Readings: NewReadingStore(readingsPath),
		Elements: elements,
	}, nil
}
