package styles

import "github.com/charmbracelet/lipgloss"

// CelerityPalette implements ColorPalette for the Celerity CLI theme.
type CelerityPalette struct{}

// Ensure CelerityPalette implements ColorPalette at compile time.
var _ ColorPalette = (*CelerityPalette)(nil)

// NewCelerityPalette creates a new Celerity color palette.
func NewCelerityPalette() *CelerityPalette {
	return &CelerityPalette{}
}

func (p *CelerityPalette) Name() string { return "celerity" }

func (p *CelerityPalette) Primary() lipgloss.TerminalColor {
	// Indigo colors with light/dark mode support (indigo-600/400)
	return lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#818cf8"}
}

func (p *CelerityPalette) Secondary() lipgloss.TerminalColor {
	return lipgloss.Color("#6366f1") // indigo-500
}

func (p *CelerityPalette) Error() lipgloss.TerminalColor   { return ErrorColor }
func (p *CelerityPalette) Warning() lipgloss.TerminalColor { return WarningColor }
func (p *CelerityPalette) Info() lipgloss.TerminalColor    { return InfoColor }
func (p *CelerityPalette) Success() lipgloss.TerminalColor { return SuccessColor }
func (p *CelerityPalette) Muted() lipgloss.TerminalColor   { return MutedColor }
func (p *CelerityPalette) Text() lipgloss.TerminalColor    { return TextColor }
func (p *CelerityPalette) TextSubtle() lipgloss.TerminalColor { return TextSubtleColor }
