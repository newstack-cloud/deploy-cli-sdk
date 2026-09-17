package shared

import (
	"fmt"
	"sort"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// SectionGrouper provides a generic implementation for grouping items
// into Resources, Child Blueprints, and Links sections.
// It works with any type implementing splitpane.Item.
// When items implement GroupableItem, resources are grouped under abstract type
// headers and each link is nested under the group it belongs to, leaving only
// links between two ungrouped resources in a section of their own.
type SectionGrouper struct {
	MaxExpandDepth int
}

type groupingResult struct {
	// Holds group headers and ungrouped resources sorted together
	// by name. Group children are not included; they are only materialised
	// by flattenGroups once links have been injected.
	topLevel       []splitpane.Item
	resourceGroups map[string]ResourceGroup // resource name → its abstract group
}

// GroupItems organizes items into sections using the splitpane.Item interface.
func (g *SectionGrouper) GroupItems(items []splitpane.Item, isExpanded func(id string) bool) []splitpane.Section {
	var resources, children, links []splitpane.Item

	for _, item := range items {
		if item.GetParentID() != "" {
			children = append(children, item)
			continue
		}
		switch item.GetItemType() {
		case "resource":
			resources = append(resources, item)
		case "child":
			children = append(children, item)
			children = g.appendExpandedChildren(children, item, isExpanded)
		case "link":
			links = append(links, item)
		}
	}

	gr := applyAbstractGrouping(resources)
	classified := classifyLinks(links, gr.resourceGroups)

	// Links must be injected before flattening so that they are materialised as
	// tree rows when their group is expanded.
	injectGroupLinks(gr.topLevel, classified.byGroup)

	return g.buildSections(
		flattenGroups(gr.topLevel, 0, isExpanded),
		anyGroupHoldsLinks(gr.topLevel),
		children,
		classified,
	)
}

func (g *SectionGrouper) buildSections(
	resources []splitpane.Item,
	resourcesHoldLinks bool,
	children []splitpane.Item,
	classified classifiedLinks,
) []splitpane.Section {
	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  resourceSectionName(resourcesHoldLinks),
			Items: resources,
		})
	}
	if len(children) > 0 {
		sections = append(sections, splitpane.Section{Name: "Child Blueprints", Items: children})
	}
	if len(classified.ungrouped) > 0 {
		SortItems(classified.ungrouped)
		sections = append(sections, splitpane.Section{Name: "Links", Items: classified.ungrouped})
	}

	return sections
}

func resourceSectionName(holdsLinks bool) string {
	if holdsLinks {
		return "Resources & Links"
	}
	return "Resources"
}

// Reports whether any group in the section owns links.
func anyGroupHoldsLinks(topLevel []splitpane.Item) bool {
	for _, item := range topLevel {
		if group, ok := item.(*ResourceGroupItem); ok && len(group.Links) > 0 {
			return true
		}
	}
	return false
}

// appendExpandedChildren recursively appends children of an expanded item.
// For child blueprints, it also applies abstract grouping to the child's resources.
func (g *SectionGrouper) appendExpandedChildren(
	result []splitpane.Item,
	item splitpane.Item,
	isExpanded func(id string) bool,
) []splitpane.Item {
	if isExpanded == nil || !isExpanded(item.GetID()) {
		return result
	}
	if item.GetDepth() >= g.MaxExpandDepth {
		return result
	}

	childItems := item.GetChildren()
	childResources, childChildren, childLinks := partitionByType(childItems)

	// Apply abstract grouping to resources at this child level
	gr := applyAbstractGrouping(childResources)
	cl := classifyLinks(childLinks, gr.resourceGroups)
	injectGroupLinks(gr.topLevel, cl.byGroup)

	for _, r := range flattenGroups(gr.topLevel, item.GetDepth()+1, isExpanded) {
		result = append(result, r)
		if r.IsExpandable() {
			result = g.appendExpandedChildren(result, r, isExpanded)
		}
	}

	SortItems(childChildren)
	for _, child := range childChildren {
		result = append(result, child)
		if child.IsExpandable() {
			result = g.appendExpandedChildren(result, child, isExpanded)
		}
	}

	// Append the links at this level that belong to no group
	SortItems(cl.ungrouped)
	result = append(result, cl.ungrouped...)

	return result
}

func partitionByType(items []splitpane.Item) (resources, children, links []splitpane.Item) {
	for _, item := range items {
		switch item.GetItemType() {
		case "resource":
			resources = append(resources, item)
		case "child":
			children = append(children, item)
		case "link":
			links = append(links, item)
		}
	}
	return
}

// Groups resources under abstract type headers.
// Resources implementing GroupableItem are nested; others pass through unchanged.
// The returned top-level list is sorted by name, mixing group headers and
// ungrouped resources; call flattenGroups to materialise expanded children.
func applyAbstractGrouping(resources []splitpane.Item) groupingResult {
	type groupKey struct{ name, typ string }
	groupMap := make(map[groupKey]*ResourceGroupItem)
	var groupOrder []groupKey
	var ungrouped []splitpane.Item
	resourceGroups := make(map[string]ResourceGroup)

	for _, item := range resources {
		rg := extractGroup(item)
		if rg == nil {
			ungrouped = append(ungrouped, item)
			continue
		}
		key := groupKey{rg.GroupName, rg.GroupType}
		resourceGroups[item.GetName()] = *rg

		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &ResourceGroupItem{Group: *rg}
			groupOrder = append(groupOrder, key)
		}
		groupMap[key].Children = append(groupMap[key].Children, item)
	}

	topLevel := make([]splitpane.Item, 0, len(groupOrder)+len(ungrouped))
	for _, key := range groupOrder {
		group := groupMap[key]
		SortItems(group.Children)
		topLevel = append(topLevel, group)
	}
	topLevel = append(topLevel, ungrouped...)
	SortItems(topLevel)

	return groupingResult{topLevel: topLevel, resourceGroups: resourceGroups}
}

