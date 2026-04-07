package stageui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type StageExportsTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestStageExportsTestSuite(t *testing.T) {
	suite.Run(t, new(StageExportsTestSuite))
}

func (s *StageExportsTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *StageExportsTestSuite) Test_GetID_root_item_returns_root() {
	item := &StageExportsInstanceItem{Name: "root-inst", Path: ""}
	s.Equal("root", item.GetID())
}

func (s *StageExportsTestSuite) Test_GetID_child_item_returns_path() {
	item := &StageExportsInstanceItem{Name: "child-a", Path: "child-a"}
	s.Equal("child-a", item.GetID())
}

func (s *StageExportsTestSuite) Test_GetName_returns_name() {
	item := &StageExportsInstanceItem{Name: "my-instance"}
	s.Equal("my-instance", item.GetName())
}

func (s *StageExportsTestSuite) Test_GetIcon_new_exports_returns_create_icon() {
	item := &StageExportsInstanceItem{NewCount: 2}
	s.Equal("✓", item.GetIcon(false))
}

func (s *StageExportsTestSuite) Test_GetIcon_modified_exports_returns_update_icon() {
	item := &StageExportsInstanceItem{ModifiedCount: 1}
	s.Equal("±", item.GetIcon(false))
}

func (s *StageExportsTestSuite) Test_GetIcon_removed_exports_returns_delete_icon() {
	item := &StageExportsInstanceItem{RemovedCount: 3}
	s.Equal("-", item.GetIcon(false))
}

func (s *StageExportsTestSuite) Test_GetIcon_no_changes_returns_no_change_icon() {
	item := &StageExportsInstanceItem{}
	s.Equal("○", item.GetIcon(false))
}

func (s *StageExportsTestSuite) Test_GetIcon_new_count_takes_priority_over_modified() {
	item := &StageExportsInstanceItem{NewCount: 1, ModifiedCount: 2, RemovedCount: 3}
	s.Equal("✓", item.GetIcon(false))
}

func (s *StageExportsTestSuite) Test_GetAction_returns_new_count() {
	item := &StageExportsInstanceItem{NewCount: 2}
	s.Contains(item.GetAction(), "2 new")
}

func (s *StageExportsTestSuite) Test_GetAction_returns_modified_count() {
	item := &StageExportsInstanceItem{ModifiedCount: 3}
	s.Contains(item.GetAction(), "3 modified")
}

func (s *StageExportsTestSuite) Test_GetAction_returns_removed_count() {
	item := &StageExportsInstanceItem{RemovedCount: 1}
	s.Contains(item.GetAction(), "1 removed")
}

func (s *StageExportsTestSuite) Test_GetAction_returns_combined_changes() {
	item := &StageExportsInstanceItem{NewCount: 1, ModifiedCount: 2, RemovedCount: 3}
	action := item.GetAction()
	s.Contains(action, "1 new")
	s.Contains(action, "2 modified")
	s.Contains(action, "3 removed")
}

func (s *StageExportsTestSuite) Test_GetAction_returns_unchanged_when_no_changes() {
	item := &StageExportsInstanceItem{UnchangedCount: 5}
	s.Contains(item.GetAction(), "5 unchanged")
}

func (s *StageExportsTestSuite) Test_GetAction_returns_no_exports_when_empty() {
	item := &StageExportsInstanceItem{}
	s.Equal("no exports", item.GetAction())
}

func (s *StageExportsTestSuite) Test_GetDepth_returns_depth() {
	item := &StageExportsInstanceItem{Depth: 2}
	s.Equal(2, item.GetDepth())
}

func (s *StageExportsTestSuite) Test_GetParentID_root_returns_empty() {
	item := &StageExportsInstanceItem{Path: ""}
	s.Equal("", item.GetParentID())
}

func (s *StageExportsTestSuite) Test_GetParentID_top_level_child_returns_root() {
	item := &StageExportsInstanceItem{Path: "child-a"}
	s.Equal("root", item.GetParentID())
}

func (s *StageExportsTestSuite) Test_GetParentID_nested_child_returns_parent_path() {
	item := &StageExportsInstanceItem{Path: "child-a/child-b"}
	s.Equal("child-a", item.GetParentID())
}

func (s *StageExportsTestSuite) Test_GetItemType_root_returns_empty() {
	item := &StageExportsInstanceItem{Depth: 0}
	s.Equal("", item.GetItemType())
}

