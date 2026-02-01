package ui

import (
	"path/filepath"
	"strings"
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
