package shared

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/outpututil"
)

// Duration rendering for the detail panes. Each renderer returns an empty
// string when there is nothing recorded, so callers can decide whether the
// section is worth a heading at all.

// RenderResourceDurations renders how long a resource took to deploy.
func RenderResourceDurations(durations *state.ResourceCompletionDurations, s *styles.Styles) string {
	if durations == nil {
		return ""
	}

	sb := strings.Builder{}
	writeDuration(&sb, "Config Complete", durations.ConfigCompleteDuration, s)
	writeDuration(&sb, "Total", durations.TotalDuration, s)
	writeAttemptDurations(&sb, durations.AttemptDurations, s)
	return sb.String()
}

// RenderLinkDurations renders how long a link took to deploy, broken down by
// the components the engine reports separately.
func RenderLinkDurations(durations *state.LinkCompletionDurations, s *styles.Styles) string {
	if durations == nil {
		return ""
	}

	sb := strings.Builder{}
	writeLinkComponentDuration(&sb, "Linked Resources Update", durations.LinkedResourcesUpdate, s)
	writeLinkComponentDuration(&sb, "Intermediary Resources", durations.IntermediaryResources, s)
	writeDuration(&sb, "Total", durations.TotalDuration, s)
	return sb.String()
}

// RenderInstanceDurations renders how long a blueprint instance took to deploy.
func RenderInstanceDurations(durations *state.InstanceCompletionDuration, s *styles.Styles) string {
	if durations == nil {
		return ""
	}

	sb := strings.Builder{}
	writeDuration(&sb, "Prepare", durations.PrepareDuration, s)
	writeDuration(&sb, "Total", durations.TotalDuration, s)
	return sb.String()
}

func writeLinkComponentDuration(
	sb *strings.Builder,
	label string,
	durations *state.LinkComponentCompletionDurations,
	s *styles.Styles,
) {
	if durations == nil {
		return
	}
	writeDuration(sb, label, durations.TotalDuration, s)
}

func writeDuration(sb *strings.Builder, label string, milliseconds *float64, s *styles.Styles) {
	if milliseconds == nil || *milliseconds <= 0 {
		return
	}
	sb.WriteString(s.Muted.Render(fmt.Sprintf(
		"  %s: %s", label, outpututil.FormatDuration(*milliseconds),
	)))
	sb.WriteString("\n")
}

// Lists per-attempt durations, which only tell you
// anything once a deployment has been retried.
func writeAttemptDurations(sb *strings.Builder, attempts []float64, s *styles.Styles) {
	if len(attempts) < 2 {
		return
	}
	formatted := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		formatted = append(formatted, outpututil.FormatDuration(attempt))
	}

	sb.WriteString(s.Muted.Render(fmt.Sprintf(
		"  Attempts: %s", strings.Join(formatted, ", "),
	)))
	sb.WriteString("\n")
}

// RenderTimingSection renders a labelled timing block, or nothing when there
// are no durations to show.
func RenderTimingSection(sb *strings.Builder, content string, s *styles.Styles) {
	if content == "" {
		return
	}

	sb.WriteString("\n")
	sb.WriteString(s.Category.Render("Timing:"))
	sb.WriteString("\n")
	sb.WriteString(content)
}
