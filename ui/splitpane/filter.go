package splitpane

import "strings"

// IsFiltering reports whether the filter input is currently accepting keys.
// Host models check this before handling their own single-key shortcuts, so
// that typing a search term does not trigger them.
func (m Model) IsFiltering() bool {
	return m.filterInput
}

// FilterTerm returns the active search term, empty when nothing is filtered.
func (m Model) FilterTerm() string {
	return m.filterTerm
}

// HasFilter reports whether a search term is narrowing the displayed items.
func (m Model) HasFilter() bool {
	return m.filterTerm != ""
}

// SetFilterTerm applies a search term, for callers driving the pane
// programmatically rather than through key presses.
func (m *Model) SetFilterTerm(term string) {
	m.filterTerm = term
	m.resolveSelectedIndex()
	if m.initialized {
		m.updateViewports()
	}
}

// StatusFilterPrefix qualifies a filter word as a status rather than a name,
// as in "status:failed".
const StatusFilterPrefix = "status:"

// Splits a search term into the statuses it asks for and the text
// it asks for, which are matched against different things.
type parsedFilter struct {
	statuses []string
	text     string
}

// Reads "status:failed orders" as "failed things matching
// orders". Several statuses widen the match rather than narrowing it, since
// "status:failed status:interrupted" reads as either, not both at once.
func parseFilterTerm(term string) parsedFilter {
	parsed := parsedFilter{}
	var textWords []string

	for word := range strings.FieldsSeq(term) {
		lowered := strings.ToLower(word)
		if status, found := strings.CutPrefix(lowered, StatusFilterPrefix); found {
			if status != "" {
				parsed.statuses = append(parsed.statuses, status)
			}
			continue
		}
		textWords = append(textWords, lowered)
	}

	parsed.text = strings.Join(textWords, " ")
	return parsed
}

// Reports whether an item satisfies the active search term.
//
// Group headers carry their abstract type and name in the display name
// ("[celerity/api] ordersApi"), so a single substring test covers searching by
// abstract type, by abstract resource name and by concrete item name alike.
// A "status:" word is matched against the status the item is showing instead.
func (m Model) itemMatchesFilter(item Item) bool {
	if m.filterTerm == "" {
		return true
	}

	parsed := parseFilterTerm(m.filterTerm)
	if !m.itemMatchesStatuses(item, parsed.statuses) {
		return false
	}

	if parsed.text == "" {
		return true
	}

	return strings.Contains(strings.ToLower(item.GetName()), parsed.text) ||
		strings.Contains(strings.ToLower(item.GetID()), parsed.text)
}

// Reports whether an item is in any of the given statuses.
//
// A status is matched as a prefix so the list narrows while the word is being
// typed, the way the name filter does: "status:fail" finds failures rather than
// nothing at all until the final "ed" is typed.
func (m Model) itemMatchesStatuses(item Item, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}

	if m.config.StatusKeywords == nil {
		// Nothing can describe its status, so a status filter matches nothing
		// rather than silently matching everything.
		return false
	}

	keywords := m.config.StatusKeywords(item)
	for _, wanted := range statuses {
		for _, keyword := range keywords {
			if strings.HasPrefix(keyword, wanted) {
				return true
			}
		}
	}

	return false
}

// Narrows each section, dropping any left with no items.
func (m Model) filterSections(sections []Section) []Section {
	if !m.HasFilter() {
		return sections
	}

	filtered := make([]Section, 0, len(sections))
	for _, section := range sections {
		items := filterItemsPreservingStructure(section.Items, m.itemMatchesFilter)
		if len(items) > 0 {
			filtered = append(filtered, Section{Name: section.Name, Items: items})
		}
	}
	return filtered
}

// Narrows a depth-ordered item list, keeping
// ancestors of matches and the full contents of a matching ancestor.
//
// The list is flat with nesting expressed through GetDepth, which is how the
// pane renders it, an item's descendants are the items that follow it until the
// depth returns to its own or shallower.
func filterItemsPreservingStructure(items []Item, matches func(Item) bool) []Item {
	if len(items) == 0 {
		return nil
	}

	selfMatch := make([]bool, len(items))
	for i, item := range items {
		selfMatch[i] = matches(item)
	}

	subtreeMatch := computeSubtreeMatches(items, selfMatch)

	kept := make([]Item, 0, len(items))
	// ancestorMatch[d] records whether the item at depth d enclosing the current
	// one matched in its own right, in which case its whole subtree is kept.
	ancestorMatch := map[int]bool{}
	for i, item := range items {
		depth := item.GetDepth()
		// Only levels above this one are ancestors. Entries at this depth or
		// deeper belong to preceding siblings and their subtrees, which enclose
		// nothing here.
		clearDeeperThan(ancestorMatch, depth-1)

		underMatchingAncestor := anyShallowerThan(ancestorMatch, depth)
		ancestorMatch[depth] = selfMatch[i] || underMatchingAncestor

		if subtreeMatch[i] || underMatchingAncestor {
			kept = append(kept, item)
		}
	}

	return kept
}

func computeSubtreeMatches(items []Item, selfMatch []bool) []bool {
	subtreeMatch := make([]bool, len(items))
	// pending[d] records a resolved match at depth d that has not yet been
	// attributed to an enclosing item.
	pending := map[int]bool{}

	for i := len(items) - 1; i >= 0; i-- {
		depth := items[i].GetDepth()
		descendantMatched := anyDeeperThan(pending, depth)
		subtreeMatch[i] = selfMatch[i] || descendantMatched

		// Reaching this item closes off everything nested under it.
		clearDeeperThan(pending, depth)
		if subtreeMatch[i] {
			pending[depth] = true
		}
	}

	return subtreeMatch
}

func anyDeeperThan(levels map[int]bool, depth int) bool {
	for level, set := range levels {
		if set && level > depth {
			return true
		}
	}
	return false
}

func anyShallowerThan(levels map[int]bool, depth int) bool {
	for level, set := range levels {
		if set && level < depth {
			return true
		}
	}
	return false
}

func clearDeeperThan(levels map[int]bool, depth int) {
	for level := range levels {
		if level > depth {
			delete(levels, level)
		}
	}
}
