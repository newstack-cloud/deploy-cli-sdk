package stateutil

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// LabelledValue is one line of a failure view, such as the state or file an import or
// export was working on.
type LabelledValue struct {
	Label string
	Value string
}

// RenderFailure renders an import or export failure for a TUI: the heading, the given
// details followed by the error and its cause as aligned labelled lines, and the hint
// on what to do. Every line is wrapped to fit within width.
func RenderFailure(
	styles *stylespkg.Styles,
	heading string,
	details []LabelledValue,
	err error,
	width int,
) string {
	explanation := stateio.ExplainError(err)
	lines := append([]LabelledValue{}, details...)
	lines = append(lines, LabelledValue{Label: "Error", Value: explanation.Summary})
	if explanation.Cause != "" {
		lines = append(lines, LabelledValue{Label: "Cause", Value: styles.Muted.Render(explanation.Cause)})
	}

	var view strings.Builder
	fmt.Fprintf(&view, "\n  %s %s\n\n", styles.Error.Render("✗"), heading)
	labelWidth := longestLabel(lines)
	for _, line := range lines {
		prefix := fmt.Sprintf("    %-*s ", labelWidth+1, line.Label+":")
		view.WriteString(sharedui.WrapLabelled(prefix, line.Value, width) + "\n")
	}
	if explanation.Hint != "" {
		view.WriteString("\n" + sharedui.WrapLabelled("    ", explanation.Hint, width) + "\n")
	}
	return view.String()
}

// FormatFailureText formats an import or export failure as plain text for headless
// output: the heading and error summary on the first line, then the given details, the
// cause and the hint, each indented under it.
func FormatFailureText(heading string, details []LabelledValue, err error) string {
	explanation := stateio.ExplainError(err)
	var text strings.Builder
	summary := strings.ReplaceAll(explanation.Summary, "\n", "\n  ")
	fmt.Fprintf(&text, "%s: %s\n", heading, summary)
	for _, detail := range details {
		fmt.Fprintf(&text, "  %s: %s\n", detail.Label, detail.Value)
	}
	if explanation.Cause != "" {
		fmt.Fprintf(&text, "  Cause: %s\n", explanation.Cause)
	}
	if explanation.Hint != "" {
		fmt.Fprintf(&text, "  %s\n", explanation.Hint)
	}
	return text.String()
}

func longestLabel(lines []LabelledValue) int {
	longest := 0
	for _, line := range lines {
		longest = max(longest, len(line.Label))
	}
	return longest
}
