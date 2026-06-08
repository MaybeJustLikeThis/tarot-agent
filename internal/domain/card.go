package domain

import "time"

// ArcanaType distinguishes Major from Minor arcana.
type ArcanaType string

const (
	MajorArcana ArcanaType = "major"
	MinorArcana ArcanaType = "minor"
)

// Suit represents the four suits in Minor Arcana.
type Suit string

const (
	SuitWands     Suit = "wands"
	SuitCups      Suit = "cups"
	SuitSwords    Suit = "swords"
	SuitPentacles Suit = "pentacles"
	SuitNone      Suit = ""
)

// Orientation indicates whether a card is upright or reversed.
type Orientation string

const (
	Upright  Orientation = "upright"
	Reversed Orientation = "reversed"
)

// Card represents a single tarot card.
type Card struct {
	ID              string     `json:"id"`
	NameCN          string     `json:"name_cn"`
	NameEN          string     `json:"name_en"`
	Arcana          ArcanaType `json:"arcana"`
	Suit            Suit       `json:"suit"`
	Number          int        `json:"number"`
	UprightKeywords []string   `json:"upright_keywords"`
	ReversedKeywords []string  `json:"reversed_keywords"`
	UprightMeaning  string     `json:"upright_meaning"`
	ReversedMeaning string     `json:"reversed_meaning"`
	ImageFile        string           `json:"image_file"`
	NameAliases      []string         `json:"name_aliases,omitempty"`
	// Enriched fields (added for professional readings)
	Element          string           `json:"element,omitempty"`
	Astrology        *AstrologyInfo   `json:"astrology,omitempty"`
	Numerology       *NumerologyInfo  `json:"numerology,omitempty"`
	Imagery          string           `json:"imagery,omitempty"`
	KeywordsContext  *KeywordsContext  `json:"keywords_context,omitempty"`
	CourtRole        *CourtRole       `json:"court_role,omitempty"`
}

// AstrologyInfo holds astrological correspondences for a card.
type AstrologyInfo struct {
	Planet string `json:"planet,omitempty"`
	Zodiac string `json:"zodiac,omitempty"`
	Note   string `json:"note,omitempty"`
}

// NumerologyInfo holds numerological significance of a card.
type NumerologyInfo struct {
	Number  int    `json:"number"`
	Meaning string `json:"meaning"`
	Note    string `json:"note,omitempty"`
}

// KeywordsContext holds scenario-specific keywords.
type KeywordsContext struct {
	Love   []string `json:"love"`
	Career []string `json:"career"`
	Growth []string `json:"growth"`
}

// CourtRole describes the special role of court cards (Page/Knight/Queen/King).
type CourtRole struct {
	Archetype   string `json:"archetype"`
	Personality string `json:"personality"`
	AsPerson    string `json:"as_person"`
	AsMessage   string `json:"as_message"`
}

// Position describes a named position within a spread.
type Position struct {
	Index       int    `json:"index"`
	NameCN      string `json:"name_cn"`
	NameEN      string `json:"name_en"`
	Description string `json:"description"`
}

// Spread defines a card layout pattern.
type Spread struct {
	ID          string     `json:"id"`
	NameCN      string     `json:"name_cn"`
	NameEN      string     `json:"name_en"`
	Description string     `json:"description"`
	Positions   []Position `json:"positions"`
	MinCards    int        `json:"min_cards"`
	MaxCards    int        `json:"max_cards"`
}

// DrawnCard pairs a card with its orientation and spread position.
type DrawnCard struct {
	Card        Card        `json:"card"`
	Orientation Orientation `json:"orientation"`
	Position    Position    `json:"position"`
}

// DrawResult holds the outcome of a card draw.
type DrawResult struct {
	SpreadID string       `json:"spread_id"`
	Cards    []DrawnCard  `json:"cards"`
}

// Reading represents a complete tarot reading session.
type Reading struct {
	ID             string      `json:"id"`
	Question       string      `json:"question"`
	SpreadID       string      `json:"spread_id"`
	Cards          []DrawnCard `json:"cards"`
	Interpretation string      `json:"interpretation"`
	Disclaimer     string      `json:"disclaimer"`
	Model          string      `json:"model"`
	CreatedAt      time.Time   `json:"created_at"`
}