func (s *StageExportsTestSuite) Test_GetItemType_child_returns_child() {
	item := &StageExportsInstanceItem{Depth: 1}
	s.Equal("child", item.GetItemType())
}

func (s *StageExportsTestSuite) Test_IsExpandable_returns_false() {
	item := &StageExportsInstanceItem{}
	s.False(item.IsExpandable())
}

func (s *StageExportsTestSuite) Test_CanDrillDown_returns_false() {
	item := &StageExportsInstanceItem{}
	s.False(item.CanDrillDown())
}

func (s *StageExportsTestSuite) Test_GetChildren_returns_nil() {
	item := &StageExportsInstanceItem{}
	s.Nil(item.GetChildren())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_true_for_new_count() {
	item := &StageExportsInstanceItem{NewCount: 1}
	s.True(item.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_true_for_modified_count() {
	item := &StageExportsInstanceItem{ModifiedCount: 1}
	s.True(item.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_true_for_removed_count() {
	item := &StageExportsInstanceItem{RemovedCount: 1}
	s.True(item.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_false_for_unchanged_only() {
	item := &StageExportsInstanceItem{UnchangedCount: 3}
	s.False(item.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_returns_nil_for_nil_changes() {
	result := BuildExportChangeHierarchy(nil, "root")
	s.Nil(result)
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_returns_root_item() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("value1")},
		},
	}
	items := BuildExportChangeHierarchy(bc, "my-instance")
	s.Len(items, 1)

	rootItem := items[0].(*StageExportsInstanceItem)
	s.Equal("my-instance", rootItem.Name)
	s.Equal("", rootItem.Path)
	s.Equal(0, rootItem.Depth)
	s.Equal(1, rootItem.NewCount)
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_uses_root_display_name_when_empty() {
	bc := &changes.BlueprintChanges{}
	items := BuildExportChangeHierarchy(bc, "")
	s.Len(items, 1)

	rootItem := items[0].(*StageExportsInstanceItem)
	s.Equal("(root)", rootItem.Name)
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_includes_child_items() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"rootExport": {NewValue: stringNode("v1")},
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"child-a": {
				NewExports: map[string]provider.FieldChange{
					"childExport": {NewValue: stringNode("v2")},
				},
			},
		},
	}

	items := BuildExportChangeHierarchy(bc, "my-instance")
	s.Len(items, 2)

	rootItem := items[0].(*StageExportsInstanceItem)
	s.Equal("my-instance", rootItem.Name)
	s.Equal(0, rootItem.Depth)
	s.Equal(1, rootItem.NewCount)

	childItem := items[1].(*StageExportsInstanceItem)
	s.Equal("child-a", childItem.Name)
	s.Equal("child-a", childItem.Path)
	s.Equal(1, childItem.Depth)
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_includes_new_children() {
	bc := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"new-child": {
				NewExports: map[string]provider.FieldChange{
					"newChildExport": {NewValue: stringNode("val")},
				},
			},
		},
	}

	items := BuildExportChangeHierarchy(bc, "my-instance")
	s.Len(items, 2) // root + new-child

	newChildItem := items[1].(*StageExportsInstanceItem)
	s.Equal("new-child", newChildItem.Name)
	s.Equal(1, newChildItem.NewCount)
}

func (s *StageExportsTestSuite) Test_BuildExportChangeHierarchy_sorts_children_alphabetically() {
	bc := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"z-child": {},
			"a-child": {},
			"m-child": {},
		},
	}

	items := BuildExportChangeHierarchy(bc, "root")
	s.Len(items, 4) // root + 3 children

	s.Equal("a-child", items[1].(*StageExportsInstanceItem).Name)
	s.Equal("m-child", items[2].(*StageExportsInstanceItem).Name)
	s.Equal("z-child", items[3].(*StageExportsInstanceItem).Name)
}

func (s *StageExportsTestSuite) Test_NewStageExportsModel_creates_model() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("value1")},
		},
	}
	model := NewStageExportsModel(bc, "my-instance", 120, 40, s.styles)
	s.NotNil(model)
}

