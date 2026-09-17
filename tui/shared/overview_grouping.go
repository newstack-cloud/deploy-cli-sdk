package shared

import (
	"fmt"
	"sort"
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// OverviewGroup buckets overview entries that share one abstract resource group.
type OverviewGroup[T any] struct {
	// Group is the abstract resource the entries were expanded from,
	// or nil for the bucket of entries that belong to no group.
	Group *ResourceGroup
	// Entries holds the bucket's entries in their original order.
	Entries []T
}

// GroupOverviewEntries buckets entries by abstract resource group for the
// overview screens, mirroring the nesting used in the navigation tree.
// Buckets are ordered by group label with the ungrouped bucket last, and
// entry order is preserved within each bucket.
//
// When no entry belongs to a group, a single ungrouped bucket is returned so
// callers can render a flat list without special-casing.
func GroupOverviewEntries[T any](entries []T, groupOf func(T) *ResourceGroup) []OverviewGroup[T] {
	if len(entries) == 0 {
		return nil
	}

	buckets := make(map[string]*OverviewGroup[T])
	var labels []string
	var ungrouped []T

	for _, entry := range entries {
		group := groupOf(entry)
		if group == nil {
			ungrouped = append(ungrouped, entry)
			continue
		}

		label := ResourceGroupLabel(group)
		if _, exists := buckets[label]; !exists {
			buckets[label] = &OverviewGroup[T]{Group: group}
			labels = append(labels, label)
		}
		buckets[label].Entries = append(buckets[label].Entries, entry)
	}

	sort.Strings(labels)

	result := make([]OverviewGroup[T], 0, len(labels)+1)
	for _, label := range labels {
		result = append(result, *buckets[label])
	}

	if len(ungrouped) > 0 {
		result = append(result, OverviewGroup[T]{Entries: ungrouped})
	}

	return result
}

// ResourceGroupLabel returns the display label for an abstract resource group,
// matching the "[type] name" format used by group headers in the tree.
func ResourceGroupLabel(group *ResourceGroup) string {
	if group == nil {
		return ""
	}
	return fmt.Sprintf("[%s] %s", group.GroupType, group.GroupName)
}

// RenderOverviewGroupHeader writes the header line for an abstract resource
// group in an overview list. Nothing is written for the ungrouped bucket.
func RenderOverviewGroupHeader(sb *strings.Builder, group *ResourceGroup, indent string, s *styles.Styles) {
	if group == nil {
		return
	}
	sb.WriteString(indent)
	sb.WriteString(s.Category.Render(ResourceGroupLabel(group)))
	sb.WriteString("\n")
}

// OverviewEntryIndent returns the indent for an entry in an overview list,
// adding a level of nesting for entries that sit under a group header.
func OverviewEntryIndent(base string, group *ResourceGroup) string {
	if group == nil {
		return base
	}
	return base + "  "
}

// ParentElementPath returns the path of an element's parent, derived from a
// path previously built by BuildElementPath. It returns an empty string for
// top-level elements.
func ParentElementPath(path, elementType, elementName string) string {
	segment := elementType + "." + elementName
	if path == segment {
		return ""
	}
	return strings.TrimSuffix(path, "::"+segment)
}
