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
	Error             lipgloss.Style
	Warning           lipgloss.Style
	Info              lipgloss.Style
	Muted             lipgloss.Style
	Spinner           lipgloss.Style
	Category          lipgloss.Style
	Location          lipgloss.Style
	DiagnosticMessage lipgloss.Style
	DiagnosticAction  lipgloss.Style

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
		Error:             r.NewStyle().Foreground(palette.Error()),
		Warning:           r.NewStyle().Foreground(palette.Warning()),
		Info:              r.NewStyle().Foreground(palette.Info()),
		Muted:             r.NewStyle().Foreground(palette.Muted()),
		Spinner:           r.NewStyle().Foreground(palette.Primary()),
		Category:          r.NewStyle().Foreground(palette.Primary()),
		Location:          r.NewStyle().MarginLeft(2).Foreground(palette.Primary()),
		DiagnosticMessage: r.NewStyle().MarginLeft(2),
		DiagnosticAction:  r.NewStyle().MarginTop(2),
		Palette:           palette,
	}
}

// NewDefaultStyles creates a new instance of styles with the default renderer.
func NewDefaultStyles(palette ColorPalette) *Styles {
	return NewStyles(lipgloss.DefaultRenderer(), palette)
}
