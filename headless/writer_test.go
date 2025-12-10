package headless

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WriterSuite struct {
	suite.Suite
}

func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterSuite))
}

func (s *WriterSuite) Test_println_adds_prefix() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[stage] ")

	w.Println("Starting deployment")

	s.Equal("[stage] Starting deployment\n", buf.String())
}

func (s *WriterSuite) Test_printf_adds_prefix() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[deploy] ")

	w.Printf("count: %d\n", 5)

	s.Equal("[deploy] count: 5\n", buf.String())
}

func (s *WriterSuite) Test_printf_with_multiple_args() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[test] ")

	w.Printf("name=%s, value=%d\n", "foo", 42)

	s.Equal("[test] name=foo, value=42\n", buf.String())
}

func (s *WriterSuite) Test_println_empty_no_prefix() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[stage] ")

	w.PrintlnEmpty()

	s.Equal("\n", buf.String())
}

func (s *WriterSuite) Test_separator_creates_line() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[test] ")

	w.Separator('─', 10)

	s.Equal("[test] ──────────\n", buf.String())
}

func (s *WriterSuite) Test_double_separator() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[stage] ")

	w.DoubleSeparator(5)

	s.Equal("[stage] ═════\n", buf.String())
}

func (s *WriterSuite) Test_single_separator() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[deploy] ")

	w.SingleSeparator(5)

	s.Equal("[deploy] ─────\n", buf.String())
}

func (s *WriterSuite) Test_empty_prefix() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "")

	w.Println("no prefix")

	s.Equal("no prefix\n", buf.String())
}

func (s *WriterSuite) Test_prefix_without_trailing_space() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[compact]")

	w.Println("message")

	s.Equal("[compact]message\n", buf.String())
}

func (s *WriterSuite) Test_writer_returns_underlying_writer() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[test] ")

	s.Equal(buf, w.Writer())
}

func (s *WriterSuite) Test_prefix_returns_prefix_string() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[stage] ")

	s.Equal("[stage] ", w.Prefix())
}

func (s *WriterSuite) Test_separator_zero_width() {
	buf := &bytes.Buffer{}
	w := NewPrefixedWriter(buf, "[test] ")

	w.Separator('═', 0)

	s.Equal("[test] \n", buf.String())
}