// Expands the sorted top-level list into the final render order,
// placing each expanded group's children directly beneath their own header.
// Sorting must already have happened at the top level; sorting the flattened
// result would detach children from their parent group.
func flattenGroups(
	topLevel []splitpane.Item,
	baseDepth int,
	isExpanded func(id string) bool,
) []splitpane.Item {
	result := make([]splitpane.Item, 0, len(topLevel))
	for _, item := range topLevel {
		result = append(result, item)

		group, ok := item.(*ResourceGroupItem)
		if !ok || isExpanded == nil || !isExpanded(group.GetID()) {
			continue
		}
		result = appendGroupChildren(result, group, baseDepth+1)
	}
	return result
}

func appendGroupChildren(result []splitpane.Item, group *ResourceGroupItem, depth int) []splitpane.Item {
	for _, child := range group.Children {
		result = append(result, &DepthAdjustedItem{Item: child, AdjustedDepth: depth})
	}

	for _, link := range group.Links {
		result = append(result, &DepthAdjustedItem{
			Item:          link,
			AdjustedDepth: depth,
			DisplayName:   groupScopedLinkName(group, link),
		})
	}

	return result
}

// Shortens a link to the end that is not in this group,
// since naming the group's own resource again only costs width. An outbound
// arrow reads as "this group reaches", an inbound one as "this is reached by".
// A link with both ends here keeps both names, as neither is redundant.
func groupScopedLinkName(group *ResourceGroupItem, link splitpane.Item) string {
	classifiable, ok := link.(LinkClassifiable)
	if !ok {
		return ""
	}
	resourceA, resourceB := classifiable.GetLinkResourceNames()

	inGroup := make(map[string]bool, len(group.Children))
	for _, child := range group.Children {
		inGroup[child.GetName()] = true
	}

	switch {
	case inGroup[resourceA] && inGroup[resourceB]:
		return ""
	case inGroup[resourceA]:
		return LinkToName(resourceB)
	case inGroup[resourceB]:
		return LinkFromName(resourceA)
	}
	return ""
}

func extractGroup(item splitpane.Item) *ResourceGroup {
	groupable, ok := item.(GroupableItem)
	if !ok {
		return nil
	}
	return groupable.GetResourceGroup()
}

type classifiedLinks struct {
	byGroup   map[string][]splitpane.Item // group ID → links belonging to it
	ungrouped []splitpane.Item
}

// Decides which group each link belongs under.
//
// Links are directed, running out of resource A into resource B, so a link
// belongs with the abstract resource it originates from: reading a group then
// tells you what that resource connects to. A link out of an ungrouped resource
// into a grouped one still has somewhere sensible to live, so it falls back to
// the destination's group rather than being stranded at the top level.
func classifyLinks(
	links []splitpane.Item,
	resourceGroups map[string]ResourceGroup,
) classifiedLinks {
	result := classifiedLinks{byGroup: make(map[string][]splitpane.Item)}

	for _, link := range links {
		lc, ok := link.(LinkClassifiable)
		if !ok {
			result.ungrouped = append(result.ungrouped, link)
			continue
		}
		resourceA, resourceB := lc.GetLinkResourceNames()

		switch group := linkGroupID(resourceGroups, resourceA, resourceB); group {
		case "":
			result.ungrouped = append(result.ungrouped, link)
		default:
			result.byGroup[group] = append(result.byGroup[group], link)
		}
	}

	return result
}

func linkGroupID(
	resourceGroups map[string]ResourceGroup,
	resourceA string,
	resourceB string,
) string {
	source, hasSource := resourceGroups[resourceA]
	destination, hasDestination := resourceGroups[resourceB]

	if hasSource && source.IsAmbient() {
		return ambientLinkGroupID(source, destination, hasDestination)
	}
	if hasSource {
		return groupIDFor(source)
	}
	if hasDestination {
		return groupIDFor(destination)
	}
	return ""
}

// Places a link that runs out of an ambient resource.
func ambientLinkGroupID(
	source ResourceGroup,
	destination ResourceGroup,
	hasDestination bool,
) string {
	if !hasDestination {
		return ""
	}
	if destination.IsAmbient() {
		return groupIDFor(source)
	}
	return groupIDFor(destination)
}

func groupIDFor(group ResourceGroup) string {
	return fmt.Sprintf("group:%s:%s", group.GroupType, group.GroupName)
}

func injectGroupLinks(
	items []splitpane.Item,
	byGroup map[string][]splitpane.Item,
) {
	if len(byGroup) == 0 {
		return
	}
	for _, item := range items {
		group, ok := item.(*ResourceGroupItem)
		if !ok {
			continue
		}
		if links, found := byGroup[group.GetID()]; found {
			SortItems(links)
			group.Links = links
		}
	}
}

// SortItems sorts items alphabetically by name.
func SortItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}
