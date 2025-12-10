package headless

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// FormatMappingNode formats a MappingNode for display.
func FormatMappingNode(node *core.MappingNode) string {
	if node == nil {
		return "null"
	}

	// Handle scalar values
	if node.Scalar != nil {
		return FormatScalarValue(node.Scalar)
	}

	// Handle string values with substitutions
	if node.StringWithSubstitutions != nil {
		var sb strings.Builder
		sb.WriteString("\"")
		for _, v := range node.StringWithSubstitutions.Values {
			if v.StringValue != nil {
				sb.WriteString(*v.StringValue)
			} else if v.SubstitutionValue != nil {
				sb.WriteString("${...}")
			}
		}
		sb.WriteString("\"")
		return sb.String()
	}

	// Handle arrays
	if node.Items != nil {
		items := make([]string, 0, len(node.Items))
		for _, item := range node.Items {
			items = append(items, FormatMappingNode(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	}

	// Handle maps
	if node.Fields != nil {
		return "{...}"
	}

	return "unknown"
}

// FormatScalarValue formats a ScalarValue for display.
func FormatScalarValue(scalar *core.ScalarValue) string {
	if scalar == nil {
		return "null"
	}

	if scalar.StringValue != nil {
		return fmt.Sprintf("\"%s\"", *scalar.StringValue)
	}
	if scalar.IntValue != nil {
		return fmt.Sprintf("%d", *scalar.IntValue)
	}
	if scalar.FloatValue != nil {
		return fmt.Sprintf("%f", *scalar.FloatValue)
	}
	if scalar.BoolValue != nil {
		return fmt.Sprintf("%t", *scalar.BoolValue)
	}

	return "null"
}

// DiagnosticLevel represents the level of a diagnostic.
type DiagnosticLevel int

const (
	DiagnosticLevelError DiagnosticLevel = iota
	DiagnosticLevelWarning
	DiagnosticLevelInfo
)

// DiagnosticLevelName returns the display name for a diagnostic level.
func DiagnosticLevelName(level DiagnosticLevel) string {
	switch level {
	case DiagnosticLevelError:
		return "ERROR"
	case DiagnosticLevelWarning:
		return "WARNING"
	case DiagnosticLevelInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// DiagnosticLevelFromCore converts core.DiagnosticLevel to headless.DiagnosticLevel.
func DiagnosticLevelFromCore(level core.DiagnosticLevel) DiagnosticLevel {
	switch level {
	case core.DiagnosticLevelError:
		return DiagnosticLevelError
	case core.DiagnosticLevelWarning:
		return DiagnosticLevelWarning
	case core.DiagnosticLevelInfo:
		return DiagnosticLevelInfo
	default:
		return DiagnosticLevelError
	}
}
