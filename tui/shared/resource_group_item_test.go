package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type ResourceGroupItemTestSuite struct {
	suite.Suite
}

func TestResourceGroupItemTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceGroupItemTestSuite))
}

func (s *ResourceGroupItemTestSuite) Test_GetID_returns_prefixed_id() {
	group := &ResourceGroupItem{Group: ResourceGroup{GroupType: "celerity/function", GroupName: "myFunc"}}
	s.Equal("group:celerity/function:myFunc", group.GetID())
}

func (s *ResourceGroupItemTestSuite) Test_GetName_returns_bracketed_type_and_name() {
	group := &ResourceGroupItem{Group: ResourceGroup{GroupType: "celerity/api", GroupName: "myApi"}}
	s.Equal("[celerity/api] myApi", group.GetName())
}

func (s *ResourceGroupItemTestSuite) Test_GetItemType_returns_resource() {
	group := &ResourceGroupItem{}
	s.Equal("resource", group.GetItemType())
}

func (s *ResourceGroupItemTestSuite) Test_IsExpandable_returns_true() {
	group := &ResourceGroupItem{}
	s.True(group.IsExpandable())
}

func (s *ResourceGroupItemTestSuite) Test_CanDrillDown_returns_false() {
	group := &ResourceGroupItem{}
	s.False(group.CanDrillDown())
}

func (s *ResourceGroupItemTestSuite) Test_GetChildren_includes_resources_and_links() {
	res1 := &mockGroupChild{name: "res1"}
	res2 := &mockGroupChild{name: "res2"}
	link := &mockGroupChild{name: "link1"}
	group := &ResourceGroupItem{
		Children:      []splitpane.Item{res1, res2},
		InternalLinks: []splitpane.Item{link},
	}
	children := group.GetChildren()
	s.Len(children, 3)
}

func (s *ResourceGroupItemTestSuite) Test_aggregate_icon_returns_worst_case() {
	children := []splitpane.Item{
		&mockGroupChild{icon: IconSuccess},
		&mockGroupChild{icon: IconFailed},
		&mockGroupChild{icon: IconPending},
	}
	group := &ResourceGroupItem{Children: children}
	s.Equal(IconFailed, group.GetIcon(false))
}

func (s *ResourceGroupItemTestSuite) Test_aggregate_icon_returns_no_change_for_all_no_change() {
	children := []splitpane.Item{
		&mockGroupChild{icon: IconNoChange},
		&mockGroupChild{icon: IconNoChange},
	}
	group := &ResourceGroupItem{Children: children}
	s.Equal(IconNoChange, group.GetIcon(false))
}

func (s *ResourceGroupItemTestSuite) Test_aggregate_action_returns_highest_priority() {
	children := []splitpane.Item{
		&mockGroupChild{action: string(ActionNoChange)},
		&mockGroupChild{action: string(ActionUpdate)},
		&mockGroupChild{action: string(ActionCreate)},
	}
	group := &ResourceGroupItem{Children: children}
	s.Equal(string(ActionUpdate), group.GetAction())
}

func (s *ResourceGroupItemTestSuite) Test_aggregate_action_returns_delete_over_update() {
	children := []splitpane.Item{
		&mockGroupChild{action: string(ActionUpdate)},
		&mockGroupChild{action: string(ActionDelete)},
	}
	group := &ResourceGroupItem{Children: children}
	s.Equal(string(ActionDelete), group.GetAction())
}

func (s *ResourceGroupItemTestSuite) Test_depth_adjusted_item_overrides_depth() {
	inner := &mockGroupChild{name: "inner", depth: 0}
	adjusted := &DepthAdjustedItem{Item: inner, AdjustedDepth: 2}
	s.Equal(2, adjusted.GetDepth())
	s.Equal("inner", adjusted.GetName())
}

func (s *ResourceGroupItemTestSuite) Test_depth_adjusted_item_unwrap() {
	inner := &mockGroupChild{name: "inner"}
	adjusted := &DepthAdjustedItem{Item: inner, AdjustedDepth: 1}
	s.Equal(inner, adjusted.Unwrap())
}

// --- mock item for testing ---

type mockGroupChild struct {
	name   string
	icon   string
	action string
	depth  int
}

func (m *mockGroupChild) GetID() string              { return m.name }
func (m *mockGroupChild) GetName() string             { return m.name }
func (m *mockGroupChild) GetIcon(bool) string         { return m.icon }
func (m *mockGroupChild) GetAction() string           { return m.action }
func (m *mockGroupChild) GetDepth() int               { return m.depth }
func (m *mockGroupChild) GetParentID() string         { return "" }
func (m *mockGroupChild) GetItemType() string         { return "resource" }
func (m *mockGroupChild) IsExpandable() bool          { return false }
func (m *mockGroupChild) CanDrillDown() bool          { return false }
func (m *mockGroupChild) GetChildren() []splitpane.Item { return nil }
