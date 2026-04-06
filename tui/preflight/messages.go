// Package preflight provides shared types for preflight check TUI integration.
// The actual preflight check implementation (e.g. plugin dependency verification)
// lives in the consuming CLI, but the message types and rendering utilities
// are shared so that deployment TUI models can handle preflight results
// without importing CLI-specific packages.
package preflight

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// SatisfiedMsg indicates all preflight checks passed (or were skipped).
type SatisfiedMsg struct{}

// InstalledMsg indicates dependencies were installed and the engine
// needs a restart before continuing.
type InstalledMsg struct {
	CommandName         string
	RestartInstructions string
	InstalledPlugins    []string
	InstalledCount      int
}

// ErrorMsg indicates the preflight check failed.
type ErrorMsg struct {
	Err error
}

// RenderInstallSummary renders the plugin installation summary for use
// in both the preflight complete view and parent TUI quitting views.
// The signature matches the original bluelink preflightui.RenderInstallSummary
// for compatibility.
func RenderInstallSummary(
	s *styles.Styles,
	plugins []string,
	installedCount int,
	restartInstructions string,
	commandName string,
) string {
	var sb strings.Builder

	sb.WriteString("\n  ")
	sb.WriteString(s.Muted.Render(
		"The deploy configuration requires plugin(s) that were not installed.",
	))
	sb.WriteString("\n  ")
	sb.WriteString(s.Selected.Render(
		fmt.Sprintf("%d missing plugin(s) installed:", installedCount),
	))

	sb.WriteString("\n\n")
	for _, p := range plugins {
		sb.WriteString("  ")
		sb.WriteString(s.Muted.Render("• "))
		sb.WriteString(p)
		sb.WriteString("\n")
	}

	sb.WriteString("\n  ")
	sb.WriteString(restartInstructions)
	if commandName != "" {
		sb.WriteString("\n  ")
		sb.WriteString(s.Muted.Render(
			fmt.Sprintf("Re-run the `%s` command after restarting the engine.", commandName),
		))
	}
	sb.WriteString("\n\n")
	return sb.String()
}
