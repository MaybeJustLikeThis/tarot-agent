package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	mdHeadingRe = regexp.MustCompile(`(?m)^#{1,3}\s+(.+)$`)
	mdBoldRe    = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdListRe    = regexp.MustCompile(`(?m)^[-*]\s+`)
)

// renderMarkdown converts basic markdown to terminal-styled text.
// maxWidth controls line wrapping (0 = no wrapping).
func renderMarkdown(text string, maxWidth int) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Wrap long lines before applying styles (viewport can't handle
		// ANSI-inflated line widths correctly).
		wrapped := []string{line}
		if maxWidth > 0 {
			wrapped = wrapLine(line, maxWidth)
		}

		for _, wline := range wrapped {
			if matches := mdHeadingRe.FindStringSubmatch(wline); len(matches) > 1 {
				styled := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("▎ " + matches[1])
				result = append(result, styled)
				continue
			}

			if mdListRe.MatchString(wline) {
				styled := mdListRe.ReplaceAllString(wline, "  • ")
				result = append(result, styled)
				continue
			}

			styled := mdBoldRe.ReplaceAllStringFunc(wline, func(match string) string {
				inner := mdBoldRe.FindStringSubmatch(match)
				if len(inner) > 1 {
					return lipgloss.NewStyle().Bold(true).Render(inner[1])
				}
				return match
			})

			result = append(result, styled)
		}
	}

	return strings.Join(result, "\n")
}

// wrapLine splits a plain text line to fit within maxWidth visible columns.
// CJK characters count as 2 columns. Returns the original line if it fits.
func wrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 || stringWidth(line) <= maxWidth {
		return []string{line}
	}

	var result []string
	var current strings.Builder
	currentW := 0

	for _, r := range line {
		rw := runeWidth(r)
		if currentW+rw > maxWidth && current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
			currentW = 0
		}
		current.WriteRune(r)
		currentW += rw
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// stringWidth returns the display width of a string (CJK = 2 columns).
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// runeWidth returns the display width of a single rune.
func runeWidth(r rune) int {
	if r < 0x100 {
		return 1
	}
	// CJK Unified Ideographs and common fullwidth ranges
	if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified
		(r >= 0x3000 && r <= 0x303F) || // CJK Punctuation
		(r >= 0xFF00 && r <= 0xFFEF) || // Fullwidth Forms
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Ext A
		(r >= 0x2E80 && r <= 0x2EFF) { // CJK Radicals
		return 2
	}
	// Fallback: count UTF-8 bytes as rough proxy (not perfect but good enough)
	_ = utf8.RuneLen(r)
	return 1
}
