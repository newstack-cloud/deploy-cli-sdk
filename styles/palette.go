package styles

import "github.com/charmbracelet/lipgloss"

// ColorPalette defines the color scheme for a CLI theme.
// Themes must implement this interface to provide their brand colors.
type ColorPalette interface {
	// Name returns the theme identifier (e.g., "bluelink", "celerity").
	Name() string

	// Brand colors - these differ per theme
	Primary() lipgloss.TerminalColor
	Secondary() lipgloss.TerminalColor

	// Semantic colors - typically consistent across themes
	Error() lipgloss.TerminalColor
	Warning() lipgloss.TerminalColor
	Info() lipgloss.TerminalColor
	Success() lipgloss.TerminalColor

	// UI colors
	Muted() lipgloss.TerminalColor
	Text() lipgloss.TerminalColor
	TextSubtle() lipgloss.TerminalColor
}
