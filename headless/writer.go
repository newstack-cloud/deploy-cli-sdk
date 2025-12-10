package headless

import (
	"fmt"
	"io"
	"strings"
)

// PrefixedWriter wraps an io.Writer to auto-prefix all lines.
type PrefixedWriter struct {
	w      io.Writer
	prefix string // e.g., "[stage] ", "[deploy] "
}

// NewPrefixedWriter creates a writer that prefixes all output lines.
func NewPrefixedWriter(w io.Writer, prefix string) *PrefixedWriter {
	return &PrefixedWriter{w: w, prefix: prefix}
}

// Printf writes a formatted line with the prefix.
func (pw *PrefixedWriter) Printf(format string, args ...any) {
	fmt.Fprintf(pw.w, pw.prefix+format, args...)
}

// Println writes a line with the prefix.
func (pw *PrefixedWriter) Println(s string) {
	fmt.Fprintln(pw.w, pw.prefix+s)
}

// PrintlnEmpty writes an empty line (no prefix).
func (pw *PrefixedWriter) PrintlnEmpty() {
	fmt.Fprintln(pw.w)
}

// Separator writes a separator line with the given character.
func (pw *PrefixedWriter) Separator(char rune, width int) {
	fmt.Fprintln(pw.w, pw.prefix+strings.Repeat(string(char), width))
}

// DoubleSeparator writes a double-line separator (═).
func (pw *PrefixedWriter) DoubleSeparator(width int) {
	pw.Separator('═', width)
}

// SingleSeparator writes a single-line separator (─).
func (pw *PrefixedWriter) SingleSeparator(width int) {
	pw.Separator('─', width)
}

// Writer returns the underlying io.Writer.
func (pw *PrefixedWriter) Writer() io.Writer {
	return pw.w
}

// Prefix returns the prefix string.
func (pw *PrefixedWriter) Prefix() string {
	return pw.prefix
}
