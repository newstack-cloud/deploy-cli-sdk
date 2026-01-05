package headless

import (
	"fmt"
	"strings"

	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// Printer provides common headless output patterns.
type Printer struct {
	w     *PrefixedWriter
	width int // Terminal width for wrapping
}

// NewPrinter creates a headless printer.
func NewPrinter(w *PrefixedWriter, width int) *Printer {
	if width <= 0 {
		width = 80 // Default width
	}
	return &Printer{w: w, width: width}
}

// Writer returns the underlying PrefixedWriter.
func (p *Printer) Writer() *PrefixedWriter {
	return p.w
}

// Width returns the configured terminal width.
func (p *Printer) Width() int {
	return p.width
}

// ProgressItem prints a progress event line.
// Format: icon type: name - action (suffix)
func (p *Printer) ProgressItem(icon, itemType, name, action, suffix string) {
	line := fmt.Sprintf("%s %s: %s - %s", icon, itemType, name, action)
	if suffix != "" {
		line += " " + suffix
	}
	p.w.Println(line)
}

// ItemHeader prints an item header with type, name, and action.
// Format: itemType (padded to 10 chars) + space + name + padding + action
// The action is right-aligned to column 60.
func (p *Printer) ItemHeader(itemType, name, action string) {
	const typeWidth = 10
	p.w.Printf("%-*s %s", typeWidth, itemType, name)
	// Calculate padding: 60 total - typeWidth (fixed) - 1 (space after type) - name length
	padding := 60 - typeWidth - 1 - len(name)
	if padding > 0 {
		fmt.Fprint(p.w.Writer(), strings.Repeat(" ", padding))
	}
	fmt.Fprintf(p.w.Writer(), " %s\n", action)
}

// FieldAdd prints an added field.
func (p *Printer) FieldAdd(path, value string) {
	p.w.Printf("  + %s: %s\n", path, value)
}

// FieldModify prints a modified field.
func (p *Printer) FieldModify(path, oldValue, newValue string) {
	p.w.Printf("  ~ %s: %s -> %s\n", path, oldValue, newValue)
}

// FieldRemove prints a removed field.
func (p *Printer) FieldRemove(path string) {
	p.w.Printf("  - %s\n", path)
}

// NoChanges prints a "no changes" message.
func (p *Printer) NoChanges() {
	p.w.Println("  (no field changes)")
}

// CountSummary prints a count with pluralization.
func (p *Printer) CountSummary(count int, singular, plural, verb string) {
	if count > 0 {
		p.w.Printf("  %d %s %s\n", count, sdkstrings.Pluralize(count, singular, plural), verb)
	}
}

// Diagnostic prints a diagnostic message with level, location, and wrapped text.
func (p *Printer) Diagnostic(level string, message string, line, col int) {
	// Build location string
	location := ""
	if line > 0 {
		location = fmt.Sprintf(" [line %d, col %d]", line, col)
	}

	// Calculate prefix for wrapping
	prefix := fmt.Sprintf("  %s%s: ", level, location)
	availableWidth := p.width - len(p.w.Prefix()) - len(prefix) - 4
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width for readability
	}

	// Wrap message
	wrappedMessage := sdkstrings.WrapText(message, availableWidth)
	messageLines := strings.Split(wrappedMessage, "\n")

	// First line includes the prefix
	if len(messageLines) > 0 {
		p.w.Printf("  %s%s: %s\n", level, location, messageLines[0])
	}

	// Continuation lines are indented to align with the message
	indent := strings.Repeat(" ", len(prefix))
	for i := 1; i < len(messageLines); i++ {
		fmt.Fprintf(p.w.Writer(), "%s%s\n", indent, messageLines[i])
	}
}

// NextStep prints a "to do X, run:" instruction.
func (p *Printer) NextStep(description, command string) {
	p.w.Println(description)
	p.w.Printf("  %s\n", command)
}
