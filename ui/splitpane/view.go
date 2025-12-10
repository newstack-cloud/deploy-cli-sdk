package splitpane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
)

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

	// Use custom header renderer if provided, otherwise use default
	if m.config.HeaderRenderer != nil {
		sb.WriteString(m.config.HeaderRenderer.RenderHeader(&m, m.config.Styles))
	} else {
		sb.WriteString(m.renderDefaultHeader())
	}

	// Get items, grouped if SectionGrouper is configured
	if m.config.SectionGrouper != nil {
		sections := m.config.SectionGrouper.GroupItems(m.items, m.IsExpanded)
		itemIndex := 0
		for i, section := range sections {
			if len(section.Items) == 0 {
				continue
			}

			// Section header
			sb.WriteString(m.config.Styles.Category.Render(section.Name))
			sb.WriteString("\n")
			sb.WriteString(m.config.Styles.Muted.Render(strings.Repeat("─", ui.SafeWidth(m.leftPane.Width-4))))
			sb.WriteString("\n")

			// Section items
			for _, item := range section.Items {
				line := m.renderItemLine(item, itemIndex == m.selectedIndex)
				sb.WriteString(line)
				sb.WriteString("\n")
				itemIndex += 1
			}

			// Add spacing between sections (not after the last one)
			if i < len(sections)-1 {
				sb.WriteString("\n")
			}
		}
	} else {
		// No grouper - render items flat
		for i, item := range m.items {
			line := m.renderItemLine(item, i == m.selectedIndex)
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
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
	expandIndicator := ""
	effectiveDepth := depth + len(m.navigationStack)
	if item.IsExpandable() && effectiveDepth < m.config.MaxExpandDepth {
		if m.expandedItems[item.GetID()] {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	}

	// Calculate max name length accounting for indentation, type indicator, and expand indicator
	maxNameLen := max(
		m.leftPane.Width-20-len(indent)-len(typeIndicator)-len(expandIndicator),
		10,
	)
	name := sdkstrings.TruncateString(item.GetName(), maxNameLen)

	line := fmt.Sprintf("%s%s%s%s %s", indent, expandIndicator, typeIndicator, icon, name)

	// Pad to align action badges
	action := item.GetAction()
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

	if m.config.SectionGrouper != nil {
		sections := m.config.SectionGrouper.GroupItems(m.items, m.IsExpanded)
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
