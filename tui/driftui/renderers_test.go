package driftui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type RenderersTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestRenderersTestSuite(t *testing.T) {
	suite.Run(t, new(RenderersTestSuite))
}

func (s *RenderersTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func (s *RenderersTestSuite) Test_renderResourceChanges_new_fields() {
	renderer := &DriftDetailsRenderer{}
	newVal := "new-tag-value"
	result := &container.ResourceReconcileResult{
		Changes: &provider.Changes{
			NewFields: []provider.FieldChange{
				{
					FieldPath: "spec.newTag",
					NewValue: &core.MappingNode{
						Scalar: &core.ScalarValue{StringValue: &newVal},
					},
				},
			},
		},
	}
	item := &DriftItem{
		Type:           DriftItemTypeResource,
		Name:           "myBucket",
		ResourceType:   "aws/s3/bucket",
		DriftType:      container.ReconciliationTypeDrift,
		Recommended:    container.ReconciliationActionAcceptExternal,
		ResourceResult: result,
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "spec.newTag")
	s.Contains(output, "new-tag-value")
	s.Contains(output, "+ spec.newTag")
}

func (s *RenderersTestSuite) Test_renderResourceChanges_removed_fields() {
	renderer := &DriftDetailsRenderer{}
	result := &container.ResourceReconcileResult{
		Changes: &provider.Changes{
			RemovedFields: []string{"spec.oldTag", "spec.deprecatedField"},
		},
	}
	item := &DriftItem{
		Type:           DriftItemTypeResource,
		Name:           "myBucket",
		ResourceType:   "aws/s3/bucket",
		DriftType:      container.ReconciliationTypeDrift,
		Recommended:    container.ReconciliationActionAcceptExternal,
		ResourceResult: result,
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "spec.oldTag")
	s.Contains(output, "spec.deprecatedField")
	s.Contains(output, "- spec.oldTag")
}

func (s *RenderersTestSuite) Test_renderResourceChanges_no_changes() {
	renderer := &DriftDetailsRenderer{}
	result := &container.ResourceReconcileResult{
		Changes: &provider.Changes{},
	}
	item := &DriftItem{
		Type:           DriftItemTypeResource,
		Name:           "myBucket",
		ResourceType:   "aws/s3/bucket",
		DriftType:      container.ReconciliationTypeDrift,
		Recommended:    container.ReconciliationActionAcceptExternal,
		ResourceResult: result,
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "No field changes detected")
}

func (s *RenderersTestSuite) Test_renderResourceChanges_all_field_types() {
	renderer := &DriftDetailsRenderer{}
	oldVal := "old"
	newVal := "new"
	addedVal := "added"
	result := &container.ResourceReconcileResult{
		Changes: &provider.Changes{
			ModifiedFields: []provider.FieldChange{
				{
					FieldPath: "spec.modified",
					PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldVal}},
					NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newVal}},
				},
			},
			NewFields: []provider.FieldChange{
				{
					FieldPath: "spec.added",
					NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &addedVal}},
				},
			},
			RemovedFields: []string{"spec.removed"},
		},
	}
	item := &DriftItem{
		Type:           DriftItemTypeResource,
		Name:           "myBucket",
		ResourceType:   "aws/s3/bucket",
		DriftType:      container.ReconciliationTypeDrift,
		Recommended:    container.ReconciliationActionAcceptExternal,
		ResourceResult: result,
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "spec.modified")
	s.Contains(output, "spec.added")
	s.Contains(output, "spec.removed")
}

func (s *RenderersTestSuite) Test_renderLinkDetails_with_link_data_updates() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:        DriftItemTypeLink,
		Name:        "myLink",
		DriftType:   container.ReconciliationTypeDrift,
		Recommended: container.ReconciliationActionAcceptExternal,
		LinkResult: &container.LinkReconcileResult{
			LinkName: "myLink",
			Type:     container.ReconciliationTypeDrift,
			LinkDataUpdates: map[string]*core.MappingNode{
				"connection.endpoint": nil,
				"auth.token":          nil,
			},
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Link Data Affected:")
	s.Contains(output, "connection.endpoint")
	s.Contains(output, "auth.token")
}

func (s *RenderersTestSuite) Test_renderLinkDetails_without_link_data_updates() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:        DriftItemTypeLink,
		Name:        "myLink",
		DriftType:   container.ReconciliationTypeDrift,
		Recommended: container.ReconciliationActionAcceptExternal,
		LinkResult: &container.LinkReconcileResult{
			LinkName: "myLink",
			Type:     container.ReconciliationTypeDrift,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.NotContains(output, "Link Data Affected:")
}

