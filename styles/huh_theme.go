package styles

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// NewHuhTheme creates a huh form theme using the provided color palette.
func NewHuhTheme(palette ColorPalette) *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styles
	t.Focused.Title = lipgloss.NewStyle().Foreground(palette.Primary()).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(palette.Muted())
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(palette.Error())
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(palette.Error())

	// Select styles
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(palette.Primary()).SetString("> ")
	t.Focused.Option = lipgloss.NewStyle().Foreground(palette.Text())
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(palette.Primary()).Bold(true)

	// Text input styles
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(palette.Primary())
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(palette.Primary())
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(palette.Text())
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(palette.TextSubtle())

	// Confirm button styles
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(palette.Secondary()).
		Padding(0, 1).
		Bold(true)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(palette.Text()).
		Padding(0, 1)

	// Blurred field styles (less prominent)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(palette.Muted())
	t.Blurred.Description = lipgloss.NewStyle().Foreground(palette.TextSubtle())
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(palette.Muted())
	t.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.Option = lipgloss.NewStyle().Foreground(palette.Muted())
	t.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(palette.Muted())

	t.Blurred.FocusedButton = lipgloss.NewStyle().
		Foreground(palette.Text()).
		Padding(0, 1)
	t.Blurred.BlurredButton = lipgloss.NewStyle().
		Foreground(palette.Muted()).
		Padding(0, 1)

	return t
}

// NewBluelinkHuhTheme creates a huh form theme using the Bluelink color scheme.
// Deprecated: Use NewHuhTheme with NewBluelinkPalette() instead.
func NewBluelinkHuhTheme() *huh.Theme {
	return NewHuhTheme(NewBluelinkPalette())
}
