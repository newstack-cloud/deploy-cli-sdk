package splitpane

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The status picker is how a status filter is discovered and applied without
// having to know the "status:" syntax or read it off a hint line. A cramped
// pane cannot show the available statuses inline, and that is exactly where
// filtering matters most, so they get a view of their own instead.
//
// It opens with tab while a filter is being typed, which matches the
// tab-completion gesture the key already suggests.

// IsPickingStatus reports whether the status picker is open.
func (m Model) IsPickingStatus() bool {
	return m.statusPicker
}

// Reports whether there is a vocabulary to pick from.
func (m Model) statusPickerAvailable() bool {
	return len(m.config.StatusFilterHints) > 0
}

func (m *Model) openStatusPicker() {
	if !m.statusPickerAvailable() {
		return
	}
	m.statusPicker = true
	m.statusPickerIndex = m.indexOfAppliedStatus()
	if m.initialized {
		m.updateViewports()
		m.leftPane.GotoTop()
	}
}

func (m *Model) closeStatusPicker() {
	m.statusPicker = false
	if m.initialized {
		m.updateViewports()
	}
}

// Starts the picker on the status already in the term, so
// reopening it shows where you are rather than jumping back to the top.
func (m Model) indexOfAppliedStatus() int {
	applied := parseFilterTerm(m.filterTerm).statuses
	if len(applied) == 0 {
		return 0
	}

	for i, status := range m.config.StatusFilterHints {
		if slices.Contains(applied, status) {
			return i
		}
	}

	return 0
}

// This drives the picker. Anything not understood closes it,
// so a stray key returns to typing rather than being swallowed.
func (m *Model) handleStatusPickerKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.statusPickerIndex > 0 {
			m.statusPickerIndex -= 1
			m.refreshStatusPicker()
		}
	case "down", "j":
		if m.statusPickerIndex < len(m.config.StatusFilterHints)-1 {
			m.statusPickerIndex += 1
			m.refreshStatusPicker()
		}
	case "enter":
		m.applyPickedStatus()
	case "esc", "tab":
		m.closeStatusPicker()
	case "ctrl+c":
		return func() tea.Msg { return QuitMsg{} }
	}
	return nil
}

func (m *Model) refreshStatusPicker() {
	if m.initialized {
		m.updateViewports()
	}
}

// Puts the chosen status into the term, replacing whatever
// status was there before and leaving any name the user had typed alone.
func (m *Model) applyPickedStatus() {
	if m.statusPickerIndex >= len(m.config.StatusFilterHints) {
		m.closeStatusPicker()
		return
	}

	parsed := parseFilterTerm(m.filterTerm)
	chosen := StatusFilterPrefix + m.config.StatusFilterHints[m.statusPickerIndex]

	term := chosen
	if parsed.text != "" {
		term = chosen + " " + parsed.text
	}

	m.statusPicker = false
	m.filterTerm = term
	m.afterFilterChange()
}

// Reports how many items each status would leave visible, so
// the picker shows where the problems are rather than just what can be typed.
func (m Model) statusPickerCounts() []int {
	counts := make([]int, len(m.config.StatusFilterHints))
	for i, status := range m.config.StatusFilterHints {
		probe := m
		probe.filterTerm = StatusFilterPrefix + status
		probe.statusPicker = false
		counts[i] = len(probe.visibleItems())
	}

	return counts
}

// Draws the picker in place of the item list.
func (m Model) renderStatusPicker(sb *strings.Builder) {
	styles := m.config.Styles

	sb.WriteString(styles.Category.Render("Filter by status"))
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", safePaneWidth(m.leftPane.Width-4))))
	sb.WriteString("\n")

	counts := m.statusPickerCounts()
	for i, status := range m.config.StatusFilterHints {
		line := fmt.Sprintf("  %s  (%d)", status, counts[i])
		if i == m.statusPickerIndex {
			sb.WriteString(styles.SelectedNavItem.Render(line))
		} else if counts[i] == 0 {
			// Nothing is in this state, so it is displayed as muted.
			sb.WriteString(styles.Muted.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render("  ↑/↓ choose  enter apply  esc back"))
	sb.WriteString("\n")
}

// Keeps a separator from being rendered with a negative width.
func safePaneWidth(width int) int {
	if width < 1 {
		return 1
	}
	return width
}
