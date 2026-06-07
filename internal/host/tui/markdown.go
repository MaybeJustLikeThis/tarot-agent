package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	mdHeadingRe = regexp.MustCompile(`(?m)^#{1,3}\s+(.+)$`)
	mdBoldRe    = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdListRe    = regexp.MustCompile(`(?m)^[-*]\s+`)
)

// renderMarkdown converts basic markdown to terminal-styled text.
func renderMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if matches := mdHeadingRe.FindStringSubmatch(line); len(matches) > 1 {
			styled := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("▎ " + matches[1])
			result = append(result, styled)
			continue
		}

		if mdListRe.MatchString(line) {
			styled := mdListRe.ReplaceAllString(line, "  • ")
			result = append(result, styled)
			continue
		}

		styled := mdBoldRe.ReplaceAllStringFunc(line, func(match string) string {
			inner := mdBoldRe.FindStringSubmatch(match)
			if len(inner) > 1 {
				return lipgloss.NewStyle().Bold(true).Render(inner[1])
			}
			return match
		})

		result = append(result, styled)
	}

	return strings.Join(result, "\n")
}
