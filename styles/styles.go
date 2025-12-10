package styles

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Styles holds the styles to be used across command TUI components.
// This is the theme-agnostic styles struct that works with any ColorPalette.
type Styles struct {
	Selected          lipgloss.Style
	SelectedListItem  lipgloss.Style
	Selectable        lipgloss.Style
	Title             lipgloss.Style
	ListItem          lipgloss.Style
	Pagination        lipgloss.Style
	Help              lipgloss.Style
	Success           lipgloss.Style
	Error             lipgloss.Style
	Warning           lipgloss.Style
	Info              lipgloss.Style
	Muted             lipgloss.Style
	Spinner           lipgloss.Style
	Category          lipgloss.Style
	Location          lipgloss.Style
	DiagnosticMessage lipgloss.Style
	DiagnosticAction  lipgloss.Style
	Header            lipgloss.Style
	SelectedNavItem   lipgloss.Style
	Hint              lipgloss.Style
	Key               lipgloss.Style
	// For displaying CLI commands (e.g. "bluelink deploy --name my-stack")
	Command lipgloss.Style
	Border  func(focused bool) lipgloss.Style

	// Palette provides access to the underlying colors for custom rendering.
	Palette ColorPalette
}

// NewStyles creates a new instance of styles using the provided color palette.
func NewStyles(r *lipgloss.Renderer, palette ColorPalette) *Styles {
	// SelectedListItem uses PaddingLeft(2) to align with the "> " prefix added in rendering
	selectedListItem := r.NewStyle().
		PaddingLeft(2).
		Foreground(palette.Primary())

	return &Styles{
		Selected:          r.NewStyle().Foreground(palette.Primary()).Bold(true),
		SelectedListItem:  selectedListItem,
		Selectable:        r.NewStyle().Foreground(palette.Secondary()),
		Title:             r.NewStyle().Foreground(palette.Primary()).Bold(true),
		ListItem:          r.NewStyle().PaddingLeft(4),
		Pagination:        list.DefaultStyles().PaginationStyle.PaddingLeft(4),
		Help:              list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1),
		Success:           r.NewStyle().Foreground(palette.Success()),
		Error:             r.NewStyle().Foreground(palette.Error()),
		Warning:           r.NewStyle().Foreground(palette.Warning()),
		Info:              r.NewStyle().Foreground(palette.Info()),
		Muted:             r.NewStyle().Foreground(palette.Muted()),
		Spinner:           r.NewStyle().Foreground(palette.Primary()),
		Category:          r.NewStyle().Foreground(palette.Primary()),
		Location:          r.NewStyle().MarginLeft(2).Foreground(palette.Primary()),
		DiagnosticMessage: r.NewStyle().MarginLeft(2),
		DiagnosticAction:  r.NewStyle().MarginTop(2),
		Header:            r.NewStyle().Bold(true).Foreground(palette.Primary()),
		SelectedNavItem:   r.NewStyle().Background(palette.Primary()).Foreground(palette.Text()),
		Hint:              r.NewStyle().Foreground(palette.Primary()).Italic(true),
		Key:               r.NewStyle().Foreground(palette.Primary()).Bold(true),
		Command:           r.NewStyle().Foreground(palette.Info()),
		Border:            borderStyleFunc(palette),
		Palette:           palette,
	}
}

func borderStyleFunc(palette ColorPalette) func(focused bool) lipgloss.Style {
	return func(focused bool) lipgloss.Style {
		focusedBorderColor := palette.Primary()
		unfocusedBorderColor := palette.Muted()

		baseStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

		if focused {
			return baseStyle.BorderForeground(focusedBorderColor)
		}

		return baseStyle.BorderForeground(unfocusedBorderColor)
	}
}

// NewDefaultStyles creates a new instance of styles with the default renderer.
func NewDefaultStyles(palette ColorPalette) *Styles {
	return NewStyles(lipgloss.DefaultRenderer(), palette)
}
