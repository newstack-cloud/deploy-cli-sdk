package shared

import (
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// HeadlessResourceInfo carries the data needed to group and print a resource
// in headless (non-interactive) output.
type HeadlessResourceInfo struct {
	// Path is the full path key (e.g. "childA/myFunc" or "myFunc" for top-level).
	Path string
	// Name is the resource name (last path segment).
	Name string
	// Metadata holds resource metadata with annotations for grouping.
	Metadata *state.ResourceMetadataState
}

// HeadlessResourceGroup holds resources that share the same abstract group.
type HeadlessResourceGroup struct {
	Group     ResourceGroup
	Resources []HeadlessResourceInfo
}

// GroupHeadlessResources partitions resources by abstract group.
// Resources whose metadata contains grouping annotations are placed in groups;
// others are returned as ungrouped. Group order is preserved (first seen).
func GroupHeadlessResources(
	resources []HeadlessResourceInfo,
) ([]HeadlessResourceGroup, []HeadlessResourceInfo) {
	type groupKey struct{ name, typ string }
	groupMap := make(map[groupKey]*HeadlessResourceGroup)
	var groupOrder []groupKey
	var ungrouped []HeadlessResourceInfo

	for _, res := range resources {
		rg := ExtractGrouping(res.Metadata)
		if rg == nil {
			ungrouped = append(ungrouped, res)
			continue
		}
		key := groupKey{rg.GroupName, rg.GroupType}
		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &HeadlessResourceGroup{Group: *rg}
			groupOrder = append(groupOrder, key)
		}
		groupMap[key].Resources = append(groupMap[key].Resources, res)
	}

	groups := make([]HeadlessResourceGroup, 0, len(groupOrder))
	for _, key := range groupOrder {
		g := groupMap[key]
		sort.Slice(g.Resources, func(i, j int) bool {
			return g.Resources[i].Name < g.Resources[j].Name
		})
		groups = append(groups, *g)
	}

	sort.Slice(ungrouped, func(i, j int) bool {
		return ungrouped[i].Name < ungrouped[j].Name
	})

	return groups, ungrouped
}

// SplitResourcesByPathLevel separates resources into those at the given path level
// (direct children of pathPrefix) and those nested deeper.
// For top-level resources, pass pathPrefix = "".
func SplitResourcesByPathLevel(
	resources []HeadlessResourceInfo,
	pathPrefix string,
) (atLevel, nested []HeadlessResourceInfo) {
	for _, res := range resources {
		rel := res.Path
		if pathPrefix != "" {
			if !strings.HasPrefix(res.Path, pathPrefix+"/") {
				continue
			}
			rel = strings.TrimPrefix(res.Path, pathPrefix+"/")
		}
		if strings.Contains(rel, "/") {
			nested = append(nested, res)
		} else {
			atLevel = append(atLevel, res)
		}
	}
	return
}
