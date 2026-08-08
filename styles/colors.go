package styles

import "github.com/charmbracelet/lipgloss"

// Shared semantic colors - consistent across all themes.
// These colors follow user expectations for status indicators.
//
// Status colors are adaptive: the Light variants are darker (Tailwind -700)
// shades that stay legible on light terminal backgrounds, and the Dark variants
// are brighter (Tailwind -400) shades for dark backgrounds. lipgloss selects
// between them based on the detected terminal background.
var (
	// Status colors (Tailwind-based)
	ErrorColor   = lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f87171"} // red-700 / red-400
	WarningColor = lipgloss.AdaptiveColor{Light: "#c2410c", Dark: "#fb923c"} // orange-700 / orange-400
	InfoColor    = lipgloss.AdaptiveColor{Light: "#1d4ed8", Dark: "#60a5fa"} // blue-700 / blue-400
	SuccessColor = lipgloss.AdaptiveColor{Light: "#15803d", Dark: "#4ade80"} // green-700 / green-400

	// Neutral colors with light/dark mode support
	MutedColor      = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	TextColor       = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#ffffff"}
	TextSubtleColor = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"}
)
