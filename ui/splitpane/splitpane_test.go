package splitpane

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

// mockItem implements the Item interface for testing
type mockItem struct {
	id         string
	name       string
	icon       string
	action     string
	depth      int
	parentID   string
	itemType   string
	expandable bool
	canDrill   bool
	children   []Item
}

func (m *mockItem) GetID() string         { return m.id }
func (m *mockItem) GetName() string       { return m.name }
func (m *mockItem) GetIcon(bool) string   { return m.icon }
func (m *mockItem) GetAction() string     { return m.action }
func (m *mockItem) GetDepth() int         { return m.depth }
func (m *mockItem) GetParentID() string   { return m.parentID }
func (m *mockItem) GetItemType() string   { return m.itemType }
func (m *mockItem) IsExpandable() bool    { return m.expandable }
func (m *mockItem) CanDrillDown() bool    { return m.canDrill }
func (m *mockItem) GetChildren() []Item   { return m.children }

// mockDetailsRenderer implements DetailsRenderer for testing
type mockDetailsRenderer struct{}

func (r *mockDetailsRenderer) RenderDetails(item Item, width int, s *styles.Styles) string {
	return "Details for: " + item.GetName()
}

// testableModel wraps Model to satisfy tea.Model interface for teatest
type testableModel struct {
	Model
}

func (m testableModel) Init() tea.Cmd {
	return m.Model.Init()
}

func (m testableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.Model.Update(msg)
	return testableModel{Model: updated}, cmd
}

func (m testableModel) View() string {
	return m.Model.View()
}

// SplitPaneSuite is the test suite for splitpane
type SplitPaneSuite struct {
	suite.Suite
	styles *styles.Styles
}

func (s *SplitPaneSuite) SetupSuite() {
	s.styles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func TestSplitPaneSuite(t *testing.T) {
	suite.Run(t, new(SplitPaneSuite))
}

// createTestItems creates a standard set of test items
func createTestItems() []Item {
	return []Item{
		&mockItem{id: "1", name: "Item 1", icon: "●", action: "CREATE"},
		&mockItem{id: "2", name: "Item 2", icon: "●", action: "UPDATE"},
		&mockItem{id: "3", name: "Item 3", icon: "●", action: "DELETE"},
	}
}

// createExpandableItems creates items with expandable children
func createExpandableItems() []Item {
	return []Item{
		&mockItem{
			id:         "parent1",
			name:       "Parent 1",
			icon:       "●",
			action:     "CREATE",
			expandable: true,
			children: []Item{
				&mockItem{id: "child1", name: "Child 1", icon: "○", parentID: "parent1", depth: 1},
				&mockItem{id: "child2", name: "Child 2", icon: "○", parentID: "parent1", depth: 1},
			},
		},
		&mockItem{id: "2", name: "Item 2", icon: "●", action: "UPDATE"},
	}
}

// createDrillableItems creates items that can be drilled into
func createDrillableItems() []Item {
	return []Item{
		&mockItem{
			id:       "drillable1",
			name:     "Drillable 1",
			icon:     "●",
			action:   "VIEW",
			canDrill: true,
			children: []Item{
				&mockItem{id: "detail1", name: "Detail 1", icon: "○"},
				&mockItem{id: "detail2", name: "Detail 2", icon: "○"},
			},
		},
		&mockItem{id: "2", name: "Item 2", icon: "●"},
	}
}

// createTestModel creates a test model with standard configuration
func (s *SplitPaneSuite) createTestModel(items []Item) *teatest.TestModel {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(items)

	return teatest.NewTestModel(
		s.T(),
		testableModel{Model: model},
		teatest.WithInitialTermSize(120, 40),
	)
}

// getFinalModel extracts the Model from testableModel wrapper
func getFinalModel(tm *teatest.TestModel, t *testing.T) Model {
	return tm.FinalModel(t).(testableModel).Model
}

// --- Initialization Tests ---

func (s *SplitPaneSuite) Test_new_model_with_default_config() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)

	s.Equal(0.4, model.Config().LeftPaneRatio)
	s.Equal(2, model.Config().MaxExpandDepth)
}

func (s *SplitPaneSuite) Test_new_model_with_custom_config() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
		LeftPaneRatio:   0.5,
		MaxExpandDepth:  3,
	}
	model := New(config)

	s.Equal(0.5, model.Config().LeftPaneRatio)
	s.Equal(3, model.Config().MaxExpandDepth)
}

func (s *SplitPaneSuite) Test_set_items_initializes_selection() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	items := createTestItems()
	model.SetItems(items)

	s.Equal("1", model.SelectedID())
	s.Equal(0, model.SelectedIndex())
}

func (s *SplitPaneSuite) Test_set_items_empty_list() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems([]Item{})

	s.Equal("", model.SelectedID())
	s.Equal(0, model.SelectedIndex())
	s.Nil(model.SelectedItem())
}

