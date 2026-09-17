package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type CollapsedSummaryTestSuite struct {
	suite.Suite
}

func TestCollapsedSummaryTestSuite(t *testing.T) {
	suite.Run(t, new(CollapsedSummaryTestSuite))
}

func (s *CollapsedSummaryTestSuite) Test_reports_hidden_resources_and_links() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{
			newMockResource("lambda", nil),
			newMockResource("role", nil),
		},
		Links: []splitpane.Item{
			newMockLink("lambda::role", "lambda", "role"),
		},
	}
	s.Equal("(2R 1L)", group.GetCollapsedSummary())
}

func (s *CollapsedSummaryTestSuite) Test_omits_links_when_there_are_none() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{newMockResource("bucket", nil)},
	}
	s.Equal("(1R)", group.GetCollapsedSummary())
}

func (s *CollapsedSummaryTestSuite) Test_empty_group_has_no_summary() {
	s.Equal("", (&ResourceGroupItem{}).GetCollapsedSummary())
}

func (s *CollapsedSummaryTestSuite) Test_groups_satisfy_the_summariser_interface() {
	// The pane only renders a summary for items that implement this.
	var _ splitpane.CollapsedSummariser = (*ResourceGroupItem)(nil)
}
