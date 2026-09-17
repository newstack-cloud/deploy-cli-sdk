package splitpane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// The minimum blank space kept between an item's name and its
// right-aligned action badge.
const actionGap = 2

// View implements tea.Model
func (m Model) View() string {
	if !m.initialized {
		return ""
	}

	// Two-column layout
	leftContent := m.leftPane.View()
	rightContent := m.rightPane.View()

	// Border styles - highlight focused pane
	leftBorder := m.config.Styles.Border(m.focusedPane == LeftPane).Padding(0, 1)
	rightBorder := m.config.Styles.Border(m.focusedPane == RightPane).Padding(0, 1)

	// Join the two panes
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftBorder.Render(leftContent),
		rightBorder.Render(rightContent),
	)

	// Footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, content, footer)
}

// renderLeftPane renders the navigation list in the left pane.
func (m Model) renderLeftPane() string {
	sb := strings.Builder{}

	if m.config.HeaderRenderer != nil {
		sb.WriteString(m.config.HeaderRenderer.RenderHeader(&m, m.config.Styles))
	} else {
		sb.WriteString(m.renderDefaultHeader())
	}

	m.renderFilterLine(&sb)

	// The picker stands in for the item list, which is what it is narrowing,
	// and carries its own keys, so nothing below it applies while it is open.
	if m.statusPicker {
		m.renderStatusPicker(&sb)
		return sb.String()
	}

	if m.config.SectionGrouper != nil {
		m.renderGroupedItems(&sb)
	} else {
		m.renderFlatItems(&sb)
	}

	return sb.String()
}

// Shows the active search term, how much it is hiding, and
// while it is being typed, what can be typed into it.
func (m Model) renderFilterLine(sb *strings.Builder) {
	if !m.filterInput && !m.HasFilter() {
		return
	}

	styles := m.config.Styles

	sb.WriteString(styles.Category.Render("Filter: "))
	if m.filterTerm == "" {
		sb.WriteString(styles.Muted.Render(m.emptyFilterPrompt()))
	} else {
		sb.WriteString(styles.Selected.Render(m.filterTerm))
	}
	if m.filterInput {
		sb.WriteString(styles.Selected.Render("▌"))
	}
	if m.HasFilter() {
		sb.WriteString(styles.Muted.Render(fmt.Sprintf(
			"  (%d of %d)", len(m.visibleItems()), m.totalItemCount(),
		)))
	}
	sb.WriteString("\n")

	// The picker lists the same statuses below, with counts, so the inline
	// vocabulary and the keys it describes would both be wrong while it is open.
	if m.statusPicker {
		sb.WriteString("\n")
		return
	}

	// The status vocabulary is only worth screen space while the term is being
	// typed, which is also the only moment it can be acted on.
	if m.filterInput {
		if hints := m.statusHintLine(); hints != "" {
			sb.WriteString(styles.Muted.Render(hints))
			sb.WriteString("\n")
		}
		if m.statusPickerAvailable() {
			sb.WriteString(styles.Muted.Render("tab to pick a status"))
			sb.WriteString("\n")
		}
	}

	hint := "enter to keep, esc to clear"
	if !m.filterInput {
		hint = "/ to edit, esc to clear"
	}
	sb.WriteString(styles.Muted.Render(hint))
	sb.WriteString("\n\n")
}

func (m Model) emptyFilterPrompt() string {
	if len(m.config.StatusFilterHints) > 0 {
		return "name or status"
	}
	return "name to match"
}

// Follows a status list that had to be cut short.
const truncationMarker = "…"

// Lists the status words that fit the pane, whole, so the line
// never ends mid-word, with a marker when more exist.
//
// A pane too narrow for even one of them still gets the bare qualifier, since
// knowing the filter takes a status at all is most of what the line is for, and
// a narrow pane is where filtering matters most.
func (m Model) statusHintLine() string {
	if len(m.config.StatusFilterHints) == 0 {
		return ""
	}

	available := m.leftPane.Width - 4
	if available <= 0 {
		return ""
	}

	line := ""
	shown := 0
	for _, status := range m.config.StatusFilterHints {
		candidate := StatusFilterPrefix + status
		if line != "" {
			candidate = line + "  " + candidate
		}
		if displayWidth(candidate) > available {
			break
		}
		line = candidate
		shown += 1
	}

	if shown == 0 {
		return truncatedStatusHint(available)
	}
	if shown < len(m.config.StatusFilterHints) &&
		displayWidth(line)+1+displayWidth(truncationMarker) <= available {
		line += " " + truncationMarker
	}
	return line
}

func truncatedStatusHint(available int) string {
	for _, candidate := range []string{
		StatusFilterPrefix + truncationMarker,
		StatusFilterPrefix,
	} {
		if displayWidth(candidate) <= available {
			return candidate
		}
	}
	return ""
}

// Counts the columns a string occupies rather than its bytes, so
// that a multi-byte marker is not mistaken for several columns.
func displayWidth(s string) int {
	return len([]rune(s))
}