func (s *RenderersTestSuite) Test_renderLinkDetails_with_child_path() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:        DriftItemTypeLink,
		Name:        "myLink",
		ChildPath:   "childBlueprint",
		DriftType:   container.ReconciliationTypeDrift,
		Recommended: container.ReconciliationActionAcceptExternal,
		LinkResult: &container.LinkReconcileResult{
			LinkName:  "myLink",
			ChildPath: "childBlueprint",
			Type:      container.ReconciliationTypeDrift,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Child Path:")
	s.Contains(output, "childBlueprint")
}

func (s *RenderersTestSuite) Test_renderResourceMetadata_with_child_path() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "childRes",
		ResourceType: "aws/lambda/function",
		ChildPath:    "childBlueprint",
		DriftType:    container.ReconciliationTypeDrift,
		Recommended:  container.ReconciliationActionAcceptExternal,
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Child Path:")
	s.Contains(output, "childBlueprint")
}

func (s *RenderersTestSuite) Test_renderResourceMetadata_with_resource_id() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myBucket",
		ResourceType: "aws/s3/bucket",
		DriftType:    container.ReconciliationTypeDrift,
		Recommended:  container.ReconciliationActionAcceptExternal,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName: "myBucket",
			ResourceID:   "bucket-123",
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Resource ID:")
	s.Contains(output, "bucket-123")
}

func (s *RenderersTestSuite) Test_renderResourceMetadata_interrupted_resource_exists_true() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/ec2/instance",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionManualCleanupRequired,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName:   "myResource",
			ResourceExists: true,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Resource exists:")
	s.Contains(output, "Yes")
}

func (s *RenderersTestSuite) Test_renderResourceMetadata_interrupted_resource_exists_false() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/ec2/instance",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionManualCleanupRequired,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName:   "myResource",
			ResourceExists: false,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Resource exists:")
	s.Contains(output, "No")
}

func (s *RenderersTestSuite) Test_DriftFooterRenderer_RenderFooter_normal_mode() {
	renderer := &DriftFooterRenderer{Context: DriftContextDeploy}
	model := splitpane.New(splitpane.Config{})
	output := renderer.RenderFooter(&model, s.testStyles)
	s.Contains(output, "a")
	s.Contains(output, "accept external changes")
	s.Contains(output, "q")
	s.Contains(output, "quit")
	s.Contains(output, "--force")
}

func (s *RenderersTestSuite) Test_DriftFooterRenderer_RenderFooter_stage_context() {
	renderer := &DriftFooterRenderer{Context: DriftContextStage}
	model := splitpane.New(splitpane.Config{})
	output := renderer.RenderFooter(&model, s.testStyles)
	s.Contains(output, "--skip-drift-check")
}

func (s *RenderersTestSuite) Test_DriftFooterRenderer_RenderFooter_deploy_stage_context() {
	renderer := &DriftFooterRenderer{Context: DriftContextDeployStage}
	model := splitpane.New(splitpane.Config{})
	output := renderer.RenderFooter(&model, s.testStyles)
	s.Contains(output, "--skip-drift-check")
}

func (s *RenderersTestSuite) Test_DriftFooterRenderer_RenderFooter_destroy_context() {
	renderer := &DriftFooterRenderer{Context: DriftContextDestroy}
	model := splitpane.New(splitpane.Config{})
	output := renderer.RenderFooter(&model, s.testStyles)
	s.Contains(output, "--force")
}

func (s *RenderersTestSuite) Test_DriftFooterRenderer_RenderFooter_unknown_context_no_hint() {
	renderer := &DriftFooterRenderer{Context: "unknown"}
	model := splitpane.New(splitpane.Config{})
	output := renderer.RenderFooter(&model, s.testStyles)
	// Unknown context should not show Hint line
	s.NotContains(output, "Hint:")
}

func (s *RenderersTestSuite) Test_DriftDetailsRenderer_RenderDetails_non_drift_item() {
	renderer := &DriftDetailsRenderer{}
	// Use a non-*DriftItem that still implements splitpane.Item
	item := &mockSplitpaneItem{}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Unknown item type")
}

// mockSplitpaneItem is a minimal splitpane.Item implementation for testing.
type mockSplitpaneItem struct{}

