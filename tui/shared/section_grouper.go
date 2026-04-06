package shared

import (
	"fmt"
	"sort"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// SectionGrouper provides a generic implementation for grouping items
// into Resources, Child Blueprints, and Links sections.
// It works with any type implementing splitpane.Item.
// When items implement GroupableItem, resources are grouped under
// abstract type headers and links are classified as internal or cross-group.
type SectionGrouper struct {
	MaxExpandDepth int
}

type groupingResult struct {
	items          []splitpane.Item
	resourceGroups map[string]string // resource name → group ID
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

	gr := applyAbstractGrouping(resources, isExpanded)
	classified := classifyLinks(links, gr.resourceGroups)

	// Inject internal links into their groups
	injectInternalLinks(gr.items, classified.internal)

	SortItems(gr.items)

	return g.buildSections(gr.items, children, classified)
}

func (g *SectionGrouper) buildSections(
	resources []splitpane.Item,
	children []splitpane.Item,
	classified classifiedLinks,
) []splitpane.Section {
	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{Name: "Resources", Items: resources})
	}
	if len(children) > 0 {
		sections = append(sections, splitpane.Section{Name: "Child Blueprints", Items: children})
	}
	if len(classified.crossGroup) > 0 {
		SortItems(classified.crossGroup)
		sections = append(sections, splitpane.Section{Name: "Cross-group Links", Items: classified.crossGroup})
	}
	if len(classified.ungrouped) > 0 {
		SortItems(classified.ungrouped)
		sections = append(sections, splitpane.Section{Name: "Links", Items: classified.ungrouped})
	}

	return sections
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
	gr := applyAbstractGroupingAtDepth(childResources, item.GetDepth()+1, isExpanded)
	injectInternalLinks(gr.items, classifyLinks(childLinks, gr.resourceGroups).internal)
	SortItems(gr.items)

	for _, r := range gr.items {
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

	// Append remaining links (cross-group and ungrouped from child level)
	cl := classifyLinks(childLinks, gr.resourceGroups)
	remaining := append(cl.crossGroup, cl.ungrouped...)
	SortItems(remaining)
	result = append(result, remaining...)

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

func applyAbstractGrouping(
	resources []splitpane.Item,
	isExpanded func(id string) bool,
) groupingResult {
	return applyAbstractGroupingAtDepth(resources, 0, isExpanded)
}

// applyAbstractGroupingAtDepth groups resources under abstract type headers.
// Resources implementing GroupableItem are nested; others pass through unchanged.
func applyAbstractGroupingAtDepth(
	resources []splitpane.Item,
	baseDepth int,
	isExpanded func(id string) bool,
) groupingResult {
	type groupKey struct{ name, typ string }
	groupMap := make(map[groupKey]*ResourceGroupItem)
	var groupOrder []groupKey
	var ungrouped []splitpane.Item
	resourceGroups := make(map[string]string)

	for _, item := range resources {
		rg := extractGroup(item)
		if rg == nil {
			ungrouped = append(ungrouped, item)
			continue
		}
		key := groupKey{rg.GroupName, rg.GroupType}
		groupID := fmt.Sprintf("group:%s:%s", rg.GroupType, rg.GroupName)
		resourceGroups[item.GetName()] = groupID

		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &ResourceGroupItem{Group: *rg}
			groupOrder = append(groupOrder, key)
		}
		groupMap[key].Children = append(groupMap[key].Children, item)
	}

	if len(groupMap) == 0 {
		return groupingResult{items: resources, resourceGroups: resourceGroups}
	}

	var result []splitpane.Item
	for _, key := range groupOrder {
		group := groupMap[key]
		SortItems(group.Children)
		result = append(result, group)
		if isExpanded != nil && isExpanded(group.GetID()) {
			result = appendGroupChildren(result, group, baseDepth+1)
		}
	}
	result = append(result, ungrouped...)

	return groupingResult{items: result, resourceGroups: resourceGroups}
}

func appendGroupChildren(result []splitpane.Item, group *ResourceGroupItem, depth int) []splitpane.Item {
	for _, child := range group.GetChildren() {
		result = append(result, &DepthAdjustedItem{Item: child, AdjustedDepth: depth})
	}
	return result
}

func extractGroup(item splitpane.Item) *ResourceGroup {
	groupable, ok := item.(GroupableItem)
	if !ok {
		return nil
	}
	return groupable.GetResourceGroup()
}

type classifiedLinks struct {
	internal   map[string][]splitpane.Item // group ID → internal links
	crossGroup []splitpane.Item
	ungrouped  []splitpane.Item
}

func classifyLinks(links []splitpane.Item, resourceGroups map[string]string) classifiedLinks {
	result := classifiedLinks{internal: make(map[string][]splitpane.Item)}

	for _, link := range links {
		lc, ok := link.(LinkClassifiable)
		if !ok {
			result.ungrouped = append(result.ungrouped, link)
			continue
		}
		resA, resB := lc.GetLinkResourceNames()
		groupA := resourceGroups[resA]
		groupB := resourceGroups[resB]

		switch {
		case groupA != "" && groupA == groupB:
			result.internal[groupA] = append(result.internal[groupA], link)
		case groupA != "" || groupB != "":
			result.crossGroup = append(result.crossGroup, link)
		default:
			result.ungrouped = append(result.ungrouped, link)
		}
	}

	return result
}

func injectInternalLinks(
	items []splitpane.Item,
	internal map[string][]splitpane.Item,
) {
	if len(internal) == 0 {
		return
	}
	for _, item := range items {
		group, ok := item.(*ResourceGroupItem)
		if !ok {
			continue
		}
		if links, found := internal[group.GetID()]; found {
			group.InternalLinks = links
		}
	}
}

// SortItems sorts items alphabetically by name.
func SortItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}