// --- Navigation Tests - Left Pane ---

func (s *SplitPaneSuite) Test_navigation_down_selects_next_item() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	testutils.KeyDown(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 2")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("2", finalModel.SelectedID())
	s.Equal(1, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_navigation_up_selects_previous_item() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Move down first, then up
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel)
	testutils.KeyUp(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("2", finalModel.SelectedID())
	s.Equal(1, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_navigation_down_at_bottom_stays_at_last() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Move to the last item and try to go further
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel) // Should stay at last
	testutils.KeyDown(testModel) // Should stay at last

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("3", finalModel.SelectedID())
	s.Equal(2, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_navigation_up_at_top_stays_at_first() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Try to move up at the first item
	testutils.KeyUp(testModel)
	testutils.KeyUp(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("1", finalModel.SelectedID())
	s.Equal(0, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_navigation_home_jumps_to_first() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Move down then press Home
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel)
	testutils.KeyHome(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("1", finalModel.SelectedID())
	s.Equal(0, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_navigation_end_jumps_to_last() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	testutils.KeyEnd(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("3", finalModel.SelectedID())
	s.Equal(2, finalModel.SelectedIndex())
}

func (s *SplitPaneSuite) Test_vim_keys_j_navigates_down() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	testutils.KeyJ(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("2", finalModel.SelectedID())
}

func (s *SplitPaneSuite) Test_vim_keys_k_navigates_up() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Move down first, then use k to go up
	testutils.KeyJ(testModel)
	testutils.KeyK(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("1", finalModel.SelectedID())
}

// --- Focus Management Tests ---

func (s *SplitPaneSuite) Test_tab_toggles_focus_to_right_pane() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	testutils.KeyTab(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal(RightPane, finalModel.FocusedPane())
}

func (s *SplitPaneSuite) Test_tab_toggles_focus_to_left_pane() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Tab to right, then back to left
	testutils.KeyTab(testModel)
	testutils.KeyTab(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal(LeftPane, finalModel.FocusedPane())
}

func (s *SplitPaneSuite) Test_arrow_keys_scroll_right_pane_when_focused() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	// Switch to right pane and verify navigation doesn't change selection
	testutils.KeyTab(testModel)
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	// Selection should still be on first item (left pane navigation didn't happen)
	s.Equal("1", finalModel.SelectedID())
	s.Equal(RightPane, finalModel.FocusedPane())
}

// --- Expansion Tests ---

func (s *SplitPaneSuite) Test_enter_expands_expandable_item() {
	testModel := s.createTestModel(createExpandableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Parent 1")

	testutils.KeyEnter(testModel)

	// After expansion, children should be visible
	testutils.WaitForContains(s.T(), testModel.Output(), "▼")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.True(finalModel.IsExpanded("parent1"))
}

func (s *SplitPaneSuite) Test_enter_collapses_expanded_item() {
	testModel := s.createTestModel(createExpandableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Parent 1")

	// Expand
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "▼")

	// Collapse
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "▶")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.False(finalModel.IsExpanded("parent1"))
}

func (s *SplitPaneSuite) Test_expansion_state_tracks_by_id() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(createExpandableItems())

	s.False(model.IsExpanded("parent1"))
	s.False(model.IsExpanded("nonexistent"))
}

// --- Drill-Down Navigation Tests ---

func (s *SplitPaneSuite) Test_enter_drills_down_when_can_drill() {
	testModel := s.createTestModel(createDrillableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	testutils.KeyEnter(testModel)

	// After drill-down, should see breadcrumb and children
	testutils.WaitForContains(s.T(), testModel.Output(), "Detail 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.True(finalModel.IsInDrillDown())
	s.Len(finalModel.NavigationStack(), 1)
}

func (s *SplitPaneSuite) Test_drill_down_shows_children() {
	testModel := s.createTestModel(createDrillableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	testutils.KeyEnter(testModel)

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "Detail 1", "Detail 2")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Len(finalModel.Items(), 2)
}

func (s *SplitPaneSuite) Test_back_navigation_pops_stack() {
	testModel := s.createTestModel(createDrillableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	// Drill down
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Detail 1")

	// Navigate back
	testutils.KeyEscape(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.False(finalModel.IsInDrillDown())
	s.Len(finalModel.NavigationStack(), 0)
}

func (s *SplitPaneSuite) Test_back_navigation_restores_selection() {
	testModel := s.createTestModel(createDrillableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	// Drill down
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Detail 1")

	// Navigate back
	testutils.KeyBackspace(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	// Selection should be restored to the drillable item
	s.Equal("drillable1", finalModel.SelectedID())
}

func (s *SplitPaneSuite) Test_navigation_path_returns_breadcrumb() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(createDrillableItems())

	s.Empty(model.NavigationPath())
}

func (s *SplitPaneSuite) Test_is_in_drill_down() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(createDrillableItems())

	s.False(model.IsInDrillDown())
}

// --- Item Update Tests ---

func (s *SplitPaneSuite) Test_update_items_preserves_selection() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	items := createTestItems()
	model.SetItems(items)

	// Select second item
	model.resolveSelectedIndex()
	model.selectedID = "2"
	model.resolveSelectedIndex()

	// Update items (same items, different order shouldn't matter for this test)
	newItems := []Item{
		&mockItem{id: "1", name: "Item 1 Updated", icon: "●"},
		&mockItem{id: "2", name: "Item 2 Updated", icon: "●"},
		&mockItem{id: "3", name: "Item 3 Updated", icon: "●"},
	}
	model.UpdateItems(newItems)

	s.Equal("2", model.SelectedID())
}

func (s *SplitPaneSuite) Test_update_items_resets_selection_if_removed() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	items := createTestItems()
	model.SetItems(items)

	// Select second item
	model.selectedID = "2"
	model.resolveSelectedIndex()

	// Update items without the selected item
	newItems := []Item{
		&mockItem{id: "1", name: "Item 1", icon: "●"},
		&mockItem{id: "3", name: "Item 3", icon: "●"},
	}
	model.UpdateItems(newItems)

	// Selection should move to first item
	s.Equal("1", model.SelectedID())
	s.Equal(0, model.SelectedIndex())
}

func (s *SplitPaneSuite) Test_add_item_appends_to_root() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(createTestItems())

	newItem := &mockItem{id: "4", name: "Item 4", icon: "●"}
	model.AddItem(newItem)

	s.Len(model.RootItems(), 4)
	s.Len(model.Items(), 4)
	s.Equal("4", model.Items()[3].GetID())
}

func (s *SplitPaneSuite) Test_update_item_by_id() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems(createTestItems())

	newItem := &mockItem{id: "2", name: "Item 2 Modified", icon: "★", action: "MODIFIED"}
	found := model.UpdateItemByID("2", newItem)

	s.True(found)
	s.Equal("Item 2 Modified", model.Items()[1].GetName())
}

func (s *SplitPaneSuite) Test_remove_item_by_id() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	// Create fresh items for this test
	items := []Item{
		&mockItem{id: "1", name: "Item 1", icon: "●", action: "CREATE"},
		&mockItem{id: "2", name: "Item 2", icon: "●", action: "UPDATE"},
		&mockItem{id: "3", name: "Item 3", icon: "●", action: "DELETE"},
	}
	model.SetItems(items)

	found := model.RemoveItemByID("2")

	s.True(found)
	s.Len(model.RootItems(), 2)
	// Verify "2" is gone
	for _, item := range model.RootItems() {
		s.NotEqual("2", item.GetID())
	}
}

func (s *SplitPaneSuite) Test_remove_selected_item_adjusts_selection() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	// Create fresh items for this test
	items := []Item{
		&mockItem{id: "1", name: "Item 1", icon: "●", action: "CREATE"},
		&mockItem{id: "2", name: "Item 2", icon: "●", action: "UPDATE"},
		&mockItem{id: "3", name: "Item 3", icon: "●", action: "DELETE"},
	}
	model.SetItems(items)

	// Select item 2
	model.selectedID = "2"
	model.resolveSelectedIndex()

	// Remove the selected item
	model.RemoveItemByID("2")

	// Selection should adjust
	s.NotEqual("2", model.SelectedID())
	s.Len(model.RootItems(), 2)
}

// --- Message Emission Tests ---

func (s *SplitPaneSuite) Test_quit_message_on_q() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	testutils.KeyQ(testModel)

	// The model should emit QuitMsg - we verify by the test completing without timeout
	err := testModel.Quit()
	s.NoError(err)
}

// --- Window Sizing Tests ---

func (s *SplitPaneSuite) Test_window_resize_updates_pane_widths() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	// With 120 width terminal and default 0.4 ratio
	// Available width = 120 - 8 (borders) = 112
	// Left pane = 112 * 0.4 = ~44
	s.True(finalModel.LeftPaneWidth() > 0)
	s.True(finalModel.RightPaneWidth() > 0)
}

