package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type SectionGrouperGroupingTestSuite struct {
	suite.Suite
	grouper *SectionGrouper
}

func TestSectionGrouperGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(SectionGrouperGroupingTestSuite))
}

func (s *SectionGrouperGroupingTestSuite) SetupTest() {
	s.grouper = &SectionGrouper{MaxExpandDepth: 3}
}

func (s *SectionGrouperGroupingTestSuite) Test_ungrouped_items_pass_through_unchanged() {
	items := []splitpane.Item{
		newMockResource("resA", nil),
		newMockResource("resB", nil),
		newMockLink("resA::resB", "resA", "resB"),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 2)
	s.Equal("Resources", sections[0].Name)
	s.Len(sections[0].Items, 2)
	s.Equal("Links", sections[1].Name)
	s.Len(sections[1].Items, 1)
}

func (s *SectionGrouperGroupingTestSuite) Test_grouped_resources_nest_under_group_header() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("aws/lambda/function", group),
		newMockResource("aws/iam/role", group),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 1)
	s.Equal("Resources", sections[0].Name)
	// Should have 1 group header (collapsed)
	s.Require().Len(sections[0].Items, 1)
	groupItem, ok := sections[0].Items[0].(*ResourceGroupItem)
	s.Require().True(ok)
	s.Equal("[celerity/function] myFunc", groupItem.GetName())
	s.Len(groupItem.Children, 2)
}

func (s *SectionGrouperGroupingTestSuite) Test_mixed_grouped_and_ungrouped() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("aws/lambda/function", group),
		newMockResource("ungroupedBucket", nil),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 1)
	// 1 group header + 1 ungrouped
	s.Len(sections[0].Items, 2)
}

func (s *SectionGrouperGroupingTestSuite) Test_expanded_group_injects_depth_adjusted_children() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("aws/lambda/function", group),
		newMockResource("aws/iam/role", group),
	}
	expanded := func(id string) bool { return id == "group:celerity/function:myFunc" }
	sections := s.grouper.GroupItems(items, expanded)
	s.Require().Len(sections, 1)
	// 1 group header + 2 depth-adjusted children
	s.Len(sections[0].Items, 3)
	// Children should be DepthAdjustedItem at depth 1
	adj, ok := sections[0].Items[1].(*DepthAdjustedItem)
	s.Require().True(ok)
	s.Equal(1, adj.GetDepth())
}

func (s *SectionGrouperGroupingTestSuite) Test_internal_links_added_to_group() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("resA", group),
		newMockResource("resB", group),
		newMockLink("resA::resB", "resA", "resB"),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	// Resources section with 1 group, no Links section since link is internal
	s.Require().Len(sections, 1)
	s.Equal("Resources", sections[0].Name)
	groupItem := sections[0].Items[0].(*ResourceGroupItem)
	s.Len(groupItem.InternalLinks, 1)
}

func (s *SectionGrouperGroupingTestSuite) Test_cross_group_links_in_separate_section() {
	groupA := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	groupB := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	items := []splitpane.Item{
		newMockResource("resA", groupA),
		newMockResource("resB", groupB),
		newMockLink("resA::resB", "resA", "resB"),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 2)
	s.Equal("Resources", sections[0].Name)
	s.Equal("Cross-group Links", sections[1].Name)
	s.Len(sections[1].Items, 1)
}

func (s *SectionGrouperGroupingTestSuite) Test_ungrouped_links_in_links_section() {
	items := []splitpane.Item{
		newMockResource("resA", nil),
		newMockResource("resB", nil),
		newMockLink("resA::resB", "resA", "resB"),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 2)
	s.Equal("Links", sections[1].Name)
}

func (s *SectionGrouperGroupingTestSuite) Test_multiple_groups_sorted_by_name() {
	groupB := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	groupA := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("lambda", groupA),
		newMockResource("apigw", groupB),
	}
	sections := s.grouper.GroupItems(items, noExpand)
	s.Require().Len(sections, 1)
	s.Len(sections[0].Items, 2)
	// Sorted by GetName(): "[celerity/api] myApi" < "[celerity/function] myFunc"
	s.Contains(sections[0].Items[0].GetName(), "myApi")
	s.Contains(sections[0].Items[1].GetName(), "myFunc")
}

// --- helpers ---

func noExpand(_ string) bool { return false }

type mockGroupableResource struct {
	mockGroupChild
	group *ResourceGroup
}

func (m *mockGroupableResource) GetResourceGroup() *ResourceGroup { return m.group }

func newMockResource(name string, group *ResourceGroup) splitpane.Item {
	return &mockGroupableResource{
		mockGroupChild: mockGroupChild{name: name, icon: IconPending, action: string(ActionNoChange)},
		group:          group,
	}
}

type mockClassifiableLink struct {
	mockGroupChild
	resA, resB string
}

func (m *mockClassifiableLink) GetItemType() string                    { return "link" }
func (m *mockClassifiableLink) GetLinkResourceNames() (string, string) { return m.resA, m.resB }

func newMockLink(name, resA, resB string) splitpane.Item {
	return &mockClassifiableLink{
		mockGroupChild: mockGroupChild{name: name, icon: IconPending, action: string(ActionNoChange)},
		resA:           resA,
		resB:           resB,
	}
}

func init() {
	// Ensure mock types satisfy the right interfaces.
	var _ GroupableItem = (*mockGroupableResource)(nil)
	var _ LinkClassifiable = (*mockClassifiableLink)(nil)
}
