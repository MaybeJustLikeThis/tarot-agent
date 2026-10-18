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
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}
)

// Reusable style instances (replaces per-call lipgloss helper functions).
var (
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleSubtle  = lipgloss.NewStyle().Foreground(colorSubtle)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
)

// Input area
var inputLabelStyle = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true)

// Spread selection
var spreadNumberStyle = lipgloss.NewStyle().
	Foreground(colorPrimary).
	Bold(true)

var spreadDescStyle = lipgloss.NewStyle().
	Foreground(colorMuted)

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
