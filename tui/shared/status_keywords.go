package shared

import "github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"

// Filtering a large deployment by name only gets you so far: the question is
// usually "what failed" or "what is still running", not "where is this
// resource". The status an item is already showing through its icon answers
// that, so it is offered as a filter without every item type having to describe
// its status a second time.

// statusKeywords maps the status icon an item renders to the words a filter can
// match it by. Several words per status keeps the filter forgiving: "failed"
// and "failure" mean the same thing to someone scanning a deployment.
var statusKeywords = map[string][]string{
	IconPending:          {"pending", "queued", "waiting"},
	IconInProgress:       {"in-progress", "inprogress", "running", "deploying"},
	IconSuccess:          {"success", "succeeded", "ok", "done", "complete"},
	IconFailed:           {"failed", "failure", "error"},
	IconRollingBack:      {"rolling-back", "rollingback", "rollback"},
	IconRollbackFailed:   {"rollback-failed", "rollbackfailed", "failed", "failure"},
	IconRollbackComplete: {"rollback-complete", "rollbackcomplete", "rolledback"},
	IconInterrupted:      {"interrupted", "stopped"},
	IconSkipped:          {"skipped"},
	IconNoChange:         {"no-change", "nochange", "unchanged"},
	IconRetained:         {"retained"},
}

// StatusKeywordsForIcon returns the filter words matching a status icon.
func StatusKeywordsForIcon(icon string) []string {
	return statusKeywords[icon]
}

// StatusKeywordsForItem returns the filter words for an item, taken from the
// status icon it renders. Wire it into a split pane through
// splitpane.Config.StatusKeywords.
//
// A group header reports the most severe status among its contents rather than
// one of its own, so it is deliberately given no keywords, filtering by
// "failed" should show the failures inside a group, not every resource that
// happens to sit alongside them. The header is still shown, kept as the parent
// of what matched, with its aggregate icon marking the group as the one to
// look at.
func StatusKeywordsForItem(item splitpane.Item) []string {
	if _, isGroup := UnwrapItem(item).(*ResourceGroupItem); isGroup {
		return nil
	}
	return StatusKeywordsForIcon(item.GetIcon(false))
}

// KnownStatusKeywords lists every word a status filter accepts, sorted for a
// stable hint line.
func KnownStatusKeywords() []string {
	seen := map[string]bool{}
	for _, keywords := range statusKeywords {
		for _, keyword := range keywords {
			seen[keyword] = true
		}
	}

	// Only the primary spellings are worth suggesting; the rest are there to be
	// forgiving when typed, not to be advertised.
	primary := []string{
		"failed", "in-progress", "pending", "success",
		"skipped", "no-change", "interrupted", "rolling-back",
	}
	known := make([]string, 0, len(primary))
	for _, keyword := range primary {
		if seen[keyword] {
			known = append(known, keyword)
		}
	}
	return known
}
