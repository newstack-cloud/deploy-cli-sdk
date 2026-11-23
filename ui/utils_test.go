package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/stretchr/testify/suite"
)

type UtilsSuite struct {
	suite.Suite
}

func (s *UtilsSuite) Test_truncate_path_returns_path_when_fits() {
	path := "/home/user/project.blueprint.yml"
	result := TruncatePath(path, 50)
	s.Equal(path, result)
}

func (s *UtilsSuite) Test_truncate_path_returns_path_when_exact_fit() {
	path := "/home/user/project.blueprint.yml"
	result := TruncatePath(path, len(path))
	s.Equal(path, result)
}

func (s *UtilsSuite) Test_truncate_path_truncates_long_path() {
	path := "/very/long/path/to/some/deeply/nested/project.blueprint.yml"
	result := TruncatePath(path, 40)
	s.LessOrEqual(len(result), 40)
	s.Contains(result, "project.blueprint.yml")
	s.Contains(result, "…/")
}

func (s *UtilsSuite) Test_truncate_path_preserves_filename() {
	path := "/very/long/path/to/project.blueprint.yml"
	result := TruncatePath(path, 30)
	s.Contains(result, "project.blueprint.yml")
}

func (s *UtilsSuite) Test_truncate_path_returns_filename_when_maxlen_very_small() {
	path := "/a/b/project.blueprint.yml"
	result := TruncatePath(path, len("project.blueprint.yml"))
	s.Equal("project.blueprint.yml", result)
}

func (s *UtilsSuite) Test_truncate_path_keeps_trailing_directories() {
	path := "/very/long/path/to/nested/project.blueprint.yml"
	result := TruncatePath(path, 35)
	s.Contains(result, "project.blueprint.yml")
	s.Contains(result, "…/")
	s.Contains(result, "nested")
}

func (s *UtilsSuite) Test_safe_width_returns_default_for_zero() {
	s.Equal(40, SafeWidth(0))
}

func (s *UtilsSuite) Test_safe_width_returns_default_for_negative() {
	s.Equal(40, SafeWidth(-10))
}

func (s *UtilsSuite) Test_safe_width_returns_value_when_positive() {
	s.Equal(80, SafeWidth(80))
}

func (s *UtilsSuite) Test_wrap_labelled_keeps_a_short_value_on_one_line() {
	s.Equal("    Target: memfile state in /tmp/state", WrapLabelled("    Target: ", "memfile state in /tmp/state", 80))
}

func (s *UtilsSuite) Test_wrap_labelled_aligns_continuation_lines_under_the_value() {
	value := "memfile state in /Users/someone/projects/some-project/.bluelink/deploy-engine/state"
	result := WrapLabelled("    Target: ", value, 50)

	lines := strings.Split(result, "\n")
	s.Greater(len(lines), 1)
	s.True(strings.HasPrefix(lines[0], "    Target: memfile state in"))
	for _, line := range lines {
		s.LessOrEqual(lipgloss.Width(line), 50)
	}
	for _, line := range lines[1:] {
		s.True(strings.HasPrefix(line, strings.Repeat(" ", len("    Target: "))), line)
		s.NotEqual(' ', line[len("    Target: ")], line)
	}

	joined := ""
	for i, line := range lines {
		if i == 0 {
			line = strings.TrimPrefix(line, "    Target: ")
		}
		joined += strings.TrimSpace(line)
	}
	s.Equal(strings.ReplaceAll(value, " ", ""), strings.ReplaceAll(joined, " ", ""))
}

func (s *UtilsSuite) Test_wrap_labelled_stays_within_a_terminal_too_narrow_to_align() {
	const width = 30
	result := WrapLabelled("    Output file: ", "/a/very/long/path/to/an/export/file.json", width)

	s.True(strings.HasPrefix(result, "    Output file:"), result)
	for _, line := range strings.Split(result, "\n") {
		s.LessOrEqual(lipgloss.Width(line), width, line)
	}
}

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
