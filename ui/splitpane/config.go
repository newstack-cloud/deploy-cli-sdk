package splitpane

import "github.com/newstack-cloud/deploy-cli-sdk/styles"

// Config configures the split-pane model.
type Config struct {
	// Required
	Styles          *styles.Styles
	DetailsRenderer DetailsRenderer

	// Optional: title shown in the left pane header (default: none)
	Title string

	// Optional with defaults
	LeftPaneRatio  float64        // Default: 0.4
	MaxExpandDepth int            // Default: 2
	SectionGrouper SectionGrouper // Default: no sections
	HeaderRenderer HeaderRenderer // Default: standard header with title and breadcrumb
	FooterRenderer FooterRenderer // Default: standard keyboard hints

	// StatusKeywords reports the words a "status:" filter can match an item by,
	// usually derived from the status it is already showing. Leave nil to filter
	// by name only; a status filter then matches nothing rather than everything.
	StatusKeywords func(Item) []string

	// StatusFilterHints are the status words offered while a filter is being
	// typed. The pane does not know what statuses mean, so the vocabulary comes
	// from whoever supplied StatusKeywords. Ordered most useful first, since
	// only as many as fit the pane are shown.
	StatusFilterHints []string

	// Section names (if using default grouper)
	// e.g., {"resource": "Resources"}
	SectionNames map[string]string
}
