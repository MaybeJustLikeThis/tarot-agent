package cardlookup

import (
	"fmt"
	"io"

	"github.com/MaybeJustLikeThis/tarot-agent/internal/domain"
)

// Run writes one card, a candidate list error, or the full deck.
func Run(cards []domain.Card, query string, list bool, out io.Writer) error {
	if out == nil {
		return fmt.Errorf("没有输出")
	}
	if list {
		_, err := io.WriteString(out, FormatList(cards))
		return err
	}
	card, err := Lookup(cards, query)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, Format(*card))
	return err
}
