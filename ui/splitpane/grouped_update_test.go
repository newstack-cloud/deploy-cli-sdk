package splitpane

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// groupingGrouper mimics the abstract resource grouping used by the deploy and
// inspect UIs: it synthesises a group header that is not one of the flat items,
// and nests the real items beneath it while the header is expanded.
type groupingGrouper struct {
	groupID string
}

func (g *groupingGrouper) GroupItems(items []Item, isExpanded func(id string) bool) []Section {
	header := &mockItem{
		id:         g.groupID,
		name:       "[celerity/handler] orderHandler",
		itemType:   "resource",
		expandable: true,
		children:   items,
	}

	sectionItems := []Item{header}
	if isExpanded(header.GetID()) {
		// The real groupers nest children a level below the header, which is
		// what makes them findable as descendants of it.
		for _, item := range items {
			sectionItems = append(sectionItems, &depthAdjusted{Item: item, depth: 1})
		}
	}
	return []Section{{Name: "Resources", Items: sectionItems}}
}

// depthAdjusted mirrors how the shared grouper re-reports a nested item's depth.
type depthAdjusted struct {
	Item
	depth int
}

func (d *depthAdjusted) GetDepth() int { return d.depth }
func (d *depthAdjusted) Unwrap() Item  { return d.Item }

type GroupedUpdateTestSuite struct {
	suite.Suite
	groupID string
	model   Model
	items   []Item
}

func TestGroupedUpdateTestSuite(t *testing.T) {
	suite.Run(t, new(GroupedUpdateTestSuite))
}

func (s *GroupedUpdateTestSuite) SetupTest() {
	s.groupID = "group:celerity/handler:orderHandler"
	s.items = []Item{
		&mockItem{id: "lambda", name: "lambda", itemType: "resource"},
		&mockItem{id: "role", name: "role", itemType: "resource"},
	}
	s.model = New(Config{SectionGrouper: &groupingGrouper{groupID: s.groupID}})
	s.model.SetItems(s.items)
}

func (s *GroupedUpdateTestSuite) Test_update_keeps_group_expanded() {
	s.model.expandedItems[s.groupID] = true

	s.model.UpdateItems(s.items)

	s.True(s.model.IsExpanded(s.groupID), "group collapsed by a routine item update")
	s.Len(s.model.visibleItems(), 3)
}

func (s *GroupedUpdateTestSuite) Test_update_keeps_a_group_child_selected() {
	s.model.expandedItems[s.groupID] = true
	s.model.selectedID = "role"
	s.model.resolveSelectedIndex()

	s.model.UpdateItems(s.items)

	s.Equal("role", s.model.SelectedID(), "selection jumped off the item the user picked")
	s.Equal(2, s.model.SelectedIndex())
}

func (s *GroupedUpdateTestSuite) Test_update_survives_repeated_events() {
	s.model.expandedItems[s.groupID] = true
	s.model.selectedID = "role"
	s.model.resolveSelectedIndex()

	// A deployment sends a steady stream of these.
	for range 10 {
		s.model.UpdateItems(s.items)
	}

	s.True(s.model.IsExpanded(s.groupID))
	s.Equal("role", s.model.SelectedID())
}

func (s *GroupedUpdateTestSuite) Test_update_drops_expansion_for_items_that_are_gone() {
	s.model.expandedItems[s.groupID] = true
	s.model.expandedItems["removedParent"] = true

	s.model.UpdateItems(s.items)

	s.True(s.model.IsExpanded(s.groupID))
	s.False(s.model.IsExpanded("removedParent"), "stale expansion state was retained")
}

func (s *GroupedUpdateTestSuite) Test_update_keeps_nested_children_expanded() {
	grandchild := &mockItem{id: "grandchild", name: "grandchild", itemType: "resource"}
	nested := &mockItem{
		id:         "nestedChild",
		name:       "nestedChild",
		itemType:   "child",
		expandable: true,
		children:   []Item{grandchild},
	}
	items := []Item{nested}

	model := New(Config{SectionGrouper: &groupingGrouper{groupID: s.groupID}})
	model.SetItems(items)
	model.expandedItems[s.groupID] = true
	// Expandable items nested under the group are never in the flat item slice.
	model.expandedItems["nestedChild"] = true

	model.UpdateItems(items)

	s.True(model.IsExpanded("nestedChild"))
}
