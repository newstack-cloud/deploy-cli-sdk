package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type UnwrapItemTestSuite struct {
	suite.Suite
}

func TestUnwrapItemTestSuite(t *testing.T) {
	suite.Run(t, new(UnwrapItemTestSuite))
}

func (s *UnwrapItemTestSuite) Test_returns_the_item_under_a_display_wrapper() {
	inner := newMockResource("lambda", nil)
	wrapped := &DepthAdjustedItem{Item: inner, AdjustedDepth: 1}

	s.Same(inner, UnwrapItem(wrapped))
}

func (s *UnwrapItemTestSuite) Test_passes_unwrapped_items_through() {
	item := newMockResource("lambda", nil)
	s.Same(item, UnwrapItem(item))
}

func (s *UnwrapItemTestSuite) Test_handles_a_nil_item() {
	s.Nil(UnwrapItem(nil))
}

// Selecting a resource inside an expanded group has to yield the concrete item
// type. Asserting on the item straight from the pane fails for grouped
// resources, which is what stopped the spec view from opening for them.
func (s *UnwrapItemTestSuite) Test_grouped_children_assert_to_their_concrete_type() {
	group := &ResourceGroup{GroupName: "createOrder", GroupType: "celerity/handler"}
	items := []splitpane.Item{
		newMockResource("createOrder_lambda", group),
		newMockResource("createOrder_role", group),
	}

	grouper := &SectionGrouper{MaxExpandDepth: 3}
	sections := grouper.GroupItems(items, func(string) bool { return true })
	s.Require().Len(sections, 1)

	// Index 0 is the group header; its children follow.
	child := sections[0].Items[1]
	_, direct := child.(*mockGroupableResource)
	s.False(direct, "grouped children are expected to be wrapped for display")

	_, unwrapped := UnwrapItem(child).(*mockGroupableResource)
	s.True(unwrapped, "unwrapping should recover the concrete item")
}
