package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type GroupAggregationTestSuite struct {
	suite.Suite
}

func TestGroupAggregationTestSuite(t *testing.T) {
	suite.Run(t, new(GroupAggregationTestSuite))
}

func (s *GroupAggregationTestSuite) groupOf(children ...splitpane.Item) *ResourceGroupItem {
	return &ResourceGroupItem{Children: children}
}

func (s *GroupAggregationTestSuite) Test_reports_the_most_significant_action() {
	group := s.groupOf(
		actionItem("lambda", string(ActionCreate)),
		actionItem("role", string(ActionUpdate)),
		actionItem("policy", string(ActionNoChange)),
	)
	s.Equal(string(ActionUpdate), group.GetAction())
}

func (s *GroupAggregationTestSuite) Test_reports_no_change_only_when_a_child_says_so() {
	group := s.groupOf(
		actionItem("lambda", string(ActionNoChange)),
		actionItem("role", string(ActionNoChange)),
	)
	s.Equal(string(ActionNoChange), group.GetAction())
}

func (s *GroupAggregationTestSuite) Test_children_without_an_action_are_not_reported_as_no_change() {
	// Items built from a deploy event rather than the change set carry no
	// action. Reporting "NO CHANGE" for them claims the group is being left
	// alone, which on a brand new deployment is the opposite of the truth.
	group := s.groupOf(
		actionItem("lambda", ""),
		actionItem("role", ""),
	)
	s.Equal("", group.GetAction())
}

func (s *GroupAggregationTestSuite) Test_a_child_with_an_action_still_wins_over_those_without() {
	group := s.groupOf(
		actionItem("lambda", ""),
		actionItem("role", string(ActionCreate)),
	)
	s.Equal(string(ActionCreate), group.GetAction())
}

func (s *GroupAggregationTestSuite) Test_inspect_mode_groups_carry_no_action_badge() {
	// Inspect mode uses an empty action for every item.
	group := s.groupOf(
		actionItem("lambda", string(ActionInspect)),
		actionItem("role", string(ActionInspect)),
	)
	s.Equal("", group.GetAction())
}

func (s *GroupAggregationTestSuite) Test_icon_falls_back_to_pending_when_nothing_is_known() {
	group := s.groupOf(iconItem("lambda", "?"), iconItem("role", "?"))
	s.Equal(IconPending, group.GetIcon(false))
}

func (s *GroupAggregationTestSuite) Test_icon_reports_the_most_severe_child() {
	group := s.groupOf(iconItem("lambda", IconSuccess), iconItem("role", IconFailed))
	s.Equal(IconFailed, group.GetIcon(false))
}

func (s *GroupAggregationTestSuite) Test_icon_reports_no_change_when_children_say_so() {
	group := s.groupOf(iconItem("lambda", IconNoChange), iconItem("role", IconNoChange))
	s.Equal(IconNoChange, group.GetIcon(false))
}

func actionItem(name, action string) splitpane.Item {
	return &mockGroupableResource{
		mockGroupChild: mockGroupChild{name: name, icon: IconPending, action: action},
	}
}

func iconItem(name, icon string) splitpane.Item {
	return &mockGroupableResource{
		mockGroupChild: mockGroupChild{name: name, icon: icon, action: string(ActionCreate)},
	}
}
