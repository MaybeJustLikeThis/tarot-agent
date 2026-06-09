package tools

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/voocel/tarot-agent/internal/domain"
	"github.com/voocel/tarot-agent/internal/store"
)

// DrawCards performs a card draw for the given spread type using the store.
// This is a standalone function (not a tool method) so it can be tested directly.
func DrawCards(s *store.Store, spreadType string) (*domain.DrawResult, error) {
	spread, err := s.Spreads.GetByID(spreadType)
	if err != nil {
		return nil, fmt.Errorf("get spread %q: %w", spreadType, err)
	}

	allCards := s.Cards.GetAll()
	needed := len(spread.Positions)

	if needed > len(allCards) {
		return nil, fmt.Errorf("need %d cards but only %d available", needed, len(allCards))
	}

	// Fisher-Yates shuffle with crypto/rand
	shuffled := make([]domain.Card, len(allCards))
	copy(shuffled, allCards)
	for i := len(shuffled) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, fmt.Errorf("crypto/rand: %w", err)
		}
		j := int(jBig.Int64())
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	drawn := make([]domain.DrawnCard, needed)
	for i := 0; i < needed; i++ {
		orientation, err := randomOrientation()
		if err != nil {
			return nil, fmt.Errorf("random orientation: %w", err)
		}
		drawn[i] = domain.DrawnCard{
			Card:        shuffled[i],
			Orientation: orientation,
			Position:    spread.Positions[i],
		}
	}

	return &domain.DrawResult{
		SpreadID: spread.ID,
		Cards:    drawn,
	}, nil
}

// randomOrientation returns upright or reversed with 50/50 probability.
func randomOrientation() (domain.Orientation, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return "", fmt.Errorf("crypto/rand for orientation: %w", err)
	}
	if n.Int64() == 0 {
		return domain.Upright, nil
	}
	return domain.Reversed, nil
}
