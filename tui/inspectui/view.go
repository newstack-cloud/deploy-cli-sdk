package inspectui

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/outpututil"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

func (m InspectModel) renderOverviewView() string {
	sb := strings.Builder{}

	// Header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.MarginLeft(2).Render("Instance Overview"))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.MarginLeft(2).Render(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	// Scrollable viewport content
	sb.WriteString(m.overviewViewport.View())
	sb.WriteString("\n")

	// Fixed footer with navigation help
	shared.RenderViewportScrollHints(&sb, "o", m.styles)

	return sb.String()
}

func (m *InspectModel) renderOverviewContent() string {
	if m.instanceState == nil {
		return m.styles.Muted.Render("No instance state available")
	}

	sb := strings.Builder{}

	m.renderOverviewInstanceInfo(&sb)
	m.renderOverviewResources(&sb)
	m.renderOverviewChildren(&sb)
	m.renderOverviewLinks(&sb)
	m.renderOverviewExports(&sb)
	m.renderOverviewDurations(&sb)

	return sb.String()
}

func (m *InspectModel) renderOverviewInstanceInfo(sb *strings.Builder) {
	sb.WriteString("\n")
	sb.WriteString(m.styles.Category.MarginLeft(2).Render("Instance Information"))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.MarginLeft(4).Render("Instance ID: "))
	sb.WriteString(m.instanceState.InstanceID)
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.MarginLeft(4).Render("Instance Name: "))
	sb.WriteString(m.instanceState.InstanceName)
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.MarginLeft(4).Render("Status: "))
	sb.WriteString(shared.RenderInstanceStatus(m.instanceState.Status, m.styles))
	sb.WriteString("\n\n")
}

func (m *InspectModel) renderOverviewResources(sb *strings.Builder) {
	if len(m.instanceState.Resources) == 0 {
		return
	}

	sb.WriteString(m.styles.Category.MarginLeft(2).Render(fmt.Sprintf("Resources (%d)", len(m.instanceState.Resources))))
	sb.WriteString("\n\n")

	for _, resourceState := range m.instanceState.Resources {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render(""))
		sb.WriteString(resourceState.Name)
		sb.WriteString(m.styles.Muted.Render(" ("))
		sb.WriteString(resourceState.Type)
		sb.WriteString(m.styles.Muted.Render(") - "))
		sb.WriteString(shared.RenderResourceStatus(resourceState.Status, m.styles))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (m *InspectModel) renderOverviewChildren(sb *strings.Builder) {
	if len(m.instanceState.ChildBlueprints) == 0 {
		return
	}

	sb.WriteString(m.styles.Category.MarginLeft(2).Render(fmt.Sprintf("Child Blueprints (%d)", len(m.instanceState.ChildBlueprints))))
	sb.WriteString("\n\n")

	for name, childState := range m.instanceState.ChildBlueprints {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render(""))
		sb.WriteString(name)
		sb.WriteString(m.styles.Muted.Render(" - "))
		sb.WriteString(shared.RenderInstanceStatus(childState.Status, m.styles))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (m *InspectModel) renderOverviewLinks(sb *strings.Builder) {
	if len(m.instanceState.Links) == 0 {
		return
	}

	sb.WriteString(m.styles.Category.MarginLeft(2).Render(fmt.Sprintf("Links (%d)", len(m.instanceState.Links))))
	sb.WriteString("\n\n")

	for linkName, linkState := range m.instanceState.Links {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render(""))
		sb.WriteString(linkName)
		sb.WriteString(m.styles.Muted.Render(" - "))
		sb.WriteString(shared.RenderLinkStatus(linkState.Status, m.styles))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (m *InspectModel) renderOverviewExports(sb *strings.Builder) {
	if len(m.instanceState.Exports) == 0 {
		return
	}

	sb.WriteString(m.styles.Category.MarginLeft(2).Render("Exports"))
	sb.WriteString("\n\n")

	fields := outpututil.CollectExportFieldsPretty(m.instanceState.Exports)
	for _, field := range fields {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render(field.Name + ": "))
		sb.WriteString(field.Value)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (m *InspectModel) renderOverviewDurations(sb *strings.Builder) {
	if m.instanceState.Durations == nil {
		return
	}

	durations := m.instanceState.Durations
	if (durations.PrepareDuration == nil || *durations.PrepareDuration <= 0) &&
		(durations.TotalDuration == nil || *durations.TotalDuration <= 0) {
		return
	}

	sb.WriteString(m.styles.Category.MarginLeft(2).Render("Timing"))
	sb.WriteString("\n\n")

	if durations.PrepareDuration != nil && *durations.PrepareDuration > 0 {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render("Prepare: "))
		sb.WriteString(outpututil.FormatDuration(*durations.PrepareDuration))
		sb.WriteString("\n")
	}
	if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render("Total: "))
		sb.WriteString(outpututil.FormatDuration(*durations.TotalDuration))
		sb.WriteString("\n")
	}
}

func (m InspectModel) renderSpecView() string {
	sb := strings.Builder{}

	// Header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.MarginLeft(2).Render("Resource Specification"))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.MarginLeft(2).Render(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	// Scrollable viewport content
	sb.WriteString(m.specViewport.View())
	sb.WriteString("\n")

	// Fixed footer with navigation help
	shared.RenderViewportScrollHints(&sb, "s", m.styles)

	return sb.String()
}

func (m *InspectModel) renderSpecContent(resourceState *state.ResourceState, resourceName string) string {
	if resourceState == nil || resourceState.SpecData == nil {
		return m.styles.Muted.Render("No specification data available")
	}

	sb := strings.Builder{}

	// Resource header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Category.MarginLeft(2).Render(resourceName))
	sb.WriteString("\n\n")

	// Spec fields (non-computed)
	specFields := outpututil.CollectNonComputedFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
	if len(specFields) > 0 {
		sb.WriteString(m.styles.Category.MarginLeft(2).Render("Specification"))
		sb.WriteString("\n\n")
		m.renderSpecFieldList(&sb, specFields)
		sb.WriteString("\n")
	}

	// Computed fields (outputs)
	outputFields := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(outputFields) > 0 {
		sb.WriteString(m.styles.Category.MarginLeft(2).Render("Outputs (Computed Fields)"))
		sb.WriteString("\n\n")
		m.renderSpecFieldList(&sb, outputFields)
	}

	return sb.String()
}

func (m *InspectModel) renderSpecFieldList(sb *strings.Builder, fields []outpututil.OutputField) {
	for _, field := range fields {
		if strings.Contains(field.Value, "\n") {
			sb.WriteString(m.styles.Muted.MarginLeft(4).Render(field.Name + ":"))
			sb.WriteString("\n")
			for line := range strings.SplitSeq(field.Value, "\n") {
				sb.WriteString("      ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(m.styles.Muted.MarginLeft(4).Render(field.Name + ": "))
			sb.WriteString(field.Value)
			sb.WriteString("\n")
		}
	}
}

func (m *InspectModel) renderError(err error) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Error.MarginLeft(2).Render("Error"))
	sb.WriteString("\n\n")
	sb.WriteString(m.styles.Muted.MarginLeft(4).Render(err.Error()))
	sb.WriteString("\n\n")
	sb.WriteString(m.styles.Muted.MarginLeft(2).Render("Press q to quit"))
	return sb.String()
}
