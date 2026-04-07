package cleanupui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

func (m *CleanupModel) renderInteractive() string {
	if m.err != nil {
		return renderError(m.err, m.styles)
	}

	if m.done {
		return m.renderSummary()
	}

	return m.renderProgress()
}

func (m *CleanupModel) renderProgress() string {
	var sb strings.Builder
	sb.WriteString("\n")

	for i, step := range m.steps {
		if i < m.currentStepIndex {
			fmt.Fprintf(&sb, "  %s %s\n",
				m.styles.Success.Render("✓"),
				step.name)
		} else if i == m.currentStepIndex {
			fmt.Fprintf(&sb, "  %s Cleaning up %s...\n",
				m.spinner.View(), step.name)
		} else {
			fmt.Fprintf(&sb, "  %s %s\n",
				m.styles.Muted.Render("○"),
				m.styles.Muted.Render(step.name))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func (m *CleanupModel) renderSummary() string {
	var sb strings.Builder

	// Calculate totals
	var totalDeleted int64
	var failedCount int
	for _, op := range m.completedOps {
		totalDeleted += op.ItemsDeleted
		if op.Status == manage.CleanupOperationStatusFailed {
			failedCount += 1
		}
	}

	// Header - show different message if there were failures
	sb.WriteString("\n  ")
	if failedCount > 0 {
		sb.WriteString(m.styles.Warning.Render("!"))
		sb.WriteString(" Cleanup completed with errors\n\n")
	} else {
		sb.WriteString(m.styles.Success.Render("✓"))
		sb.WriteString(" Cleanup complete\n\n")
	}

	// Summary stats
	sb.WriteString(fmt.Sprintf("  Total items deleted: %s\n",
		strconv.FormatInt(totalDeleted, 10)))
	sb.WriteString(fmt.Sprintf("  Resource types processed: %d\n\n",
		len(m.completedOps)))

	// Details for each cleanup type
	sb.WriteString("  Details:\n")
	for _, op := range m.completedOps {
		statusIcon := "✓"
		statusStyle := m.styles.Success
		if op.Status == manage.CleanupOperationStatusFailed {
			statusIcon = "✗"
			statusStyle = m.styles.Error
		}

		itemsStr := strconv.FormatInt(op.ItemsDeleted, 10)
		durationStr := formatDuration(op.Duration())

		fmt.Fprintf(&sb, "    %s %-25s %s items deleted",
			statusStyle.Render(statusIcon),
			cleanupTypeName(op.CleanupType),
			itemsStr)

		if durationStr != "" {
			fmt.Fprintf(&sb, " (%s)", durationStr)
		}
		sb.WriteString("\n")

		if op.ErrorMessage != "" {
			fmt.Fprintf(&sb, "      %s\n",
				m.styles.Error.Render(op.ErrorMessage))
		}
	}

	// Footer with quit hint
	sb.WriteString("\n  ")
	sb.WriteString(m.styles.Muted.Render("Press q to quit"))
	sb.WriteString("\n")

	return sb.String()
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		return ""
	}
	if seconds == 0 {
		return "<1s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if secs == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dm%ds", minutes, secs)
}

func (m *CleanupModel) renderHeadless() {
	if m.err != nil {
		if !m.headlessSummaryPrinted {
			fmt.Fprintln(m.headlessWriter, "Cleanup failed")
			fmt.Fprintf(m.headlessWriter, "Error: %s\n", m.err.Error())
			m.headlessSummaryPrinted = true
		}
		return
	}

	if !m.done {
		// Only print progress for steps we haven't printed yet
		if m.currentStepIndex < len(m.steps) && m.currentStepIndex > m.headlessLastPrinted {
			step := m.steps[m.currentStepIndex]
			fmt.Fprintf(m.headlessWriter, "Cleaning up %s...\n", step.name)
			m.headlessLastPrinted = m.currentStepIndex
		}
		return
	}

	// Only print summary once
	if m.headlessSummaryPrinted {
		return
	}
	m.headlessSummaryPrinted = true

	m.printHeadlessSummary()
}

func (m *CleanupModel) printHeadlessSummary() {
	var totalDeleted int64
	var failedCount int
	for _, op := range m.completedOps {
		totalDeleted += op.ItemsDeleted
		if op.Status == manage.CleanupOperationStatusFailed {
			failedCount += 1
		}
	}

	if failedCount > 0 {
		fmt.Fprintln(m.headlessWriter, "Cleanup completed with errors")
	} else {
		fmt.Fprintln(m.headlessWriter, "Cleanup complete")
	}
	fmt.Fprintln(m.headlessWriter)
	fmt.Fprintf(m.headlessWriter, "Total items deleted: %s\n",
		strconv.FormatInt(totalDeleted, 10))
	fmt.Fprintf(m.headlessWriter, "Resource types processed: %d\n\n",
		len(m.completedOps))

	m.printHeadlessOperationDetails()
}

func (m *CleanupModel) printHeadlessOperationDetails() {
	fmt.Fprintln(m.headlessWriter, "Details:")
	for _, op := range m.completedOps {
		statusIcon := "[✓]"
		if op.Status == manage.CleanupOperationStatusFailed {
			statusIcon = "[✗]"
		}

		itemsStr := strconv.FormatInt(op.ItemsDeleted, 10)
		durationStr := formatDuration(op.Duration())

		line := fmt.Sprintf("  %s %-25s %s items deleted",
			statusIcon,
			cleanupTypeName(op.CleanupType),
			itemsStr)

		if durationStr != "" {
			line += fmt.Sprintf(" (%s)", durationStr)
		}
		fmt.Fprintln(m.headlessWriter, line)

		if op.ErrorMessage != "" {
			fmt.Fprintf(m.headlessWriter, "      Error: %s\n", op.ErrorMessage)
		}
	}

	fmt.Fprintln(m.headlessWriter)
}

func cleanupTypeName(ct manage.CleanupType) string {
	switch ct {
	case manage.CleanupTypeValidations:
		return "validations"
	case manage.CleanupTypeChangesets:
		return "changesets"
	case manage.CleanupTypeReconciliationResults:
		return "reconciliation results"
	case manage.CleanupTypeEvents:
		return "events"
	default:
		return string(ct)
	}
}

func renderError(err error, styles *stylespkg.Styles) string {
	var sb strings.Builder
	sb.WriteString("\n  ")
	sb.WriteString(styles.Error.Render("✗"))
	sb.WriteString(" Cleanup failed\n\n")
	fmt.Fprintf(&sb, "  %s\n\n", styles.Error.Render(err.Error()))
	sb.WriteString("  ")
	sb.WriteString(styles.Muted.Render("Press q to quit"))
	sb.WriteString("\n")
	return sb.String()
}