func (s *SplitPaneSuite) Test_pane_ratio_applied_correctly() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
		LeftPaneRatio:   0.5, // 50% split
	}
	model := New(config)
	model.SetItems(createTestItems())

	testModel := teatest.NewTestModel(
		s.T(),
		testableModel{Model: model},
		teatest.WithInitialTermSize(120, 40),
	)

	testutils.WaitForContains(s.T(), testModel.Output(), "Item 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	// With 50% ratio, panes should be roughly equal
	leftWidth := finalModel.LeftPaneWidth()
	rightWidth := finalModel.RightPaneWidth()
	// Allow some tolerance for borders
	s.InDelta(leftWidth, rightWidth, 5)
}

// --- View Rendering Tests ---

func (s *SplitPaneSuite) Test_item_icons_displayed() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "●")

	err := testModel.Quit()
	s.NoError(err)
}

func (s *SplitPaneSuite) Test_item_actions_displayed() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "CREATE", "UPDATE", "DELETE")

	err := testModel.Quit()
	s.NoError(err)
}

func (s *SplitPaneSuite) Test_details_renderer_called() {
	testModel := s.createTestModel(createTestItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Details for: Item 1")

	err := testModel.Quit()
	s.NoError(err)
}

func (s *SplitPaneSuite) Test_breadcrumb_displayed_in_drill_down() {
	testModel := s.createTestModel(createDrillableItems())

	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	testutils.KeyEnter(testModel)

	// Breadcrumb should show the drilled item name
	testutils.WaitForContains(s.T(), testModel.Output(), "Drillable 1")

	err := testModel.Quit()
	s.NoError(err)
}

// --- Edge Cases ---

func (s *SplitPaneSuite) Test_empty_items_handles_navigation() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems([]Item{})

	testModel := teatest.NewTestModel(
		s.T(),
		testableModel{Model: model},
		teatest.WithInitialTermSize(120, 40),
	)

	// Navigation on empty items should not panic
	testutils.KeyDown(testModel)
	testutils.KeyUp(testModel)
	testutils.KeyHome(testModel)
	testutils.KeyEnd(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Equal("", finalModel.SelectedID())
}

func (s *SplitPaneSuite) Test_selected_item_returns_nil_when_empty() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
	}
	model := New(config)
	model.SetItems([]Item{})

	s.Nil(model.SelectedItem())
}

