package splitpane

import "github.com/newstack-cloud/deploy-cli-sdk/styles"

// Item represents a selectable item in the left pane.
// Consumers implement this interface for their domain-specific items.
type Item interface {
	// GetID provides a unique identifier that is primarily
	// used for expansion tracking in the split pane
	GetID() string
	// GetName provides a human-readable name for the item
	GetName() string
	// GetIcon provides a status icon (styled when not selected)
	GetIcon(selected bool) string
	// GetAction provides action badge text (e.g "CREATE", "UPDATE", etc.)
	GetAction() string
	// GetDepth provides the nesting depth for indentation
	GetDepth() int
	// GetParentID returns the parent item ID (empty for top-level items)
	GetParentID() string
	// GetItemType returns the type for section grouping (e.g., "resource", "child")
	GetItemType() string
	// IsExpandable indicates whether the item can be expanded in-place
	IsExpandable() bool
	// CanDrillDown indicates whether the item can be drilled into for detailed view
	CanDrillDown() bool
	// GetChildren provides a list of child items
	// displayed when the current item is expanded
	GetChildren() []Item
}

// Section groups items under a header in the left pane
type Section struct {
	Name  string
	Items []Item
}

// SectionGrouper organizes items into named sections.
// Optional - if nil, items render without section headers.
type SectionGrouper interface {
	GroupItems(items []Item, isExpanded func(id string) bool) []Section
}

// DetailsRenderer renders the right pane content for a selected item.
// Consumers implement this for domain-specific detail views.
type DetailsRenderer interface {
	RenderDetails(item Item, width int, styles *styles.Styles) string
}

// HeaderRenderer customizes the left pane header.
// Optional - if nil, uses default header with title and breadcrumb.
type HeaderRenderer interface {
	RenderHeader(model *Model, styles *styles.Styles) string
}

// FooterRenderer renders custom footer content.
// Optional - if nil, uses default keyboard hints footer.
type FooterRenderer interface {
	RenderFooter(model *Model, styles *styles.Styles) string
}

// NavigationFrame represents a frame in the drill-down navigation stack
type NavigationFrame struct {
	// ParentID is the identifier of the item drilled into
	ParentID string
	// ParentName is the display name for the breadcrumb
	ParentName string
	// SelectedID is the ID of the item that was selected before drilling down
	SelectedID string
	// Items are the items at this level (snapshot for back navigation)
	Items []Item
}

// --- Messages emitted by the split-pane model ---

// ItemSelectedMsg is sent when the selected item changes
type ItemSelectedMsg struct {
	Item Item
}

// ItemExpandedMsg is sent when an item is expanded/collapsed
type ItemExpandedMsg struct {
	Item     Item
	Expanded bool
}

// DrillDownMsg is sent when drilling into an item
type DrillDownMsg struct {
	Item Item
}

// BackMsg is sent when navigating back (Esc at root level)
type BackMsg struct{}

// QuitMsg is sent when the user presses q/ctrl+c
type QuitMsg struct{}

// ItemsUpdatedMsg can be sent to trigger a viewport refresh
// when items have been modified externally
type ItemsUpdatedMsg struct{}
