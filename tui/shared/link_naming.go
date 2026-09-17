package shared

import "strings"

// LinkNameSeparator joins the two resources of a link in its logical name,
// which is the identifier the engine uses: "{resourceA}::{resourceB}".
const LinkNameSeparator = "::"

// What a link reads as on screen. Links are directed,
// flowing from resource A to resource B, so an arrow says more than a symmetric
// glyph would. It matches the arrow already used for outbound links in the
// detail panes.
const linkDisplaySeparator = " " + linkOutboundMarker

const (
	linkOutboundMarker = "→ "
	linkInboundMarker  = "← "
)

// LinkToName renders a link as seen from its source: "→ destination".
func LinkToName(resourceB string) string {
	return linkOutboundMarker + resourceB
}

// LinkFromName renders a link as seen from its destination: "← source".
func LinkFromName(resourceA string) string {
	return linkInboundMarker + resourceA
}

// FormatLinkName renders a link for display. The logical name stays the
// identifier used for lookups and expansion; this is only what a person reads.
func FormatLinkName(resourceA, resourceB string) string {
	if resourceA == "" || resourceB == "" {
		// Nothing sensible to point between, so show whichever half exists.
		return resourceA + resourceB
	}
	return resourceA + linkDisplaySeparator + resourceB
}

// FormatLogicalLinkName renders a link given its logical "a::b" name, leaving
// anything that is not in that form untouched.
func FormatLogicalLinkName(linkName string) string {
	resourceA, resourceB, found := strings.Cut(linkName, LinkNameSeparator)
	if !found {
		return linkName
	}
	return FormatLinkName(resourceA, resourceB)
}