// How many items would show with no filter, for the "n of m"
// counter.
func (m Model) totalItemCount() int {
	if m.config.SectionGrouper == nil {
		return len(m.items)
	}

	total := 0
	for _, section := range m.config.SectionGrouper.GroupItems(m.items, m.isExpandedForDisplay) {
		total += len(section.Items)
	}
	return total
}

func (m Model) renderGroupedItems(sb *strings.Builder) {
	sections := m.displaySections()
	if len(sections) == 0 {
		sb.WriteString(m.config.Styles.Muted.Render("No items match the filter"))
		sb.WriteString("\n")
		return
	}

	itemIndex := 0
	for i, section := range sections {
		if len(section.Items) == 0 {
			continue
		}

		sb.WriteString(m.config.Styles.Category.Render(section.Name))
		sb.WriteString("\n")
		sb.WriteString(m.config.Styles.Muted.Render(strings.Repeat("─", ui.SafeWidth(m.leftPane.Width-4))))
		sb.WriteString("\n")

		for _, item := range section.Items {
			line := m.renderItemLine(item, itemIndex == m.selectedIndex)
			sb.WriteString(line)
			sb.WriteString("\n")
			itemIndex += 1
		}

		if i < len(sections)-1 {
			sb.WriteString("\n")
		}
	}
}

