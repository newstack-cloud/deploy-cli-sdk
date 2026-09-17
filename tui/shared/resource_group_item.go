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
	Group ResourceGroup
	// Children holds the concrete resources expanded from the abstract resource.
	Children []splitpane.Item
	// Links holds the links that belong under this group, including those between two of
	// its own resources, and those running out of one of them to a resource
	// elsewhere.
	Links []splitpane.Item
}

func (g *ResourceGroupItem) GetID() string {
	return groupIDFor(g.Group)
}

func (g *ResourceGroupItem) GetName() string {
	return fmt.Sprintf("[%s] %s", g.Group.GroupType, g.Group.GroupName)
}

// GetIcon reports the most severe status among everything the group holds.
// Links count as a link that failed on the way out of this abstract resource is a
// failure of it, and reporting only the resources would show the group as
// healthy while it is not.
func (g *ResourceGroupItem) GetIcon(selected bool) string {
	return aggregateIcon(g.GetChildren())
}

// GetAction reports the most significant action among everything the group
// holds, links included, for the same reason as GetIcon.
func (g *ResourceGroupItem) GetAction() string {
	return aggregateAction(g.GetChildren())
}

func (g *ResourceGroupItem) GetDepth() int       { return 0 }
func (g *ResourceGroupItem) GetParentID() string { return "" }
func (g *ResourceGroupItem) GetItemType() string { return "resource" }
func (g *ResourceGroupItem) IsExpandable() bool  { return true }
func (g *ResourceGroupItem) CanDrillDown() bool  { return false }

// GetCollapsedSummary reports what the group holds while it is collapsed, so
// that resources and links folded into a group are still discoverable without
// expanding every group in the tree. The R and L markers match the type
// indicators the tree already uses for nested items, and it is kept terse
// because it shares the line with the group name.
func (g *ResourceGroupItem) GetCollapsedSummary() string {
	parts := make([]string, 0, 2)
	if count := len(g.Children); count > 0 {
		parts = append(parts, fmt.Sprintf("%dR", count))
	}
	if count := len(g.Links); count > 0 {
		parts = append(parts, fmt.Sprintf("%dL", count))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func (g *ResourceGroupItem) GetChildren() []splitpane.Item {
	items := make([]splitpane.Item, 0, len(g.Children)+len(g.Links))
	items = append(items, g.Children...)
	items = append(items, g.Links...)
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

// Shows the most severe of the children's icons. Children whose
// icon is not recognised carry no status to report, so when none of them do the
// group falls back to pending rather than claiming there is nothing to do.
func aggregateIcon(children []splitpane.Item) string {
	best := IconPending
	found := false
	bestPriority := 0
	for _, child := range children {
		icon := child.GetIcon(false)
		priority, ok := iconPriority[icon]
		if !ok {
			continue
		}
		if !found || priority > bestPriority {
			best = icon
			bestPriority = priority
			found = true
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

// Shows the most significant of the children's actions.
//
// Not every child has one, items built from a deploy event rather than the
// change set carry no action, and inspect mode uses an empty action throughout.
// "NO CHANGE" is a claim in its own right, so it is only reported when a child
// actually says so; a group with nothing to report shows no badge at all.
func aggregateAction(children []splitpane.Item) string {
	best := ActionType("")
	found := false
	bestPriority := 0
	for _, child := range children {
		action := ActionType(child.GetAction())
		priority, ok := actionPriority[action]
		if !ok {
			continue
		}
		if !found || priority > bestPriority {
			best = action
			bestPriority = priority
			found = true
		}
	}
	return string(best)
}

// DepthAdjustedItem wraps a splitpane.Item to report a different depth, and
// optionally a different name, while preserving all other behavior.
type DepthAdjustedItem struct {
	splitpane.Item
	AdjustedDepth int
	// DisplayName overrides the name for this position in the tree. A link
	// shown inside the group that owns it names only its far end, since the
	// near end is the group you are already looking at.
	DisplayName string
}

func (d *DepthAdjustedItem) GetDepth() int { return d.AdjustedDepth }

func (d *DepthAdjustedItem) GetName() string {
	if d.DisplayName != "" {
		return d.DisplayName
	}
	return d.Item.GetName()
}

// Unwrap returns the underlying item for type assertions in detail renderers.
func (d *DepthAdjustedItem) Unwrap() splitpane.Item { return d.Item }

// UnwrapItem returns the item underneath any display wrappers the navigation
// tree has put around it. Items nested under an abstract resource group are
// wrapped to adjust their depth and name, so a type assertion against the
// concrete item type fails unless it goes through here.
func UnwrapItem(item splitpane.Item) splitpane.Item {
	for {
		wrapped, ok := item.(interface{ Unwrap() splitpane.Item })
		if !ok {
			return item
		}
		item = wrapped.Unwrap()
	}
}

// RenderGroupDetails renders the right-pane detail view for a ResourceGroupItem.
func RenderGroupDetails(group *ResourceGroupItem, width int, s *styles.Styles) string {
	sb := &strings.Builder{}
	RenderSectionHeader(sb, group.GetName(), width, s)
	RenderLabelValue(sb, "Abstract Type", group.Group.GroupType, s)
	RenderLabelValue(sb, "Resources", fmt.Sprintf("%d", len(group.Children)), s)
	if len(group.Links) > 0 {
		RenderLabelValue(sb, "Links", fmt.Sprintf("%d", len(group.Links)), s)
	}
	sb.WriteString("\n")

	renderGroupChildList(sb, "Resources", group.Children, s)
	if len(group.Links) > 0 {
		renderGroupChildList(sb, "Links", group.Links, s)
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
