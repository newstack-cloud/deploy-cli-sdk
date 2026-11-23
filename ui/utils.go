package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SafeWidth returns a safe width for separator lines, ensuring it's never negative.
func SafeWidth(width int) int {
	if width <= 0 {
		return 40 // Default width for headless mode
	}
	return width
}

// TruncatePath shortens a file path to fit within maxLen characters,
// preserving the file name and as much of the trailing directory
// context as possible.
// For example, "/very/long/path/to/project.blueprint.yml" with maxLen 30
// becomes "…/to/project.blueprint.yml".
func TruncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	fileName := filepath.Base(path)
	if maxLen <= len(fileName) {
		return fileName
	}

	dir := filepath.Dir(path)
	parts := strings.Split(dir, string(filepath.Separator))

	// Build the truncated path from the right, adding directory
	// segments until we run out of space.
	// Reserve space for the ellipsis prefix "…/" and the filename.
	const ellipsis = "…"
	available := maxLen - len(ellipsis) - 1 - len(fileName) // "…" + "/" + filename
	if available <= 0 {
		return ellipsis + string(filepath.Separator) + fileName
	}

	var kept []string
	used := 0
	for i := len(parts) - 1; i >= 0; i-- {
		seg := parts[i]
		need := len(seg)
		if len(kept) > 0 {
			need += 1 // separator
		}
		if used+need > available {
			break
		}
		kept = append([]string{seg}, kept...)
		used += need
	}

	suffix := strings.Join(kept, string(filepath.Separator))
	if suffix != "" {
		return ellipsis + string(filepath.Separator) + suffix + string(filepath.Separator) + fileName
	}
	return ellipsis + string(filepath.Separator) + fileName
}

// Below this many columns for the value, aligning it under its label would leave too
// little room to read it.
const minWrapWidth = 20

// WrapLabelled renders prefix followed by value, wrapping the value to fit within width
// and aligning continuation lines under the start of the value. The prefix holds any
// indentation and the label, e.g. "    Target: ". Values without spaces, such as long
// paths, are broken mid-word. In a terminal too narrow to leave the value a readable
// column, the whole line is wrapped to width instead, without the alignment.
func WrapLabelled(prefix string, value string, width int) string {
	width = SafeWidth(width)
	prefixWidth := lipgloss.Width(prefix)
	if width-prefixWidth < minWrapWidth {
		return trimLineEnds(lipgloss.NewStyle().Width(width).Render(prefix + value))
	}

	wrapped := trimLineEnds(lipgloss.NewStyle().Width(width - prefixWidth).Render(value))
	padding := strings.Repeat(" ", prefixWidth)
	return prefix + strings.ReplaceAll(wrapped, "\n", "\n"+padding)
}

// lipgloss pads every wrapped line to the full width with spaces.
func trimLineEnds(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
