package styles

import "github.com/charmbracelet/lipgloss"

// Shared semantic colors - consistent across all themes.
// These colors follow user expectations for status indicators.
var (
	// Status colors (Tailwind-based)
	ErrorColor   = lipgloss.Color("#dc2626") // red-600
	WarningColor = lipgloss.Color("#f97316") // orange-500
	InfoColor    = lipgloss.Color("#2563eb") // blue-600
	SuccessColor = lipgloss.Color("#16a34a") // green-600

	// Neutral colors with light/dark mode support
	MutedColor      = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	TextColor       = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#ffffff"}
	TextSubtleColor = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"}
)
