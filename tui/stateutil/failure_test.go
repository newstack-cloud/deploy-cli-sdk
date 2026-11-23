package stateutil

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type FailureTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
	err    error
}

func TestFailureTestSuite(t *testing.T) {
	suite.Run(t, new(FailureTestSuite))
}

func (s *FailureTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
	s.err = &stateio.StateAccessError{
		Code:    stateio.ErrCodeStateDecryptionFailed,
		Message: "the state could not be decrypted with the configured key",
		Hint:    "Check that the configured key is the one this state was encrypted with.",
		Err:     errors.New("crypto: envelope was sealed with key \"a\", this provider holds \"b\""),
	}
}

func (s *FailureTestSuite) Test_renders_aligned_labelled_lines_and_the_hint() {
	details := []LabelledValue{{Label: "Source", Value: "memfile state in /state"}}
	view := RenderFailure(s.styles, "Export failed", details, s.err, 120)

	s.Contains(view, "✗ Export failed\n")
	s.Contains(view, "    Source: memfile state in /state\n")
	s.Contains(view, "    Error:  the state could not be decrypted with the configured key\n")
	s.Contains(view, "    Cause:  envelope was sealed with key \"a\", this provider holds \"b\"\n")
	s.Contains(view, "\n    Check that the configured key is the one this state was encrypted with.\n")
	s.NotContains(view, "state_decryption_failed")
}

func (s *FailureTestSuite) Test_wraps_every_line_to_the_width() {
	const width = 50
	view := RenderFailure(s.styles, "Export failed", nil, s.err, width)

	for _, line := range strings.Split(view, "\n") {
		s.LessOrEqual(lipgloss.Width(line), width, line)
	}
}

func (s *FailureTestSuite) Test_formats_plain_text_for_headless_output() {
	details := []LabelledValue{{Label: "Source", Value: "memfile state in /state"}}
	s.Equal(
		"Export failed: the state could not be decrypted with the configured key\n"+
			"  Source: memfile state in /state\n"+
			"  Cause: envelope was sealed with key \"a\", this provider holds \"b\"\n"+
			"  Check that the configured key is the one this state was encrypted with.\n",
		FormatFailureText("Export failed", details, s.err),
	)
}

func (s *FailureTestSuite) Test_formats_an_error_without_a_cause_or_hint() {
	text := FormatFailureText("Import failed", nil, errors.New("connection refused"))
	s.Equal("Import failed: connection refused\n", text)
}

func (s *FailureTestSuite) Test_indents_a_multi_line_summary() {
	err := &stateio.ImportError{
		Code:    stateio.ErrCodeInstanceNameImmutable,
		Message: "nothing was imported\ninstances cannot be renamed:\n  - inst-1 is named \"a\", not \"b\"",
	}

	s.Equal(
		"Import failed: nothing was imported\n"+
			"  instances cannot be renamed:\n"+
			"    - inst-1 is named \"a\", not \"b\"\n",
		FormatFailureText("Import failed", nil, err),
	)
}
