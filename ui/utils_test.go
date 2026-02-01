package ui

import (
	"testing"

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

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
