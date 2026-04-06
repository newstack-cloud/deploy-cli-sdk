package shared

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure ResourceGroupItem implements splitpane.Item.
var _ splitpane.Item = (*ResourceGroupItem)(nil)

// ResourceGroupItem is a synthetic navigation tree item that groups
// concrete cloud resources under their abstract type parent.
type ResourceGroupItem struct {
	Group         ResourceGroup
	Children      []splitpane.Item
	InternalLinks []splitpane.Item
}

func (g *ResourceGroupItem) GetID() string {
	return fmt.Sprintf("group:%s:%s", g.Group.GroupType, g.Group.GroupName)
}

func (g *ResourceGroupItem) GetName() string {
	return fmt.Sprintf("[%s] %s", g.Group.GroupType, g.Group.GroupName)
}

func (g *ResourceGroupItem) GetIcon(selected bool) string {
	return aggregateIcon(g.Children)
}

func (g *ResourceGroupItem) GetAction() string {
	return aggregateAction(g.Children)
}

func (g *ResourceGroupItem) GetDepth() int       { return 0 }
func (g *ResourceGroupItem) GetParentID() string { return "" }
func (g *ResourceGroupItem) GetItemType() string { return "resource" }
func (g *ResourceGroupItem) IsExpandable() bool  { return true }
func (g *ResourceGroupItem) CanDrillDown() bool  { return false }

func (g *ResourceGroupItem) GetChildren() []splitpane.Item {
	items := make([]splitpane.Item, 0, len(g.Children)+len(g.InternalLinks))
	items = append(items, g.Children...)
	items = append(items, g.InternalLinks...)
	return items
}

// higher priority = more severe; the most severe icon is shown on the group header
var iconPriority = map[string]int{
	IconFailed:           7,
	IconRollbackFailed:   6,
	IconRollingBack:      5,
	IconInterrupted:      4,
	IconInProgress:       3,
	IconPending:          2,
	IconSuccess:          1,
	IconSkipped:          0,
	IconNoChange:         -1,
	IconRollbackComplete: -1,
}

func aggregateIcon(children []splitpane.Item) string {
	best := IconNoChange
	bestPri := -2
	for _, child := range children {
		icon := child.GetIcon(false)
		if pri, ok := iconPriority[icon]; ok && pri > bestPri {
			best = icon
			bestPri = pri
		}
	}
	return best
}

// action priority: higher = more significant (shown on group header)
var actionPriority = map[ActionType]int{
	ActionDelete:   5,
	ActionRecreate: 4,
	ActionUpdate:   3,
	ActionCreate:   2,
	ActionNoChange: 0,
}

func aggregateAction(children []splitpane.Item) string {
	best := ActionNoChange
	bestPri := 0
	for _, child := range children {
		action := ActionType(child.GetAction())
		if pri, ok := actionPriority[action]; ok && pri > bestPri {
			best = action
			bestPri = pri
		}
	}
	return string(best)
}

// DepthAdjustedItem wraps a splitpane.Item to report a different depth
// while preserving all other behavior.
type DepthAdjustedItem struct {
	splitpane.Item
	AdjustedDepth int
}

func (d *DepthAdjustedItem) GetDepth() int { return d.AdjustedDepth }

// Unwrap returns the underlying item for type assertions in detail renderers.
func (d *DepthAdjustedItem) Unwrap() splitpane.Item { return d.Item }

// RenderGroupDetails renders the right-pane detail view for a ResourceGroupItem.
func RenderGroupDetails(group *ResourceGroupItem, width int, s *styles.Styles) string {
	sb := &strings.Builder{}
	RenderSectionHeader(sb, group.GetName(), width, s)
	RenderLabelValue(sb, "Abstract Type", group.Group.GroupType, s)
	RenderLabelValue(sb, "Resources", fmt.Sprintf("%d", len(group.Children)), s)
	if len(group.InternalLinks) > 0 {
		RenderLabelValue(sb, "Internal Links", fmt.Sprintf("%d", len(group.InternalLinks)), s)
	}
	sb.WriteString("\n")

	renderGroupChildList(sb, "Resources", group.Children, s)
	if len(group.InternalLinks) > 0 {
		renderGroupChildList(sb, "Internal Links", group.InternalLinks, s)
	}

	return sb.String()
}

func renderGroupChildList(sb *strings.Builder, title string, items []splitpane.Item, s *styles.Styles) {
	sb.WriteString("  " + s.Category.Render(title) + "\n")
	for _, item := range items {
		icon := item.GetIcon(false)
		action := item.GetAction()
		line := fmt.Sprintf("    %s %s", icon, item.GetName())
		if action != "" && action != string(ActionNoChange) {
			line += "  " + action
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n")
}
