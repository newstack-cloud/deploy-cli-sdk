package shared

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LinkNamingTestSuite struct {
	suite.Suite
}

func TestLinkNamingTestSuite(t *testing.T) {
	suite.Run(t, new(LinkNamingTestSuite))
}

func (s *LinkNamingTestSuite) Test_a_link_reads_as_a_direction() {
	s.Equal("queue → dlq", FormatLinkName("queue", "dlq"))
	s.Equal("queue → dlq", FormatLogicalLinkName("queue::dlq"))
}

func (s *LinkNamingTestSuite) Test_one_sided_links_show_what_there_is() {
	s.Equal("queue", FormatLinkName("queue", ""))
	s.Equal("dlq", FormatLinkName("", "dlq"))
	s.Equal("", FormatLinkName("", ""))
}

func (s *LinkNamingTestSuite) Test_a_name_that_is_not_a_link_is_left_alone() {
	s.Equal("justAResource", FormatLogicalLinkName("justAResource"))
}

func (s *LinkNamingTestSuite) Test_naming_from_one_end() {
	s.Equal("→ dlq", LinkToName("dlq"))
	s.Equal("← queue", LinkFromName("queue"))
}

// The logical name stays the identifier: it is what the engine reports, what
// headless output prints, and what expansion and selection are keyed on.
func (s *LinkNamingTestSuite) Test_the_logical_name_is_untouched() {
	s.Equal("::", LinkNameSeparator)
}
