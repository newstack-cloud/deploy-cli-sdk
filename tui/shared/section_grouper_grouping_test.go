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
	// One section holding the group, no Links section since the link is internal
	s.Require().Len(sections, 1)
	s.Equal("Resources & Links", sections[0].Name)
	groupItem := sections[0].Items[0].(*ResourceGroupItem)
	s.Len(groupItem.Links, 1)
}

// A link leaving one abstract resource for another belongs with the resource it
// runs out of, rather than in a flat list of every crossing in the deployment.
func (s *SectionGrouperGroupingTestSuite) Test_a_cross_group_link_nests_under_its_source() {
	fnGroup := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	apiGroup := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	items := []splitpane.Item{
		newMockResource("lambda", fnGroup),
		newMockResource("gateway", apiGroup),
		newMockLink("lambda::gateway", "lambda", "gateway"),
	}

	sections := s.grouper.GroupItems(items, noExpand)

	s.Require().Len(sections, 1, "there should be no separate cross-group section")
	s.Equal("Resources & Links", sections[0].Name)

	groups := map[string]*ResourceGroupItem{}
	for _, item := range sections[0].Items {
		group := item.(*ResourceGroupItem)
		groups[group.Group.GroupName] = group
	}

	s.Len(groups["myFunc"].Links, 1, "the link should sit with the resource it runs out of")
	s.Empty(groups["myApi"].Links, "the destination group should not also hold it")
}

func (s *SectionGrouperGroupingTestSuite) Test_a_link_out_of_an_ungrouped_resource_falls_back_to_its_destination() {
	fnGroup := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("lambda", fnGroup),
		newMockResource("standaloneBucket", nil),
		newMockLink("standaloneBucket::lambda", "standaloneBucket", "lambda"),
	}

	sections := s.grouper.GroupItems(items, noExpand)

	s.Require().Len(sections, 1, "the link has a group to live under, so no Links section")
	for _, item := range sections[0].Items {
		if group, ok := item.(*ResourceGroupItem); ok {
			s.Len(group.Links, 1, "the link should fall back to the group it points into")
		}
	}
}

func (s *SectionGrouperGroupingTestSuite) Test_a_cross_group_link_is_visible_when_its_group_is_expanded() {
	fnGroup := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	apiGroup := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	items := []splitpane.Item{
		newMockResource("lambda", fnGroup),
		newMockResource("gateway", apiGroup),
		newMockLink("lambda::gateway", "lambda", "gateway"),
	}

	sections := s.grouper.GroupItems(items, expandAll)

	// Inside the group that owns it, the link names only its far end.
	s.Equal(
		[]string{
			"[celerity/api] myApi",
			"gateway",
			"[celerity/function] myFunc",
			"lambda",
			"→ gateway",
		},
		itemNames(sections[0].Items),
	)
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

func (s *SectionGrouperGroupingTestSuite) Test_expanded_group_children_stay_under_own_header() {
	apiGroup := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	fnGroup := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("apiGateway", apiGroup),
		newMockResource("apiStage", apiGroup),
		newMockResource("lambdaFunction", fnGroup),
		newMockResource("lambdaRole", fnGroup),
	}
	sections := s.grouper.GroupItems(items, expandAll)
	s.Require().Len(sections, 1)
	s.Equal(
		[]string{
			"[celerity/api] myApi",
			"apiGateway",
			"apiStage",
			"[celerity/function] myFunc",
			"lambdaFunction",
			"lambdaRole",
		},
		itemNames(sections[0].Items),
	)
}

func (s *SectionGrouperGroupingTestSuite) Test_ungrouped_resources_sort_alongside_group_headers() {
	fnGroup := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("lambdaFunction", fnGroup),
		// sorts between the group header and its child
		newMockResource("Zbucket", nil),
	}
	sections := s.grouper.GroupItems(items, expandAll)
	s.Require().Len(sections, 1)
	// The ungrouped resource must not land between the header and its child.
	s.Equal(
		[]string{"Zbucket", "[celerity/function] myFunc", "lambdaFunction"},
		itemNames(sections[0].Items),
	)
}

func (s *SectionGrouperGroupingTestSuite) Test_expanded_group_renders_internal_links_as_rows() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("lambdaFunction", group),
		newMockResource("lambdaRole", group),
		newMockLink("lambdaFunction::lambdaRole", "lambdaFunction", "lambdaRole"),
	}
	sections := s.grouper.GroupItems(items, expandAll)
	s.Require().Len(sections, 1)
	s.Equal(
		[]string{
			"[celerity/function] myFunc",
			"lambdaFunction",
			"lambdaRole",
			"lambdaFunction::lambdaRole",
		},
		itemNames(sections[0].Items),
	)
	// The link row is nested under the group, not at top level.
	s.Equal(1, sections[0].Items[3].GetDepth())
}

