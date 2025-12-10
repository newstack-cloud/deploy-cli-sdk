package splitpane

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// PaneType indicates which pane is focused
type PaneType int

const (
	LeftPane PaneType = iota
	RightPane
)

// Model is a Bubble Tea model for a two-pane split view.
// Left pane shows a navigable list; right pane shows details.
type Model struct {
	// Layout
	leftPane      viewport.Model
	rightPane     viewport.Model
	focusedPane   PaneType
	width, height int
	initialized   bool

	// Navigation state
	rootItems       []Item            // Root-level items (for drill-down refresh)
	items           []Item            // Current level's items (displayed)
	selectedID      string            // ID of the selected item (stable across updates)
	selectedIndex   int               // Cached index (computed from selectedID)
	expandedItems   map[string]bool   // Tracks expanded items by ID
	navigationStack []NavigationFrame // Drill-down history

	// Configuration
	config Config
}

// New creates a new split-pane model with the given configuration.
func New(config Config) Model {
	if config.LeftPaneRatio == 0 {
		config.LeftPaneRatio = 0.4
	}
	if config.MaxExpandDepth == 0 {
		config.MaxExpandDepth = 2
	}

	return Model{
		config:        config,
		expandedItems: make(map[string]bool),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil
}

// SetItems sets the items to display, resetting all state.
// Use UpdateItems() instead if you want to preserve selection and expansion state.
func (m *Model) SetItems(items []Item) {
	m.rootItems = items
	m.items = items
	m.expandedItems = make(map[string]bool)
	m.navigationStack = nil

	// Set initial selection
	if len(items) > 0 {
		m.selectedID = items[0].GetID()
		m.selectedIndex = 0
	} else {
		m.selectedID = ""
		m.selectedIndex = 0
	}

	if m.initialized {
		m.updateViewports()
	}
}

// SelectedItem returns the currently selected item, or nil if none.
// Uses ID-based lookup to ensure the correct item is returned even after updates.
func (m Model) SelectedItem() Item {
	if m.selectedID == "" {
		return nil
	}
	items := m.visibleItems()
	for _, item := range items {
		if item.GetID() == m.selectedID {
			return item
		}
	}
	// Fallback to index if ID not found (shouldn't happen in normal use)
	if m.selectedIndex >= 0 && m.selectedIndex < len(items) {
		return items[m.selectedIndex]
	}
	return nil
}

// NavigationPath returns the breadcrumb path as a slice of names.
func (m Model) NavigationPath() []string {
	path := make([]string, len(m.navigationStack))
	for i, frame := range m.navigationStack {
		path[i] = frame.ParentName
	}
	return path
}

// IsInDrillDown returns true if currently viewing a drilled-down item.
func (m Model) IsInDrillDown() bool {
	return len(m.navigationStack) > 0
}

// FocusedPane returns which pane currently has focus.
func (m Model) FocusedPane() PaneType {
	return m.focusedPane
}

// Width returns the total width.
func (m Model) Width() int {
	return m.width
}

// Height returns the total height.
func (m Model) Height() int {
	return m.height
}

// LeftPaneWidth returns the left pane width.
func (m Model) LeftPaneWidth() int {
	return m.leftPane.Width
}

// RightPaneWidth returns the right pane width.
func (m Model) RightPaneWidth() int {
	return m.rightPane.Width
}

// Styles returns the configured styles.
func (m Model) Styles() *styles.Styles {
	return m.config.Styles
}

// Config returns the configuration.
func (m Model) Config() Config {
	return m.config
}

// Items returns the current items at the current navigation level.
func (m Model) Items() []Item {
	return m.items
}

// NavigationStack returns the navigation stack for drill-down.
func (m Model) NavigationStack() []NavigationFrame {
	return m.navigationStack
}

// IsExpanded returns true if the item with the given ID is expanded.
func (m Model) IsExpanded(id string) bool {
	return m.expandedItems[id]
}

// SelectedIndex returns the currently selected index.
func (m Model) SelectedIndex() int {
	return m.selectedIndex
}

// visibleItems returns the items that are currently visible (respecting expansion state).
func (m Model) visibleItems() []Item {
	if m.config.SectionGrouper != nil {
		sections := m.config.SectionGrouper.GroupItems(m.items, m.IsExpanded)
		var items []Item
		for _, section := range sections {
			items = append(items, section.Items...)
		}
		return items
	}
	return m.items
}

// SelectedID returns the ID of the currently selected item.
func (m Model) SelectedID() string {
	return m.selectedID
}

// RootItems returns the root-level items.
func (m Model) RootItems() []Item {
	return m.rootItems
}

// resolveSelectedIndex updates selectedIndex from selectedID.
// This should be called after items are updated to keep the index cache in sync.
func (m *Model) resolveSelectedIndex() {
	items := m.visibleItems()
	for i, item := range items {
		if item.GetID() == m.selectedID {
			m.selectedIndex = i
			return
		}
	}
	// Selected item no longer exists - select first item
	if len(items) > 0 {
		m.selectedID = items[0].GetID()
		m.selectedIndex = 0
	} else {
		m.selectedID = ""
		m.selectedIndex = 0
	}
}

// UpdateItems replaces items while preserving user state where possible.
// - Preserves selection if the selected item still exists (by ID)
// - Preserves expansion state for items that still exist
// - Properly handles drill-down mode by refreshing children from updated parent
func (m *Model) UpdateItems(items []Item) {
	// Always update root items
	m.rootItems = items

	if len(m.navigationStack) == 0 {
		// At root level - update displayed items directly
		m.updateItemsAtCurrentLevel(items)
	} else {
		// In drill-down - validate stack and refresh from updated root items
		m.validateNavigationStack()
	}

	if m.initialized {
		m.updateViewports()
	}
}

// updateItemsAtCurrentLevel updates items while preserving selection and expansion.
func (m *Model) updateItemsAtCurrentLevel(items []Item) {
	// Build lookup of new items by ID
	newItemsByID := make(map[string]int)
	for i, item := range items {
		newItemsByID[item.GetID()] = i
	}

	// Preserve expansion state for items that still exist
	newExpanded := make(map[string]bool)
	for id, expanded := range m.expandedItems {
		if _, exists := newItemsByID[id]; exists {
			newExpanded[id] = expanded
		}
	}

	// Update items and expansion state
	m.items = items
	m.expandedItems = newExpanded

	// Resolve selection - this will keep current selection if item still exists
	m.resolveSelectedIndex()
}

// refreshDrillDownItems updates drill-down view from current root items.
func (m *Model) refreshDrillDownItems() {
	if len(m.navigationStack) == 0 {
		return
	}

	// Find the item we drilled into by ID
	currentFrame := m.navigationStack[len(m.navigationStack)-1]
	parentItem := m.findItemByID(m.rootItems, currentFrame.ParentID)

	if parentItem == nil {
		// Parent item was removed - this shouldn't happen after validateNavigationStack
		// but handle it gracefully by popping back
		m.popNavigationStack()
		return
	}

	// Get fresh children from the parent
	m.items = parentItem.GetChildren()
	m.resolveSelectedIndex()
}

// validateNavigationStack ensures all frames still reference valid items.
// If any parent in the stack no longer exists, the stack is truncated.
func (m *Model) validateNavigationStack() {
	validFrames := []NavigationFrame{}
	currentItems := m.rootItems

	for _, frame := range m.navigationStack {
		parent := m.findItemInSlice(currentItems, frame.ParentID)
		if parent == nil {
			// Parent no longer exists - stop here
			break
		}
		validFrames = append(validFrames, NavigationFrame{
			ParentID:   frame.ParentID,
			ParentName: parent.GetName(), // Update name in case it changed
			SelectedID: frame.SelectedID,
			Items:      currentItems, // Update items snapshot
		})
		currentItems = parent.GetChildren()
	}

	m.navigationStack = validFrames
	if len(m.navigationStack) == 0 {
		// Back at root
		m.updateItemsAtCurrentLevel(m.rootItems)
	} else {
		// Still in drill-down, refresh children
		m.refreshDrillDownItems()
	}
}

// popNavigationStack pops one level from the navigation stack.
func (m *Model) popNavigationStack() {
	if len(m.navigationStack) == 0 {
		return
	}
	frame := m.navigationStack[len(m.navigationStack)-1]
	m.navigationStack = m.navigationStack[:len(m.navigationStack)-1]

	if len(m.navigationStack) == 0 {
		// Back at root
		m.items = m.rootItems
	} else {
		// Restore items from parent frame
		m.items = frame.Items
	}

	// Restore selection if possible
	if frame.SelectedID != "" {
		m.selectedID = frame.SelectedID
	}
	m.resolveSelectedIndex()
	m.expandedItems = make(map[string]bool)
}

// findItemByID searches recursively for an item by ID.
func (m *Model) findItemByID(items []Item, id string) Item {
	for _, item := range items {
		if item.GetID() == id {
			return item
		}
		// Search in children
		if children := item.GetChildren(); len(children) > 0 {
			if found := m.findItemByID(children, id); found != nil {
				return found
			}
		}
	}
	return nil
}

// findItemInSlice searches for an item by ID in a flat slice (non-recursive).
func (m *Model) findItemInSlice(items []Item, id string) Item {
	for _, item := range items {
		if item.GetID() == id {
			return item
		}
	}
	return nil
}

// AddItem adds an item to the end of the root items list.
// If currently in drill-down, the item is added to root but the view refreshes appropriately.
func (m *Model) AddItem(item Item) {
	m.rootItems = append(m.rootItems, item)

	if len(m.navigationStack) == 0 {
		// At root - add to displayed items too
		m.items = append(m.items, item)
	}
	// In drill-down, the item is in root but won't appear until navigating back

	if m.initialized {
		m.updateViewports()
	}
}

// UpdateItemByID replaces an item by ID in both root and current items.
// Returns true if the item was found and updated.
func (m *Model) UpdateItemByID(id string, newItem Item) bool {
	found := false

	// Update in root items
	for i, item := range m.rootItems {
		if item.GetID() == id {
			m.rootItems[i] = newItem
			found = true
			break
		}
	}

	// Update in current items if at root level
	if len(m.navigationStack) == 0 {
		for i, item := range m.items {
			if item.GetID() == id {
				m.items[i] = newItem
				break
			}
		}
	} else {
		// In drill-down - refresh to pick up any changes in parent's children
		m.refreshDrillDownItems()
	}

	if found && m.initialized {
		m.updateViewports()
	}

	return found
}

// RemoveItemByID removes an item by ID from root items.
// Returns true if the item was found and removed.
func (m *Model) RemoveItemByID(id string) bool {
	found := false

	// Remove from root items
	for i, item := range m.rootItems {
		if item.GetID() == id {
			m.rootItems = append(m.rootItems[:i], m.rootItems[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return false
	}

	// Remove expansion state
	delete(m.expandedItems, id)

	if len(m.navigationStack) == 0 {
		// At root - remove from displayed items too
		for i, item := range m.items {
			if item.GetID() == id {
				m.items = append(m.items[:i], m.items[i+1:]...)
				break
			}
		}
		// Adjust selection if needed
		m.resolveSelectedIndex()
	} else {
		// In drill-down - validate the navigation stack in case we removed a parent
		m.validateNavigationStack()
	}

	if m.initialized {
		m.updateViewports()
	}

	return true
}