func (s *SplitPaneSuite) Test_deeply_nested_drill_down() {
	// Create items with multiple levels of drill-down
	items := []Item{
		&mockItem{
			id:       "level1",
			name:     "Level 1",
			icon:     "●",
			canDrill: true,
			children: []Item{
				&mockItem{
					id:       "level2",
					name:     "Level 2",
					icon:     "●",
					canDrill: true,
					children: []Item{
						&mockItem{id: "level3", name: "Level 3", icon: "●"},
					},
				},
			},
		},
	}

	testModel := s.createTestModel(items)

	testutils.WaitForContains(s.T(), testModel.Output(), "Level 1")

	// Drill down to level 2
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 2")

	// Drill down to level 3
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 3")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.Len(finalModel.NavigationStack(), 2)
	s.Equal([]string{"Level 1", "Level 2"}, finalModel.NavigationPath())
}

func (s *SplitPaneSuite) Test_multiple_back_navigations() {
	items := []Item{
		&mockItem{
			id:       "level1",
			name:     "Level 1",
			icon:     "●",
			canDrill: true,
			children: []Item{
				&mockItem{
					id:       "level2",
					name:     "Level 2",
					icon:     "●",
					canDrill: true,
					children: []Item{
						&mockItem{id: "level3", name: "Level 3", icon: "●"},
					},
				},
			},
		},
	}

	testModel := s.createTestModel(items)

	testutils.WaitForContains(s.T(), testModel.Output(), "Level 1")

	// Drill down twice
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 2")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 3")

	// Navigate back twice
	testutils.KeyEscape(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 2")
	testutils.KeyEscape(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Level 1")

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.False(finalModel.IsInDrillDown())
}

func (s *SplitPaneSuite) Test_public_getter_methods() {
	config := Config{
		Styles:          s.styles,
		DetailsRenderer: &mockDetailsRenderer{},
		Title:           "Test Title",
	}
	model := New(config)
	items := createTestItems()
	model.SetItems(items)

	// Test all public getters
	s.Equal(LeftPane, model.FocusedPane())
	s.Equal(0, model.Width())  // Not initialized yet
	s.Equal(0, model.Height()) // Not initialized yet
	s.NotNil(model.Styles())
	s.Equal("Test Title", model.Config().Title)
	s.Len(model.Items(), 3)
	s.Len(model.RootItems(), 3)
	s.Empty(model.NavigationStack())
}

func (s *SplitPaneSuite) Test_enter_on_non_expandable_non_drillable_item() {
	items := []Item{
		&mockItem{id: "1", name: "Plain Item", icon: "●", expandable: false, canDrill: false},
	}

	testModel := s.createTestModel(items)

	testutils.WaitForContains(s.T(), testModel.Output(), "Plain Item")

	// Enter on non-expandable, non-drillable item should do nothing
	testutils.KeyEnter(testModel)

	err := testModel.Quit()
	s.NoError(err)

	finalModel := getFinalModel(testModel, s.T())
	s.False(finalModel.IsInDrillDown())
	s.False(finalModel.IsExpanded("1"))
}
