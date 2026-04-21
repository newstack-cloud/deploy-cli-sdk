package stageui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stateutil"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure StageItem implements splitpane.Item
var _ splitpane.Item = (*StageItem)(nil)

// GetID returns a unique identifier for the item.
func (i *StageItem) GetID() string {
	return i.Name
}

// GetName returns the display name for the item.
func (i *StageItem) GetName() string {
	return i.Name
}

// GetIcon returns a status icon for the item.
// When selected is false, the icon is styled; when true, it's unstyled
// so the selection style can apply uniformly.
func (i *StageItem) GetIcon(selected bool) string {
	// This will be called with the styles from context
	// For now, return the plain icon - styling handled by IconWithStyles
	return i.getIconChar()
}

func (i *StageItem) getIconChar() string {
	switch i.Action {
	case ActionCreate:
		return "✓"
	case ActionUpdate:
		return "±"
	case ActionDelete:
		return "-"
	case ActionRecreate:
		return "↻"
	case ActionRetain:
		return shared.IconRetained
	default:
		return "○"
	}
}

// GetIconStyled returns a styled icon for the item.
func (i *StageItem) GetIconStyled(s *styles.Styles, styled bool) string {
	icon := i.getIconChar()
	if !styled {
		return icon
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch i.Action {
	case ActionCreate:
		return successStyle.Render(icon)
	case ActionUpdate:
		return s.Warning.Render(icon)
	case ActionDelete:
		return s.Error.Render(icon)
	case ActionRecreate:
		return s.Info.Render(icon)
	case ActionRetain:
		return s.Info.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

// GetAction returns the action badge text.
func (i *StageItem) GetAction() string {
	return string(i.Action)
}

// GetDepth returns the nesting depth for indentation.
func (i *StageItem) GetDepth() int {
	return i.Depth
}

// GetParentID returns the parent item ID.
func (i *StageItem) GetParentID() string {
	return i.ParentChild
}

// GetItemType returns the type for section grouping.
func (i *StageItem) GetItemType() string {
	return string(i.Type)
}

// GetResourceGroup returns the abstract resource group for this item, if any.
func (i *StageItem) GetResourceGroup() *shared.ResourceGroup {
	if i.Type != ItemTypeResource {
		return nil
	}
	if c, ok := i.Changes.(*provider.Changes); ok && c != nil {
		if rs := c.AppliedResourceInfo.CurrentResourceState; rs != nil {
			if g := shared.ExtractGrouping(rs.Metadata); g != nil {
				return g
			}
		}
	}
	if i.ResourceState != nil {
		return shared.ExtractGrouping(i.ResourceState.Metadata)
	}
	return nil
}

// GetLinkResourceNames returns the resource names for a link item.
func (i *StageItem) GetLinkResourceNames() (string, string) {
	if i.Type != ItemTypeLink {
		return "", ""
	}
	return parseLinkName(i.Name)
}

func parseLinkName(name string) (string, string) {
	parts := strings.SplitN(name, "::", 2)
	if len(parts) != 2 {
		return name, ""
	}
	return parts[0], parts[1]
}

// IsExpandable returns true if the item can be expanded in-place.
func (i *StageItem) IsExpandable() bool {
	return i.Type == ItemTypeChild && i.Changes != nil
}

// CanDrillDown returns true if the item can be drilled into.
func (i *StageItem) CanDrillDown() bool {
	if i.Type != ItemTypeChild {
		return false
	}
	_, ok := i.Changes.(*changes.BlueprintChanges)
	return ok
}

// GetChildren returns child items when expanded.
func (i *StageItem) GetChildren() []splitpane.Item {
	if i.Type != ItemTypeChild {
		return nil
	}
	childChanges, ok := i.Changes.(*changes.BlueprintChanges)
	if !ok || childChanges == nil {
		return nil
	}

	ctx := childItemContext{
		parentName:    i.Name,
		depth:         i.Depth + 1,
		instanceState: i.InstanceState,
	}

	// Track which items have been added from changes
	addedResources := make(map[string]bool)
	addedChildren := make(map[string]bool)

	var items []splitpane.Item
	items = appendResourceItems(items, childChanges, ctx, addedResources)
	items = appendChildItems(items, childChanges, ctx, addedChildren)
	items = appendNoChangeItemsFromState(items, ctx, addedResources, addedChildren)

	return items
}

type childItemContext struct {
	parentName    string
	depth         int
	instanceState *state.InstanceState
}

func appendResourceItems(
	items []splitpane.Item,
	childChanges *changes.BlueprintChanges,
	ctx childItemContext,
	added map[string]bool,
) []splitpane.Item {
	// New resources (no resource state since they're new)
	for name, rc := range childChanges.NewResources {
		rcCopy := rc
		resourceType, displayName := extractResourceTypeAndDisplayName(&rcCopy)
		items = append(items, &StageItem{
			Type:         ItemTypeResource,
			Name:         name,
			ResourceType: resourceType,
			DisplayName:  displayName,
			Action:       ActionCreate,
			Changes:      &rcCopy,
			New:          true,
			ParentChild:  ctx.parentName,
			Depth:        ctx.depth,
		})
		added[name] = true
	}

	// Changed resources - look up resource state from instance state
	for name, rc := range childChanges.ResourceChanges {
		rcCopy := rc
		resourceType, displayName := extractResourceTypeAndDisplayName(&rcCopy)
		action := determineResourceActionFromChanges(&rcCopy)

		// Look up resource state from instance state if available
		var resourceState *state.ResourceState
		if ctx.instanceState != nil {
			resourceState = findResourceState(ctx.instanceState, name)
		}

		items = append(items, &StageItem{
			Type:          ItemTypeResource,
			Name:          name,
			ResourceType:  resourceType,
			DisplayName:   displayName,
			Action:        action,
			Changes:       &rcCopy,
			Recreate:      rcCopy.MustRecreate,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			ResourceState: resourceState,
		})
		added[name] = true
	}

	// Removed resources - look up resource state for showing ID
	for _, name := range childChanges.RemovedResources {
		var resourceState *state.ResourceState
		if ctx.instanceState != nil {
			resourceState = findResourceState(ctx.instanceState, name)
		}

		items = append(items, &StageItem{
			Type:          ItemTypeResource,
			Name:          name,
			Action:        ActionDelete,
			Removed:       true,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			ResourceState: resourceState,
		})
		added[name] = true
	}

	// Retained resources - removed from state but underlying infra preserved.
	for _, name := range childChanges.RetainedResources {
		var resourceState *state.ResourceState
		if ctx.instanceState != nil {
			resourceState = findResourceState(ctx.instanceState, name)
		}

		items = append(items, &StageItem{
			Type:          ItemTypeResource,
			Name:          name,
			Action:        ActionRetain,
			Removed:       true,
			Retained:      true,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			ResourceState: resourceState,
		})
		added[name] = true
	}

	return items
}

var findResourceState = stateutil.FindResourceState

func appendChildItems(
	items []splitpane.Item,
	childChanges *changes.BlueprintChanges,
	ctx childItemContext,
	added map[string]bool,
) []splitpane.Item {
	// New children
	for name, nc := range childChanges.NewChildren {
		ncCopy := nc
		items = append(items, &StageItem{
			Type:   ItemTypeChild,
			Name:   name,
			Action: ActionCreate,
			Changes: &changes.BlueprintChanges{
				NewResources: ncCopy.NewResources,
				NewChildren:  ncCopy.NewChildren,
				NewExports:   ncCopy.NewExports,
			},
			New:         true,
			ParentChild: ctx.parentName,
			Depth:       ctx.depth,
		})
		added[name] = true
	}

	// Changed children - get nested instance state if available
	for name, cc := range childChanges.ChildChanges {
		ccCopy := cc
		var nestedInstanceState *state.InstanceState
		if ctx.instanceState != nil && ctx.instanceState.ChildBlueprints != nil {
			nestedInstanceState = ctx.instanceState.ChildBlueprints[name]
		}
		items = append(items, &StageItem{
			Type:          ItemTypeChild,
			Name:          name,
			Action:        ActionUpdate,
			Changes:       &ccCopy,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			InstanceState: nestedInstanceState,
		})
		added[name] = true
	}

	// Removed children
	for _, name := range childChanges.RemovedChildren {
		items = append(items, &StageItem{
			Type:        ItemTypeChild,
			Name:        name,
			Action:      ActionDelete,
			Removed:     true,
			ParentChild: ctx.parentName,
			Depth:       ctx.depth,
		})
		added[name] = true
	}

	return items
}

// appendNoChangeItemsFromState adds items from instance state that have no changes.
// This ensures all resources and children are visible in the navigation.
func appendNoChangeItemsFromState(
	items []splitpane.Item,
	ctx childItemContext,
	addedResources map[string]bool,
	addedChildren map[string]bool,
) []splitpane.Item {
	if ctx.instanceState == nil {
		return items
	}

	// Add resources from instance state that have no changes
	for _, resourceState := range ctx.instanceState.Resources {
		if addedResources[resourceState.Name] {
			continue
		}
		items = append(items, &StageItem{
			Type:          ItemTypeResource,
			Name:          resourceState.Name,
			ResourceType:  resourceState.Type,
			Action:        ActionNoChange,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			ResourceState: resourceState,
		})
	}

	// Add child blueprints from instance state that have no changes
	for name, childState := range ctx.instanceState.ChildBlueprints {
		if addedChildren[name] {
			continue
		}
		items = append(items, &StageItem{
			Type:          ItemTypeChild,
			Name:          name,
			Action:        ActionNoChange,
			ParentChild:   ctx.parentName,
			Depth:         ctx.depth,
			InstanceState: childState,
			// Provide empty changes so the child can still be expanded
			Changes: &changes.BlueprintChanges{},
		})
	}

	return items
}

// ToSplitPaneItems converts a slice of StageItems to splitpane.Items.
func ToSplitPaneItems(items []StageItem) []splitpane.Item {
	result := make([]splitpane.Item, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}

// determineResourceActionFromChanges determines the action for a resource based on its changes.
// This checks for recreation requirements, field changes, and outbound link changes.
func determineResourceActionFromChanges(changes *provider.Changes) ActionType {
	if changes.MustRecreate {
		return ActionRecreate
	}
	if provider.HasAnyChanges(changes) {
		return ActionUpdate
	}
	return ActionNoChange
}
