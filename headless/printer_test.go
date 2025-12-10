package headless

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PrinterSuite struct {
	suite.Suite
	buf     *bytes.Buffer
	writer  *PrefixedWriter
	printer *Printer
}

func (s *PrinterSuite) SetupTest() {
	s.buf = &bytes.Buffer{}
	s.writer = NewPrefixedWriter(s.buf, "[test] ")
	s.printer = NewPrinter(s.writer, 80)
}

func TestPrinterSuite(t *testing.T) {
	suite.Run(t, new(PrinterSuite))
}

func (s *PrinterSuite) Test_progress_item_formats_correctly() {
	s.printer.ProgressItem("✓", "resource", "myDB", "CREATE", "")

	s.Equal("[test] ✓ resource: myDB - CREATE\n", s.buf.String())
}

func (s *PrinterSuite) Test_progress_item_with_suffix() {
	s.printer.ProgressItem("✓", "resource", "myDB", "CREATE", "(new)")

	s.Equal("[test] ✓ resource: myDB - CREATE (new)\n", s.buf.String())
}

func (s *PrinterSuite) Test_progress_item_different_icon() {
	s.printer.ProgressItem("✗", "child", "api", "DELETE", "(removed)")

	s.Equal("[test] ✗ child: api - DELETE (removed)\n", s.buf.String())
}

func (s *PrinterSuite) Test_item_header_with_padding() {
	s.printer.ItemHeader("resource", "myDB", "CREATE")

	output := s.buf.String()
	s.Contains(output, "[test] resource")
	s.Contains(output, "myDB")
	s.Contains(output, "CREATE\n")
}

func (s *PrinterSuite) Test_item_header_long_name() {
	longName := strings.Repeat("x", 100)
	s.printer.ItemHeader("resource", longName, "UPDATE")

	output := s.buf.String()
	s.Contains(output, longName)
	s.Contains(output, "UPDATE\n")
}

func (s *PrinterSuite) Test_field_add() {
	s.printer.FieldAdd("spec.cpu", "4")

	s.Equal("[test]   + spec.cpu: 4\n", s.buf.String())
}

func (s *PrinterSuite) Test_field_modify() {
	s.printer.FieldModify("spec.replicas", "2", "4")

	s.Equal("[test]   ~ spec.replicas: 2 -> 4\n", s.buf.String())
}

func (s *PrinterSuite) Test_field_remove() {
	s.printer.FieldRemove("spec.oldField")

	s.Equal("[test]   - spec.oldField\n", s.buf.String())
}

func (s *PrinterSuite) Test_no_changes() {
	s.printer.NoChanges()

	s.Equal("[test]   (no field changes)\n", s.buf.String())
}

func (s *PrinterSuite) Test_count_summary_singular() {
	s.printer.CountSummary(1, "resource", "resources", "to be created")

	s.Equal("[test]   1 resource to be created\n", s.buf.String())
}

func (s *PrinterSuite) Test_count_summary_plural() {
	s.printer.CountSummary(3, "resource", "resources", "to be created")

	s.Equal("[test]   3 resources to be created\n", s.buf.String())
}

func (s *PrinterSuite) Test_count_summary_zero_no_output() {
	s.printer.CountSummary(0, "resource", "resources", "to be created")

	s.Equal("", s.buf.String())
}

func (s *PrinterSuite) Test_diagnostic_without_location() {
	s.printer.Diagnostic("ERROR", "Resource not found", 0, 0)

	output := s.buf.String()
	s.Contains(output, "ERROR")
	s.Contains(output, "Resource not found")
	s.NotContains(output, "[line")
}

func (s *PrinterSuite) Test_diagnostic_with_location() {
	s.printer.Diagnostic("ERROR", "Resource not found", 42, 10)

	output := s.buf.String()
	s.Contains(output, "ERROR")
	s.Contains(output, "[line 42, col 10]")
	s.Contains(output, "Resource not found")
}

func (s *PrinterSuite) Test_diagnostic_warning_level() {
	s.printer.Diagnostic("WARNING", "Deprecated field used", 15, 5)

	output := s.buf.String()
	s.Contains(output, "WARNING")
	s.Contains(output, "[line 15, col 5]")
	s.Contains(output, "Deprecated field used")
}

func (s *PrinterSuite) Test_diagnostic_wraps_long_message() {
	longMessage := strings.Repeat("word ", 30)
	s.printer.Diagnostic("ERROR", longMessage, 1, 1)

	output := s.buf.String()
	s.Contains(output, "ERROR")
	// Long messages should be wrapped across multiple lines
	lines := strings.Split(output, "\n")
	s.Greater(len(lines), 1, "Long messages should wrap")
}

func (s *PrinterSuite) Test_diagnostic_continuation_indent() {
	longMessage := strings.Repeat("word ", 30)
	s.printer.Diagnostic("ERROR", longMessage, 10, 5)

	output := s.buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// First line should have the ERROR prefix
	s.Contains(lines[0], "ERROR")

	// Continuation lines should be indented
	if len(lines) > 1 {
		// Continuation lines should start with spaces (indentation)
		s.True(strings.HasPrefix(lines[1], " "), "Continuation lines should be indented")
	}
}

func (s *PrinterSuite) Test_next_step() {
	s.printer.NextStep("To apply these changes, run:", "bluelink deploy --changeset-id abc123")

	output := s.buf.String()
	s.Contains(output, "To apply these changes, run:")
	s.Contains(output, "bluelink deploy --changeset-id abc123")
}

func (s *PrinterSuite) Test_printer_default_width() {
	buf := &bytes.Buffer{}
	writer := NewPrefixedWriter(buf, "[test] ")
	printer := NewPrinter(writer, 0)

	s.Equal(80, printer.Width())
}

func (s *PrinterSuite) Test_printer_negative_width_defaults() {
	buf := &bytes.Buffer{}
	writer := NewPrefixedWriter(buf, "[test] ")
	printer := NewPrinter(writer, -10)

	s.Equal(80, printer.Width())
}

func (s *PrinterSuite) Test_printer_narrow_width() {
	buf := &bytes.Buffer{}
	writer := NewPrefixedWriter(buf, "[test] ")
	printer := NewPrinter(writer, 50)

	longMessage := strings.Repeat("word ", 20)
	printer.Diagnostic("ERROR", longMessage, 1, 1)

	// Should still work at narrow widths
	output := buf.String()
	s.Contains(output, "ERROR")
}

func (s *PrinterSuite) Test_writer_returns_prefixed_writer() {
	s.Equal(s.writer, s.printer.Writer())
}

func (s *PrinterSuite) Test_width_returns_configured_width() {
	s.Equal(80, s.printer.Width())
}
