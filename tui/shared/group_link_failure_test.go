package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

// A link between two resources of the same abstract resource is folded into
// that group, so the group header is the only thing on screen reporting it
// while the group is collapsed. If the header ignores links, a failed link is
// invisible in a tree that otherwise reads as healthy.
type GroupLinkFailureTestSuite struct {
	suite.Suite
}

func TestGroupLinkFailureTestSuite(t *testing.T) {
	suite.Run(t, new(GroupLinkFailureTestSuite))
}

func (s *GroupLinkFailureTestSuite) Test_a_failed_internal_link_shows_on_the_group() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{
			iconItem("lambda", IconSuccess),
			iconItem("role", IconSuccess),
		},
		Links: []splitpane.Item{iconItem("lambda::role", IconFailed)},
	}

	s.Equal(IconFailed, group.GetIcon(false), "the group reports its resources as healthy")
}

func (s *GroupLinkFailureTestSuite) Test_resource_failures_still_win_over_healthy_links() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{iconItem("lambda", IconFailed)},
		Links:    []splitpane.Item{iconItem("lambda::role", IconSuccess)},
	}

	s.Equal(IconFailed, group.GetIcon(false))
}

func (s *GroupLinkFailureTestSuite) Test_a_group_with_no_failures_stays_healthy() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{iconItem("lambda", IconSuccess)},
		Links:    []splitpane.Item{iconItem("lambda::role", IconSuccess)},
	}

	s.Equal(IconSuccess, group.GetIcon(false))
}

func (s *GroupLinkFailureTestSuite) Test_an_internal_link_action_counts_towards_the_group() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{actionItem("lambda", string(ActionNoChange))},
		Links:    []splitpane.Item{actionItem("lambda::role", string(ActionCreate))},
	}

	s.Equal(string(ActionCreate), group.GetAction(), "a link being created is a change to the group")
}