func (s *StageExportsTestSuite) Test_NewStageExportsModel_stores_changes() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("value1")},
		},
	}
	model := NewStageExportsModel(bc, "my-instance", 120, 40, s.styles)
	s.Equal(bc, model.changes)
	s.Equal("my-instance", model.instanceName)
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_true_when_new_exports() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("val")},
		},
	}
	model := NewStageExportsModel(bc, "inst", 120, 40, s.styles)
	s.True(model.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_true_when_removed_exports() {
	bc := &changes.BlueprintChanges{
		RemovedExports: []string{"export1"},
	}
	model := NewStageExportsModel(bc, "inst", 120, 40, s.styles)
	s.True(model.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_false_when_no_changes() {
	bc := &changes.BlueprintChanges{}
	model := NewStageExportsModel(bc, "inst", 120, 40, s.styles)
	s.False(model.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_HasExportChanges_returns_false_for_resolve_on_deploy_only() {
	prevVal := stringNode("old")
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"computed": {PrevValue: prevVal, NewValue: nil},
		},
		ResolveOnDeploy: []string{"exports.computed"},
	}
	model := NewStageExportsModel(bc, "inst", 120, 40, s.styles)
	s.False(model.HasExportChanges())
}

func (s *StageExportsTestSuite) Test_RenderDetails_returns_no_instance_for_non_item() {
	renderer := &StageExportsDetailsRenderer{}
	result := renderer.RenderDetails(&mockItem{}, 80, s.styles)
	s.Contains(result, "No instance selected")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_instance_name() {
	renderer := &StageExportsDetailsRenderer{}
	item := &StageExportsInstanceItem{
		Name:    "my-instance",
		Path:    "",
		Changes: &changes.BlueprintChanges{},
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "my-instance")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_path_for_nested_children() {
	renderer := &StageExportsDetailsRenderer{}
	item := &StageExportsInstanceItem{
		Name:    "child-b",
		Path:    "child-a/child-b",
		Depth:   2,
		Changes: &changes.BlueprintChanges{},
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "child-a/child-b")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_no_export_changes_message() {
	renderer := &StageExportsDetailsRenderer{}
	item := &StageExportsInstanceItem{
		Name:    "root",
		Changes: &changes.BlueprintChanges{},
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "No export changes")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_unchanged_count() {
	renderer := &StageExportsDetailsRenderer{}
	item := &StageExportsInstanceItem{
		Name:           "root",
		UnchangedCount: 3,
		Changes:        &changes.BlueprintChanges{},
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "3 unchanged")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_new_exports_section() {
	renderer := &StageExportsDetailsRenderer{}
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"myExport": {NewValue: stringNode("hello")},
		},
	}
	item := &StageExportsInstanceItem{
		Name:     "root",
		NewCount: 1,
		Changes:  bc,
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "New Exports")
	s.Contains(result, "myExport")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_modified_exports_section() {
	renderer := &StageExportsDetailsRenderer{}
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"modifiedExport": {
				PrevValue: stringNode("old-value"),
				NewValue:  stringNode("new-value"),
			},
		},
	}
	item := &StageExportsInstanceItem{
		Name:          "root",
		ModifiedCount: 1,
		Changes:       bc,
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "Modified Exports")
	s.Contains(result, "modifiedExport")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_removed_exports_section() {
	renderer := &StageExportsDetailsRenderer{}
	bc := &changes.BlueprintChanges{
		RemovedExports: []string{"oldExport"},
	}
	item := &StageExportsInstanceItem{
		Name:         "root",
		RemovedCount: 1,
		Changes:      bc,
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "Removed Exports")
	s.Contains(result, "oldExport")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_value_for_new_export() {
	renderer := &StageExportsDetailsRenderer{}
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"apiEndpoint": {NewValue: stringNode("https://api.example.com")},
		},
	}
	item := &StageExportsInstanceItem{
		Name:     "root",
		NewCount: 1,
		Changes:  bc,
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "https://api.example.com")
}

func (s *StageExportsTestSuite) Test_RenderDetails_shows_known_on_deploy_for_computed_export() {
	renderer := &StageExportsDetailsRenderer{}
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"computedExport": {NewValue: nil},
		},
		ResolveOnDeploy: []string{"exports.computedExport"},
	}
	item := &StageExportsInstanceItem{
		Name:     "root",
		NewCount: 1,
		Changes:  bc,
	}
	result := renderer.RenderDetails(item, 80, s.styles)
	s.Contains(result, "known on deploy")
}

func (s *StageExportsTestSuite) Test_RenderFooter_returns_navigation_hints() {
	renderer := &StageExportsFooterRenderer{}
	result := renderer.RenderFooter(&splitpane.Model{}, s.styles)
	s.NotEmpty(result)
}

func (s *StageExportsTestSuite) Test_RenderFooter_implements_footer_renderer() {
	var _ splitpane.FooterRenderer = (*StageExportsFooterRenderer)(nil)
}