func (m *mockSplitpaneItem) GetID() string              { return "mock" }
func (m *mockSplitpaneItem) GetName() string            { return "mock" }
func (m *mockSplitpaneItem) GetIcon(selected bool) string { return "○" }
func (m *mockSplitpaneItem) GetIconStyled(s *styles.Styles, styled bool) string {
	return "○"
}
func (m *mockSplitpaneItem) GetAction() string               { return "" }
func (m *mockSplitpaneItem) GetDepth() int                   { return 0 }
func (m *mockSplitpaneItem) GetParentID() string             { return "" }
func (m *mockSplitpaneItem) GetItemType() string             { return "mock" }
func (m *mockSplitpaneItem) IsExpandable() bool              { return false }
func (m *mockSplitpaneItem) CanDrillDown() bool              { return false }
func (m *mockSplitpaneItem) GetChildren() []splitpane.Item   { return nil }

func (s *RenderersTestSuite) Test_renderInterruptedState_with_external_state() {
	renderer := &DriftDetailsRenderer{}
	stateVal := "running"
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/ec2/instance",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionUpdateStatus,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName:   "myResource",
			ResourceExists: true,
			ExternalState: &core.MappingNode{
				Scalar: &core.ScalarValue{StringValue: &stateVal},
			},
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "External state:")
	s.Contains(output, "running")
}

func (s *RenderersTestSuite) Test_renderInterruptedState_no_external_state() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/ec2/instance",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionUpdateStatus,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName:   "myResource",
			ResourceExists: true,
			ExternalState:  nil,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "No external state available")
}

func (s *RenderersTestSuite) Test_renderChildDetails_with_link_children() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:      DriftItemTypeChild,
		Name:      "childBlueprint",
		ChildPath: "child1",
		Children: []*DriftItem{
			{Type: DriftItemTypeLink, Name: "link1"},
			{Type: DriftItemTypeLink, Name: "link2"},
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "2 links with drift")
}

func (s *RenderersTestSuite) Test_renderChildDetails_with_nested_children() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:      DriftItemTypeChild,
		Name:      "parentChild",
		ChildPath: "parentChild",
		Children: []*DriftItem{
			{Type: DriftItemTypeChild, Name: "nestedChild1"},
			{Type: DriftItemTypeChild, Name: "nestedChild2"},
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "2 nested child blueprints")
}

func (s *RenderersTestSuite) Test_renderChildDetails_shows_expand_hint_at_max_depth() {
	renderer := &DriftDetailsRenderer{
		MaxExpandDepth:       2,
		NavigationStackDepth: 0,
	}
	item := &DriftItem{
		Type:      DriftItemTypeChild,
		Name:      "childBlueprint",
		ChildPath: "child1",
		Depth:     2, // At max depth
		Children: []*DriftItem{
			{Type: DriftItemTypeResource, Name: "res1"},
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Press enter to inspect this child blueprint")
}

func (s *RenderersTestSuite) Test_renderDetails_manual_cleanup_instructions() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "brokenResource",
		ResourceType: "aws/ec2/instance",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionManualCleanupRequired,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceName:   "brokenResource",
			ResourceExists: false,
		},
	}
	output := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(output, "Manual cleanup required")
	s.Contains(output, "External state could not be retrieved")
}

func (s *RenderersTestSuite) Test_DriftSectionGrouper_expands_children_when_expanded() {
	grouper := &DriftSectionGrouper{MaxExpandDepth: 2}
	child := &DriftItem{
		Type:  DriftItemTypeChild,
		Name:  "child1",
		Depth: 0,
		Children: []*DriftItem{
			{Type: DriftItemTypeResource, Name: "childRes", Depth: 1, ParentChild: "child1"},
		},
	}
	items := []splitpane.Item{child}
	isExpanded := func(id string) bool { return id == "child1" }
	sections := grouper.GroupItems(items, isExpanded)

	// Should have one section: Child Blueprints containing both child and its expanded resource
	s.Len(sections, 1)
	s.Equal("Child Blueprints", sections[0].Name)
	// 2 items: the child summary + the expanded child resource
	s.Len(sections[0].Items, 2)
}

func (s *RenderersTestSuite) Test_DriftSectionGrouper_nested_items_with_parent_child_go_to_children_section() {
	grouper := &DriftSectionGrouper{}
	items := []splitpane.Item{
		&DriftItem{Type: DriftItemTypeResource, Name: "nestedRes", Depth: 1, ParentChild: "child1"},
	}
	sections := grouper.GroupItems(items, nil)
	s.Len(sections, 1)
	s.Equal("Child Blueprints", sections[0].Name)
}
