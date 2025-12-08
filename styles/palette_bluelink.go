package styles

import "github.com/charmbracelet/lipgloss"

// BluelinkPalette implements ColorPalette for the Bluelink CLI theme.
type BluelinkPalette struct{}

// Ensure BluelinkPalette implements ColorPalette at compile time.
var _ ColorPalette = (*BluelinkPalette)(nil)

// NewBluelinkPalette creates a new Bluelink color palette.
func NewBluelinkPalette() *BluelinkPalette {
	return &BluelinkPalette{}
}

func (p *BluelinkPalette) Name() string { return "bluelink" }

func (p *BluelinkPalette) Primary() lipgloss.TerminalColor {
	return lipgloss.AdaptiveColor{Light: "#072f8c", Dark: "#5882e2"}
}

func (p *BluelinkPalette) Secondary() lipgloss.TerminalColor {
	return lipgloss.Color("#2b63e3")
}

func (p *BluelinkPalette) Error() lipgloss.TerminalColor   { return ErrorColor }
func (p *BluelinkPalette) Warning() lipgloss.TerminalColor { return WarningColor }
func (p *BluelinkPalette) Info() lipgloss.TerminalColor    { return InfoColor }
func (p *BluelinkPalette) Success() lipgloss.TerminalColor { return SuccessColor }
func (p *BluelinkPalette) Muted() lipgloss.TerminalColor   { return MutedColor }
func (p *BluelinkPalette) Text() lipgloss.TerminalColor    { return TextColor }
func (p *BluelinkPalette) TextSubtle() lipgloss.TerminalColor { return TextSubtleColor }
