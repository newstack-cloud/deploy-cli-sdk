package strings

import "strings"

// FromPointer returns the string value of a pointer to a string.
// If the pointer is nil, it returns an empty string.
func FromPointer(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TruncateString truncates a string to a maximum length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Pluralize returns the singular form if count is 1, otherwise the plural form.
func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// WrapText wraps text to the specified width, breaking at word boundaries.
func WrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLen == 0 {
			// Start of a new line
			result.WriteString(word)
			lineLen = wordLen
		} else if lineLen+1+wordLen <= width {
			// Word fits on current line
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		} else {
			// Start a new line
			result.WriteString("\n")
			result.WriteString(word)
			lineLen = wordLen
		}

		_ = i // silence unused variable warning
	}

	return result.String()
}
