package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "#8B5CF6", Dark: "#A78BFA"}
	colorSecondary = lipgloss.AdaptiveColor{Light: "#EC4899", Dark: "#F472B6"}
	colorAccent    = lipgloss.AdaptiveColor{Light: "#F59E0B", Dark: "#FBBF24"}
	colorMuted     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorSuccess   = lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"}
	colorError     = lipgloss.AdaptiveColor{Light: "#EF4444", Dark: "#F87171"}
	colorBorder    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#4B5563"}
	colorText      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"}
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}
)

// Title styles
var titleStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true).
	Padding(0, 1)

var subtitleStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Italic(true).
	Padding(0, 1)

// Input area
var inputLabelStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true)

var inputPromptStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

// Spread selection
var spreadOptionStyle = lipgloss.NewStyle().
	Foreground(colorText).
	Padding(0, 2)

var spreadNumberStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

var spreadDescStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

// Reading output
var readingBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorPrimary).
	Padding(1, 2)

var readingTextStyle = lipgloss.NewStyle().
	Foreground(colorText)

var toolCallStyle = lipgloss.NewStyle().
	Foreground(colorSubtle).
	Italic(true)

var disclaimerStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Italic(true).
	Padding(0, 1)

// Status bar
var statusBarStyle = lipgloss.NewStyle().
	Foreground(colorSubtle).
	Padding(0, 1)

// Error
var errorStyle = lipgloss.NewStyle().
	Foreground(colorError).
	Bold(true)

// Spinner
var spinnerStyle = lipgloss.NewStyle().
	Foreground(colorPrimary)

// Separator
var separatorStyle = lipgloss.NewStyle().
	Foreground(colorBorder)
