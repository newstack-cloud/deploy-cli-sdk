package splitpane

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.MouseMsg:
		m.handleMouse(msg)

	case ItemsUpdatedMsg:
		// External trigger to refresh viewports
		m.updateViewports()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	// Account for borders and padding on both panes:
	// - Each pane has RoundedBorder (2 chars: left + right)
	// - Each pane has Padding(0, 1) (2 chars: left + right padding)
	// Total: 4 chars per pane = 8 chars for both panes
	borderAndPadding := 8

	// Account for vertical space:
	// - Top and bottom borders: 2 lines
	// - Footer: ~8 lines (varies, but reserve enough space)
	verticalOverhead := 10

	availableWidth := msg.Width - borderAndPadding
	leftWidth := int(float64(availableWidth) * m.config.LeftPaneRatio)
	rightWidth := availableWidth - leftWidth
	viewportHeight := msg.Height - verticalOverhead

	if !m.initialized {
		m.leftPane = viewport.New(leftWidth, viewportHeight)
		m.rightPane = viewport.New(rightWidth, viewportHeight)
		m.initialized = true
	} else {
		m.leftPane.Width = leftWidth
		m.leftPane.Height = viewportHeight
		m.rightPane.Width = rightWidth
		m.rightPane.Height = viewportHeight
	}

	m.updateViewports()
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		return func() tea.Msg { return QuitMsg{} }

	case "tab":
		m.toggleFocus()

	case "up", "k":
		return m.handleUp()

	case "down", "j":
		return m.handleDown()

	case "home":
		return m.handleHome()

	case "end":
		return m.handleEnd()

	case "pgup":
		m.handlePageUp()

	case "pgdown":
		m.handlePageDown()

	case "enter":
		return m.handleEnter()

	case "esc", "backspace":
		return m.handleBack()
	}
	return nil
}

func (m *Model) toggleFocus() {
	if m.focusedPane == LeftPane {
		m.focusedPane = RightPane
	} else {
		m.focusedPane = LeftPane
	}
}

func (m *Model) handleUp() tea.Cmd {
	if m.focusedPane == LeftPane {
		items := m.visibleItems()
		if m.selectedIndex > 0 {
			m.selectedIndex--
			// Update selectedID to match new index
			if m.selectedIndex < len(items) {
				m.selectedID = items[m.selectedIndex].GetID()
			}
			m.updateViewportsAndResetRight()
			return m.emitItemSelected()
		}
	} else {
		m.rightPane.ScrollUp(1)
	}
	return nil
}

func (m *Model) handleDown() tea.Cmd {
	if m.focusedPane == LeftPane {
		items := m.visibleItems()
		if m.selectedIndex < len(items)-1 {
			m.selectedIndex++
			// Update selectedID to match new index
			if m.selectedIndex < len(items) {
				m.selectedID = items[m.selectedIndex].GetID()
			}
			m.updateViewportsAndResetRight()
			return m.emitItemSelected()
		}
	} else {
		m.rightPane.ScrollDown(1)
	}
	return nil
}

func (m *Model) handleHome() tea.Cmd {
	if m.focusedPane == LeftPane {
		items := m.visibleItems()
		m.selectedIndex = 0
		// Update selectedID to match new index
		if len(items) > 0 {
			m.selectedID = items[0].GetID()
		}
		m.updateViewportsAndResetRight()
		return m.emitItemSelected()
	}
	m.rightPane.GotoTop()
	return nil
}

func (m *Model) handleEnd() tea.Cmd {
	if m.focusedPane == LeftPane {
		items := m.visibleItems()
		if len(items) > 0 {
			m.selectedIndex = len(items) - 1
			// Update selectedID to match new index
			m.selectedID = items[m.selectedIndex].GetID()
		}
		m.updateViewportsAndResetRight()
		return m.emitItemSelected()
	}
	m.rightPane.GotoBottom()
	return nil
}

func (m *Model) handlePageUp() {
	if m.focusedPane == RightPane {
		m.rightPane.HalfPageUp()
	}
}

func (m *Model) handlePageDown() {
	if m.focusedPane == RightPane {
		m.rightPane.HalfPageDown()
	}
}

func (m *Model) handleEnter() tea.Cmd {
	if m.focusedPane != LeftPane {
		return nil
	}

	item := m.SelectedItem()
	if item == nil {
		return nil
	}

	// Calculate effective depth (item depth + navigation stack depth)
	effectiveDepth := item.GetDepth() + len(m.navigationStack)

	// Expand/collapse if expandable and under max depth
	if item.IsExpandable() && effectiveDepth < m.config.MaxExpandDepth {
		id := item.GetID()
		m.expandedItems[id] = !m.expandedItems[id]
		m.updateViewportsAndResetRight()
		return func() tea.Msg {
			return ItemExpandedMsg{Item: item, Expanded: m.expandedItems[id]}
		}
	}

	// Drill down if at max depth and can drill
	if item.CanDrillDown() {
		m.navigationStack = append(m.navigationStack, NavigationFrame{
			ParentID:   item.GetID(),
			ParentName: item.GetName(),
			SelectedID: m.selectedID, // Save current selection for back navigation
			Items:      m.items,
		})
		m.items = item.GetChildren()
		m.selectedIndex = 0
		// Set selectedID to first child if any
		if len(m.items) > 0 {
			m.selectedID = m.items[0].GetID()
		} else {
			m.selectedID = ""
		}
		m.expandedItems = make(map[string]bool)
		m.updateViewportsAndResetRight()
		return func() tea.Msg { return DrillDownMsg{Item: item} }
	}

	return nil
}

func (m *Model) handleBack() tea.Cmd {
	if len(m.navigationStack) > 0 {
		// Pop from stack
		frame := m.navigationStack[len(m.navigationStack)-1]
		m.navigationStack = m.navigationStack[:len(m.navigationStack)-1]
		m.items = frame.Items

		// Reset expansion state - resolveSelectedIndex will re-expand
		// parents as needed to make the selected item visible
		m.expandedItems = make(map[string]bool)

		// Restore selection from when we drilled down
		if frame.SelectedID != "" {
			m.selectedID = frame.SelectedID
			m.resolveSelectedIndex()
		} else {
			m.selectedIndex = 0
			if len(m.items) > 0 {
				m.selectedID = m.items[0].GetID()
			}
		}
		m.updateViewportsAndResetRight()
		return nil
	}
	// At root - emit BackMsg for parent to handle
	return func() tea.Msg { return BackMsg{} }
}

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if m.focusedPane == LeftPane {
		m.leftPane, _ = m.leftPane.Update(msg)
	} else {
		m.rightPane, _ = m.rightPane.Update(msg)
	}
}

func (m *Model) emitItemSelected() tea.Cmd {
	item := m.SelectedItem()
	if item == nil {
		return nil
	}
	return func() tea.Msg { return ItemSelectedMsg{Item: item} }
}

func (m *Model) updateViewports() {
	m.leftPane.SetContent(m.renderLeftPane())
	m.rightPane.SetContent(m.renderRightPane())
}

func (m *Model) updateViewportsAndResetRight() {
	m.leftPane.SetContent(m.renderLeftPane())
	m.rightPane.SetContent(m.renderRightPane())
	m.rightPane.GotoTop()
	m.scrollLeftPaneToSelection()
}
