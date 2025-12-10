package strings

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StringsSuite struct {
	suite.Suite
}

// FromPointer tests

func (s *StringsSuite) Test_from_pointer_with_nil() {
	result := FromPointer(nil)
	s.Equal("", result)
}

func (s *StringsSuite) Test_from_pointer_with_value() {
	value := "hello"
	result := FromPointer(&value)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_from_pointer_with_empty_string() {
	value := ""
	result := FromPointer(&value)
	s.Equal("", result)
}

// TruncateString tests

func (s *StringsSuite) Test_truncate_string_shorter_than_max() {
	result := TruncateString("hello", 10)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_truncate_string_equal_to_max() {
	result := TruncateString("hello", 5)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_truncate_string_longer_than_max() {
	result := TruncateString("hello world", 8)
	s.Equal("hello...", result)
}

func (s *StringsSuite) Test_truncate_string_empty() {
	result := TruncateString("", 10)
	s.Equal("", result)
}

// WrapText tests

func (s *StringsSuite) Test_wrap_text_shorter_than_width() {
	result := WrapText("hello", 10)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_wrap_text_equal_to_width() {
	result := WrapText("hello", 5)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_wrap_text_single_wrap() {
	result := WrapText("hello world", 6)
	s.Equal("hello\nworld", result)
}

func (s *StringsSuite) Test_wrap_text_multiple_wraps() {
	result := WrapText("one two three four", 8)
	s.Equal("one two\nthree\nfour", result)
}

func (s *StringsSuite) Test_wrap_text_zero_width() {
	result := WrapText("hello", 0)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_wrap_text_negative_width() {
	result := WrapText("hello", -5)
	s.Equal("hello", result)
}

func (s *StringsSuite) Test_wrap_text_empty_string() {
	result := WrapText("", 10)
	s.Equal("", result)
}

func (s *StringsSuite) Test_wrap_text_preserves_word_boundaries() {
	result := WrapText("the quick brown fox", 10)
	s.Equal("the quick\nbrown fox", result)
}

func TestStringsSuite(t *testing.T) {
	suite.Run(t, new(StringsSuite))
}