// --- helpers ---

func expandAll(_ string) bool { return true }

func itemNames(items []splitpane.Item) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.GetName())
	}
	return names
}

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

// --- ambient abstract resources ---

// Everything in a deployment links out of the VPC, so grouping those links
// under it would bury the application structure and tell you nothing about the
// VPC you did not already know. They belong with the component at the far end.
func (s *SectionGrouperGroupingTestSuite) Test_links_out_of_an_ambient_resource_group_under_the_far_end() {
	vpcGroup := &ResourceGroup{
		GroupName: "appVpc", GroupType: "celerity/vpc", Role: ResourceRoleAmbient,
	}
	handlerGroup := &ResourceGroup{GroupName: "createOrder", GroupType: "celerity/handler"}
	items := []splitpane.Item{
		newMockResource("appVpc_vpc", vpcGroup),
		newMockResource("createOrder_lambda", handlerGroup),
		newMockLink("appVpc_vpc::createOrder_lambda", "appVpc_vpc", "createOrder_lambda"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Empty(groups["appVpc"].Links, "the VPC should not collect the links it plumbs")
	s.Len(groups["createOrder"].Links, 1, "the link belongs with the component it reaches")
}

// An API routing to a handler is application wiring, and reading the API to see
// what it routes to is useful, so the source keeps those.
func (s *SectionGrouperGroupingTestSuite) Test_links_out_of_a_non_ambient_resource_stay_with_the_source() {
	apiGroup := &ResourceGroup{GroupName: "ordersApi", GroupType: "celerity/api"}
	handlerGroup := &ResourceGroup{GroupName: "createOrder", GroupType: "celerity/handler"}
	items := []splitpane.Item{
		newMockResource("ordersApi_http_api", apiGroup),
		newMockResource("createOrder_lambda", handlerGroup),
		newMockLink("ordersApi_http_api::createOrder_lambda", "ordersApi_http_api", "createOrder_lambda"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Len(groups["ordersApi"].Links, 1, "an API should show what it routes to")
	s.Empty(groups["createOrder"].Links)
}

func (s *SectionGrouperGroupingTestSuite) Test_a_link_between_two_ambient_resources_stays_with_the_source() {
	vpcA := &ResourceGroup{GroupName: "vpcA", GroupType: "celerity/vpc", Role: ResourceRoleAmbient}
	vpcB := &ResourceGroup{GroupName: "vpcB", GroupType: "celerity/vpc", Role: ResourceRoleAmbient}
	items := []splitpane.Item{
		newMockResource("vpcA_vpc", vpcA),
		newMockResource("vpcB_vpc", vpcB),
		newMockLink("vpcA_vpc::vpcB_vpc", "vpcA_vpc", "vpcB_vpc"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Len(groups["vpcA"].Links, 1, "with no better end to pick, the source keeps it")
	s.Empty(groups["vpcB"].Links)
}

// The tree knows nothing about VPCs: any abstract resource the transformer
// declares as ambient behaves the same way.
func (s *SectionGrouperGroupingTestSuite) Test_ambient_is_taken_from_the_declared_role_not_the_type() {
	ambientGroup := &ResourceGroup{
		GroupName: "sharedMesh", GroupType: "vendor/mesh", Role: ResourceRoleAmbient,
	}
	handlerGroup := &ResourceGroup{GroupName: "createOrder", GroupType: "celerity/handler"}
	items := []splitpane.Item{
		newMockResource("sharedMesh_mesh", ambientGroup),
		newMockResource("createOrder_lambda", handlerGroup),
		newMockLink("sharedMesh_mesh::createOrder_lambda", "sharedMesh_mesh", "createOrder_lambda"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Empty(groups["sharedMesh"].Links)
	s.Len(groups["createOrder"].Links, 1)
}

// A resource with no declared role is a component, which is what everything
// produced before the annotation existed reads as.
func (s *SectionGrouperGroupingTestSuite) Test_an_undeclared_role_behaves_as_a_component() {
	unmarked := &ResourceGroup{GroupName: "appVpc", GroupType: "celerity/vpc"}
	handlerGroup := &ResourceGroup{GroupName: "createOrder", GroupType: "celerity/handler"}
	items := []splitpane.Item{
		newMockResource("appVpc_vpc", unmarked),
		newMockResource("createOrder_lambda", handlerGroup),
		newMockLink("appVpc_vpc::createOrder_lambda", "appVpc_vpc", "createOrder_lambda"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Len(groups["appVpc"].Links, 1, "without a declared role the source keeps its links")
	s.Empty(groups["createOrder"].Links)
}

func (s *SectionGrouperGroupingTestSuite) groupsByName(
	sections []splitpane.Section,
) map[string]*ResourceGroupItem {
	groups := map[string]*ResourceGroupItem{}
	for _, section := range sections {
		for _, item := range section.Items {
			if group, ok := item.(*ResourceGroupItem); ok {
				groups[group.Group.GroupName] = group
			}
		}
	}
	return groups
}

// The point of the ambient rule is that these links do not belong under the
// ambient resource. When the far end has no group of its own there is nowhere
// better to put the link, and leaving it loose is preferable to collecting it
// under the very resource it was being kept away from.
func (s *SectionGrouperGroupingTestSuite) Test_an_ambient_link_to_an_ungrouped_resource_is_not_kept_by_the_source() {
	ambient := &ResourceGroup{
		GroupName: "appVpc", GroupType: "celerity/vpc", Role: ResourceRoleAmbient,
	}
	items := []splitpane.Item{
		newMockResource("appVpc_vpc", ambient),
		newMockResource("plainBucket", nil),
		newMockLink("appVpc_vpc::plainBucket", "appVpc_vpc", "plainBucket"),
	}

	sections := s.grouper.GroupItems(items, noExpand)
	groups := s.groupsByName(sections)

	s.Empty(groups["appVpc"].Links, "the ambient resource collected the link anyway")

	var linkSection *splitpane.Section
	for i := range sections {
		if sections[i].Name == "Links" {
			linkSection = &sections[i]
		}
	}
	s.Require().NotNil(linkSection, "the link should be listed at the top level")
	s.Len(linkSection.Items, 1)
}

// A link between two of the ambient resource's own concrete resources is
// genuinely about it, and there is nowhere better for it to go.
func (s *SectionGrouperGroupingTestSuite) Test_an_ambient_resources_own_internal_links_stay_with_it() {
	ambient := &ResourceGroup{
		GroupName: "appVpc", GroupType: "celerity/vpc", Role: ResourceRoleAmbient,
	}
	items := []splitpane.Item{
		newMockResource("appVpc_vpc", ambient),
		newMockResource("appVpc_subnet", ambient),
		newMockLink("appVpc_vpc::appVpc_subnet", "appVpc_vpc", "appVpc_subnet"),
	}

	groups := s.groupsByName(s.grouper.GroupItems(items, noExpand))

	s.Len(groups["appVpc"].Links, 1)
}

// --- section naming ---

func (s *SectionGrouperGroupingTestSuite) Test_the_section_is_named_for_what_it_holds() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}

	// Groups holding only resources.
	sections := s.grouper.GroupItems([]splitpane.Item{
		newMockResource("lambda", group),
		newMockResource("role", group),
	}, noExpand)
	s.Equal("Resources", sections[0].Name)

	// The same groups once they carry a link as well.
	sections = s.grouper.GroupItems([]splitpane.Item{
		newMockResource("lambda", group),
		newMockResource("role", group),
		newMockLink("lambda::role", "lambda", "role"),
	}, noExpand)
	s.Equal("Resources & Links", sections[0].Name)
}

func (s *SectionGrouperGroupingTestSuite) Test_ungrouped_resources_alone_are_just_resources() {
	// Without abstract resources nothing is folded into the section, so links
	// stay in a section of their own and the title should not claim otherwise.
	sections := s.grouper.GroupItems([]splitpane.Item{
		newMockResource("bucket", nil),
		newMockResource("table", nil),
		newMockLink("bucket::table", "bucket", "table"),
	}, noExpand)

	s.Require().Len(sections, 2)
	s.Equal("Resources", sections[0].Name)
	s.Equal("Links", sections[1].Name)
}

// The title describes what the groups own, so it stays put as they are opened
// and closed rather than changing under the user.
func (s *SectionGrouperGroupingTestSuite) Test_the_title_does_not_change_with_expansion() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	items := []splitpane.Item{
		newMockResource("lambda", group),
		newMockResource("role", group),
		newMockLink("lambda::role", "lambda", "role"),
	}

	collapsed := s.grouper.GroupItems(items, noExpand)
	expanded := s.grouper.GroupItems(items, expandAll)

	s.Equal("Resources & Links", collapsed[0].Name)
	s.Equal(collapsed[0].Name, expanded[0].Name)
}