func (m Model) renderFlatItems(sb *strings.Builder) {
	items := m.visibleItems()
	if len(items) == 0 && m.HasFilter() {
		sb.WriteString(m.config.Styles.Muted.Render("No items match the filter"))
		sb.WriteString("\n")
		return
	}

	for i, item := range items {
		line := m.renderItemLine(item, i == m.selectedIndex)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
}

// renderDefaultHeader renders the default header with title and optional breadcrumb.
func (m Model) renderDefaultHeader() string {
	sb := strings.Builder{}
	headerStyle := m.config.Styles.Header

	if len(m.navigationStack) > 0 {
		// Show breadcrumb navigation when drilled into an item
		sb.WriteString(headerStyle.Render("Details"))
		sb.WriteString("\n")

		// Build breadcrumb path
		breadcrumb := "← "
		for i, frame := range m.navigationStack {
			if i > 0 {
				breadcrumb += " > "
			}
			breadcrumb += frame.ParentName
		}
		sb.WriteString(m.config.Styles.Muted.Render(breadcrumb))
		sb.WriteString("\n")
	} else if m.config.Title != "" {
		sb.WriteString(headerStyle.Render(m.config.Title))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// renderItemLine renders a single item line in the left pane.
func (m Model) renderItemLine(item Item, selected bool) string {
	// Use unstyled icon when selected so the selection foreground color applies uniformly
	icon := item.GetIcon(!selected)

	// Calculate indentation based on depth
	// Each nested level adds 2 spaces for visual hierarchy
	depth := item.GetDepth()
	indent := "  "
	if depth > 0 {
		indent = strings.Repeat("  ", depth+1) + "└ "
	}

	// Type indicator for nested items
	typeIndicator := ""
	if depth > 0 || item.GetParentID() != "" {
		itemType := item.GetItemType()
		if itemType != "" {
			typeIndicator = fmt.Sprintf("[%s] ", strings.ToUpper(itemType[:1]))
		}
	}

	// For expandable items, show expand/collapse indicator
	// The indicator has to follow what is actually rendered where a filter shows the
	// children of every group, so a group cannot read as collapsed while its
	// children are listed underneath it.
	expandIndicator := ""
	effectiveDepth := depth + len(m.navigationStack)
	if item.IsExpandable() && effectiveDepth < m.config.MaxExpandDepth {
		if m.isExpandedForDisplay(item.GetID()) {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	}

	// A collapsed item's children are hidden, so say what is folded away.
	summary := ""
	if expandIndicator == "▶ " {
		if summariser, ok := item.(CollapsedSummariser); ok {
			if text := summariser.GetCollapsedSummary(); text != "" {
				summary = "  " + text
			}
		}
	}

	// Budget the name against everything else that shares the line, rather than
	// a flat reserve, so names are only shortened when they genuinely do not fit.
	action := item.GetAction()
	overhead := len(indent) + len(expandIndicator) + len(typeIndicator) +
		lipgloss.Width(icon) + 1 + len(summary) + actionGap + len(action)
	maxNameLen := max(m.leftPane.Width-4-overhead, 10)
	name := sdkstrings.TruncateString(item.GetName(), maxNameLen)

	line := fmt.Sprintf("%s%s%s%s %s%s", indent, expandIndicator, typeIndicator, icon, name, summary)

	// Pad to align action badges
	padding := m.leftPane.Width - 4 - lipgloss.Width(line) - len(action)
	if padding > 0 {
		line += strings.Repeat(" ", padding)
	}
	line += action

	if selected {
		return m.config.Styles.SelectedNavItem.Render(line)
	}

	return line
}

// renderRightPane renders the details for the selected item.
func (m Model) renderRightPane() string {
	item := m.SelectedItem()
	if item == nil {
		return m.config.Styles.Muted.Render("No item selected")
	}

	// Use the configured details renderer
	if m.config.DetailsRenderer != nil {
		return m.config.DetailsRenderer.RenderDetails(item, m.rightPane.Width, m.config.Styles)
	}

	// Default: show basic item info
	sb := strings.Builder{}
	sb.WriteString(m.config.Styles.Header.Render(item.GetName()))
	sb.WriteString("\n")
	sb.WriteString(m.config.Styles.Muted.Render(strings.Repeat("─", ui.SafeWidth(m.rightPane.Width-4))))
	sb.WriteString("\n\n")
	sb.WriteString(m.config.Styles.Muted.Render("Action: "))
	sb.WriteString(item.GetAction())
	sb.WriteString("\n")
	return sb.String()
}

// renderFooter renders the footer with navigation hints.
func (m Model) renderFooter() string {
	// Use custom footer renderer if provided
	if m.config.FooterRenderer != nil {
		return m.config.FooterRenderer.RenderFooter(&m, m.config.Styles)
	}

	return m.renderDefaultFooter()
}

// renderDefaultFooter renders the default footer with keyboard hints.
func (m Model) renderDefaultFooter() string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	keyStyle := m.config.Styles.Key

	// Show different footer when viewing a drilled-down item
	if len(m.navigationStack) > 0 {
		// Show breadcrumb path
		sb.WriteString(m.config.Styles.Muted.Render("  Viewing: "))
		for i, frame := range m.navigationStack {
			if i > 0 {
				sb.WriteString(m.config.Styles.Muted.Render(" > "))
			}
			sb.WriteString(m.config.Styles.Selected.Render(frame.ParentName))
		}
		sb.WriteString("\n\n")

		// Navigation help for drilled-down view
		sb.WriteString(m.config.Styles.Muted.Render("  "))
		sb.WriteString(keyStyle.Render("esc"))
		sb.WriteString(m.config.Styles.Muted.Render(" back  "))
		sb.WriteString(keyStyle.Render("↑/↓"))
		sb.WriteString(m.config.Styles.Muted.Render(" navigate  "))
		sb.WriteString(keyStyle.Render("enter"))
		sb.WriteString(m.config.Styles.Muted.Render(" expand/inspect  "))
		sb.WriteString(keyStyle.Render("tab"))
		sb.WriteString(m.config.Styles.Muted.Render(" switch pane  "))
		sb.WriteString(keyStyle.Render("q"))
		sb.WriteString(m.config.Styles.Muted.Render(" quit"))
		sb.WriteString("\n")

		return sb.String()
	}

	// Standard navigation help
	sb.WriteString(m.config.Styles.Muted.Render("  "))
	sb.WriteString(keyStyle.Render("↑/↓"))
	sb.WriteString(m.config.Styles.Muted.Render(" navigate  "))
	sb.WriteString(keyStyle.Render("enter"))
	sb.WriteString(m.config.Styles.Muted.Render(" expand/collapse  "))
	sb.WriteString(keyStyle.Render("tab"))
	sb.WriteString(m.config.Styles.Muted.Render(" switch pane  "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(m.config.Styles.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}

// scrollLeftPaneToSelection scrolls the left pane viewport to ensure
// the currently selected item is visible.
func (m *Model) scrollLeftPaneToSelection() {
	items := m.visibleItems()
	if len(items) == 0 {
		return
	}

	// If the first item is selected, always scroll to the very top
	if m.selectedIndex == 0 {
		m.leftPane.GotoTop()
		return
	}

	// Calculate line number for the selected item
	// This is a simplified calculation - for section-based layouts,
	// a more sophisticated calculation would be needed
	lineNumber := m.calculateSelectedLineNumber()

	viewportHeight := m.leftPane.Height
	currentOffset := m.leftPane.YOffset

	if lineNumber < currentOffset {
		m.leftPane.SetYOffset(lineNumber)
	} else if lineNumber >= currentOffset+viewportHeight {
		m.leftPane.SetYOffset(lineNumber - viewportHeight + 1)
	}
}

// calculateSelectedLineNumber calculates the line number of the selected item.
func (m Model) calculateSelectedLineNumber() int {
	// Start after header (title line, optional breadcrumb, empty line)
	lineNumber := 1
	if len(m.navigationStack) > 0 {
		lineNumber += 1 // Breadcrumb line
	}
	lineNumber += 1 // Empty line after header

	// The filter line adds the term, its hint and a blank line.
	if m.filterInput || m.HasFilter() {
		lineNumber += 3
	}

	if m.config.SectionGrouper != nil {
		sections := m.displaySections()
		itemIndex := 0
		for _, section := range sections {
			if len(section.Items) == 0 {
				continue
			}
			// Section header + separator = 2 lines
			lineNumber += 2

			for range section.Items {
				if itemIndex == m.selectedIndex {
					return lineNumber
				}
				lineNumber += 1
				itemIndex += 1
			}
			lineNumber += 1 // Empty line after section
		}
	} else {
		lineNumber += m.selectedIndex
	}

	return lineNumber
}
