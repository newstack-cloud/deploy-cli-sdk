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

	// Section names (if using default grouper)
	// e.g., {"resource": "Resources"}
	SectionNames map[string]string
}
