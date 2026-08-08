package validateui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/diagutils"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"
)

// validateHeaderHeight is the number of rows the parent model prepends above
// the validate view (the blank line and "You selected blueprint: ..." line), so
// the viewport can reserve space for them and avoid overflowing the screen.
const validateHeaderHeight = 2

type ValidateResultMsg *types.BlueprintValidationEvent

type ValidateErrMsg struct {
	err error
}

type ValidateStreamMsg struct{}

type ValidateModel struct {
	spinner         spinner.Model
	viewport        viewport.Model
	hasDimensions   bool
	ctx             context.Context
	engine          engine.DeployEngine
	blueprintFile   string
	blueprintSource string
	resultStream    chan types.BlueprintValidationEvent
	collected       []*types.BlueprintValidationEvent
	errStream       chan error
	streaming       bool
	err             error
	width           int
	finished        bool
	// This is separate from an error in the sense that it indicates
	// that validation failed to the parent context to ensure that the program
	// exits with a non-zero exit code.
	// This is separate from the err field so that it allows us to render the
	// diagnostics in the TUI before exiting.
	validationFailed       bool
	logger                 *zap.Logger
	renderer               *glamour.TermRenderer
	headless               bool
	headlessWriter         io.Writer
	styles                 *stylespkg.Styles
	transformSpec          bool
	validateAfterTransform bool
	// operationConfig carries provider/transformer/context-variable values
	// (including the deploy target) sent to the engine in the validation payload.
	operationConfig *types.BlueprintOperationConfig
}

func (m ValidateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ValidateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		// Reserve rows for the blueprint header the parent model prepends, the
		// outcome summary line and the footer, so the viewport does not overflow
		// the screen and push those off the bottom.
		reserved := validateHeaderHeight +
			lipgloss.Height(m.summaryLine()) +
			lipgloss.Height(m.footerView())
		viewportHeight := max(1, msg.Height-reserved)

		if !m.hasDimensions {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.hasDimensions = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}
		if m.finished {
			m.viewport.SetContent(m.resultContents())
		}
	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = msg.BlueprintFile
		m.blueprintSource = msg.Source
		// SelectBlueprintMsg can be sent multiple times, we need to make sure we aren't collecting
		// duplicate results from the stream by not dispatching commands that will create multiple
		// consumers.
		if !m.streaming {
			cmds = append(cmds, startValidateStreamCmd(m, m.logger), waitForNextResultCmd(m), checkForErrCmd(m))
		}
		m.streaming = true
	case ValidateResultMsg:
		// The engine may send a terminal marker (End set, no diagnostic content)
		// purely to close the stream. For example, when a validation produced no
		// diagnostics. Collect only real diagnostics so the marker is not
		// rendered as a blank entry.
		if msg.Message != "" {
			m.collected = append(m.collected, msg)
		}
		cmds = append(cmds, checkForErrCmd(m))
		if !msg.End {
			cmds = append(cmds, waitForNextResultCmd(m))
		} else {
			m.finished = true
			m.validationFailed = checkForValidationFailure(m.collected)
			m.viewport.SetContent(m.resultContents())
			if m.headless {
				// Make sure we exit after validation completes in headless mode.
				cmds = append(cmds, tea.Quit)
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ValidateErrMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
	}

	var viewportCmd tea.Cmd
	m.viewport, viewportCmd = m.viewport.Update(msg)
	cmds = append(cmds, viewportCmd)

	return m, tea.Batch(cmds...)
}

func (m ValidateModel) footerView() string {
	hints := m.styles.Muted.Render(" ↑/↓ scroll • q quit ")
	b := lipgloss.RoundedBorder()
	b.Left = "┤"
	infoStyle := m.styles.Title.BorderStyle(b)
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(hints)-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, hints, line, info)
}

func (m ValidateModel) View() string {
	if m.headless {
		// In headless mode, print directly to configured writer and return empty string
		m.renderHeadless()
		return ""
	}

	if m.err != nil {
		return renderError(m.err, m.styles)
	}

	if !m.finished {
		return fmt.Sprintf("\n\n %s Validating project...\n\n", m.spinner.View())
	}

	// The diagnostics scroll inside the viewport; the outcome summary and the
	// footer hints are rendered outside it so they are always visible, even
	// when the validation produced no diagnostics.
	return fmt.Sprintf("%s\n%s\n%s", m.viewport.View(), m.summaryLine(), m.footerView())
}

func (m ValidateModel) resultContents() string {
	sb := strings.Builder{}

	for _, result := range m.collected {
		containerStyle := lipgloss.NewStyle().Padding(1, 1).Width(m.width)

		itemSB := strings.Builder{}
		itemSB.WriteString(m.styles.Category.Render("diagnostic"))

		levelOutput := renderDiagnosticLevel(result.Diagnostic.Level, m.styles)
		itemSB.WriteString(levelOutput)

		messageOutput, err := renderDiagnosticMessage(&result.Diagnostic, m.renderer, m.styles)
		if err != nil {
			return renderError(err, m.styles)
		}

		itemSB.WriteString(messageOutput)

		containerRendered := containerStyle.Render(itemSB.String())
		sb.WriteString(containerRendered)
	}

	return sb.String()
}

// Produces a one-line success/failure outcome for the completed
// validation. It is rendered outside the scrollable viewport (see View) so it
// is always visible, a successful validation may produce no diagnostics, and
// without this the screen would otherwise look empty.
func (m ValidateModel) summaryLine() string {
	errorCount, warningCount := 0, 0
	for _, result := range m.collected {
		switch result.Diagnostic.Level {
		case bpcore.DiagnosticLevelError:
			errorCount += 1
		case bpcore.DiagnosticLevelWarning:
			warningCount += 1
		}
	}

	if errorCount > 0 {
		return m.styles.Error.Padding(0, 1).Render(
			fmt.Sprintf("✗ Validation failed with %d error(s)%s", errorCount, warningSuffix(warningCount)),
		)
	}

	return m.styles.Success.Padding(0, 1).Render(
		fmt.Sprintf("✓ Validation succeeded%s", warningSuffix(warningCount)),
	)
}

func warningSuffix(warnings int) string {
	if warnings > 0 {
		return fmt.Sprintf(" (%d warning(s))", warnings)
	}
	return ""
}

func (m ValidateModel) renderHeadless() {
	if m.err != nil {
		fmt.Fprintln(m.headlessWriter, renderError(m.err, m.styles))
		return
	}

	fmt.Fprintf(m.headlessWriter, "Validating blueprint: %s\n\n", m.blueprintFile)

	for _, result := range m.collected {
		containerStyle := lipgloss.NewStyle().Padding(1, 1).Width(m.width)

		itemSB := strings.Builder{}
		itemSB.WriteString(m.styles.Category.Render("diagnostic"))

		levelOutput := renderDiagnosticLevel(result.Diagnostic.Level, m.styles)
		itemSB.WriteString(levelOutput)

		messageOutput, err := renderDiagnosticMessagePlain(&result.Diagnostic, m.styles)
		if err != nil {
			fmt.Fprintln(m.headlessWriter, renderError(err, m.styles))
			continue
		}

		itemSB.WriteString(messageOutput)

		containerRendered := containerStyle.Render(itemSB.String())
		fmt.Fprintln(m.headlessWriter, containerRendered)
	}

	if !m.finished {
		fmt.Fprintln(m.headlessWriter, "\nValidating project...")
	} else {
		fmt.Fprintln(m.headlessWriter, "\nValidation complete.")
	}
}

func renderDiagnosticMessagePlain(diagnostic *bpcore.Diagnostic, styles *stylespkg.Styles) (string, error) {
	sb := strings.Builder{}

	if diagnostic.Context == nil {
		sb.WriteString(styles.DiagnosticMessage.Render(diagnostic.Message))
		if hasPreciseRange(diagnostic.Range) {
			sb.WriteString(styles.Location.Render(
				fmt.Sprintf(
					"(line %d, column %d)",
					diagnostic.Range.Start.Line,
					diagnostic.Range.Start.Column,
				),
			))
		}
		return sb.String(), nil
	}

	sb.WriteString(styles.DiagnosticMessage.Render(string(diagnostic.Message)))
	suggestedActions, err := renderSuggestedActionsPlain(
		diagnostic.Context.SuggestedActions,
		diagnostic.Context.Metadata,
	)
	if err != nil {
		return sb.String(), err
	}

	sb.WriteString(suggestedActions)
	return sb.String(), nil
}

func renderSuggestedActionsPlain(
	suggestedActions []errors.SuggestedAction,
	diagMetadata map[string]any,
) (string, error) {
	sb := strings.Builder{}

	if len(suggestedActions) > 0 {
		sb.WriteString("\nSuggested Actions:\n")
	}

	for i, suggestedAction := range suggestedActions {
		fmt.Fprintf(&sb, "  %d. %s\n", i+1, suggestedAction.Title)
		fmt.Fprintf(&sb, "     %s\n", suggestedAction.Description)

		concreteAction := diagutils.GetConcreteAction(suggestedAction, diagMetadata)
		if concreteAction != nil {
			concreteActionOutput := concreteActionPlain(concreteAction)
			fmt.Fprintf(&sb, "     %s\n", concreteActionOutput)
		}

		if i < len(suggestedActions)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

func concreteActionPlain(concreteAction *diagutils.ConcreteAction) string {
	if concreteAction == nil {
		return ""
	}

	sb := strings.Builder{}
	if len(concreteAction.Commands) > 0 {
		sb.WriteString("Commands:\n")
		for _, command := range concreteAction.Commands {
			fmt.Fprintf(&sb, "  - %s\n", command)
		}
		sb.WriteString("\n")
	}

	if len(concreteAction.Links) > 0 {
		sb.WriteString("Links:\n")
		for _, link := range concreteAction.Links {
			fmt.Fprintf(&sb, "  - %s (%s)\n", link.Title, link.URL)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderDiagnosticMessage(diagnostic *bpcore.Diagnostic, renderer *glamour.TermRenderer, styles *stylespkg.Styles) (string, error) {
	sb := strings.Builder{}

	if diagnostic.Context == nil {
		sb.WriteString(styles.DiagnosticMessage.Render(diagnostic.Message))
		if hasPreciseRange(diagnostic.Range) {
			sb.WriteString(styles.Location.Render(
				fmt.Sprintf(
					"(line %d, column %d)",
					diagnostic.Range.Start.Line,
					diagnostic.Range.Start.Column,
				),
			))
		}
		return sb.String(), nil
	}

	sb.WriteString(styles.DiagnosticMessage.Render(string(diagnostic.Message)))
	suggestedActions, err := renderSuggestedActions(
		diagnostic.Context.SuggestedActions,
		renderer,
		diagnostic.Context.Metadata,
	)
	if err != nil {
		return sb.String(), err
	}

	sb.WriteString(styles.DiagnosticAction.Render(suggestedActions))
	return sb.String(), nil
}

func renderSuggestedActions(
	suggestedActions []errors.SuggestedAction,
	renderer *glamour.TermRenderer,
	diagMetadata map[string]any,
) (string, error) {
	sb := strings.Builder{}

	if len(suggestedActions) > 0 {
		sb.WriteString("\n# Suggested Actions\n")
	}

	for _, suggestedAction := range suggestedActions {
		markdown := suggestedActionMarkdown(suggestedAction, diagMetadata)
		sb.WriteString(markdown)

	}

	return renderer.Render(sb.String())
}

func suggestedActionMarkdown(suggestedAction errors.SuggestedAction, diagMetadata map[string]any) string {
	sb := strings.Builder{}

	fmt.Fprintf(&sb, "## %s\n\n%s\n", suggestedAction.Title, suggestedAction.Description)
	concreteAction := diagutils.GetConcreteAction(suggestedAction, diagMetadata)
	if concreteAction != nil {
		concreteActionOutput := concreteActionMarkdown(concreteAction)
		fmt.Fprintf(&sb, "\n%s\n", concreteActionOutput)
	}

	return sb.String()
}

func concreteActionMarkdown(concreteAction *diagutils.ConcreteAction) string {
	if concreteAction == nil {
		return ""
	}

	sb := strings.Builder{}
	if len(concreteAction.Commands) > 0 {
		sb.WriteString("Try running one of the following commands:\n")
		for _, command := range concreteAction.Commands {
			fmt.Fprintf(&sb, "  ```bash\n%s\n```\n", command)
		}
		sb.WriteString("\n")
	}

	if len(concreteAction.Links) > 0 {
		sb.WriteString("Try visiting one of the following links:\n\n")
		for _, link := range concreteAction.Links {
			fmt.Fprintf(&sb, "[%s](%s)\n", link.Title, link.URL)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderDiagnosticLevel(level bpcore.DiagnosticLevel, styles *stylespkg.Styles) string {
	switch level {
	case bpcore.DiagnosticLevelError:
		return styles.Error.MarginLeft(2).Render(
			diagnosticLevelName(level),
		)
	case bpcore.DiagnosticLevelWarning:
		return styles.Warning.MarginLeft(2).Render(
			diagnosticLevelName(level),
		)
	case bpcore.DiagnosticLevelInfo:
		return styles.Info.MarginLeft(2).Render(
			diagnosticLevelName(level),
		)
	default:
		return ""
	}
}

// ValidateModelConfig holds the configuration for creating a new ValidateModel.
type ValidateModelConfig struct {
	Context                context.Context
	Engine                 engine.DeployEngine
	Logger                 *zap.Logger
	Headless               bool
	HeadlessWriter         io.Writer
	Styles                 *stylespkg.Styles
	TransformSpec          bool
	ValidateAfterTransform bool
	// OperationConfig carries provider/transformer/context-variable values
	// (including the deploy target) sent to the engine in the validation payload.
	OperationConfig *types.BlueprintOperationConfig
}

// Returns the model's bound context, defaulting to context.Background()
// when none was supplied (e.g. models constructed directly in tests).
func (m ValidateModel) reqCtx() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func NewValidateModel(cfg ValidateModelConfig) ValidateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = cfg.Styles.Spinner
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return ValidateModel{
		spinner:                s,
		ctx:                    cfg.Context,
		engine:                 cfg.Engine,
		logger:                 cfg.Logger,
		resultStream:           make(chan types.BlueprintValidationEvent),
		errStream:              make(chan error),
		renderer:               renderer,
		headless:               cfg.Headless,
		headlessWriter:         cfg.HeadlessWriter,
		styles:                 cfg.Styles,
		transformSpec:          cfg.TransformSpec,
		validateAfterTransform: cfg.ValidateAfterTransform,
		operationConfig:        cfg.OperationConfig,
	}
}

// SetTransformSpec sets the transform spec option.
func (m *ValidateModel) SetTransformSpec(transformSpec bool) {
	m.transformSpec = transformSpec
}

// SetValidateAfterTransform sets the validate after transform option.
func (m *ValidateModel) SetValidateAfterTransform(validateAfterTransform bool) {
	m.validateAfterTransform = validateAfterTransform
}

// TransformSpec returns whether the transform spec option is enabled.
func (m ValidateModel) TransformSpec() bool {
	return m.transformSpec
}

// ValidateAfterTransform returns whether the validate-after-transform option is enabled.
func (m ValidateModel) ValidateAfterTransform() bool {
	return m.validateAfterTransform
}

func diagnosticLevelName(level bpcore.DiagnosticLevel) string {
	switch level {
	case bpcore.DiagnosticLevelError:
		return "error"
	case bpcore.DiagnosticLevelWarning:
		return "warning"
	case bpcore.DiagnosticLevelInfo:
		return "info"
	default:
		return "unknown"
	}
}

func hasPreciseRange(r *bpcore.DiagnosticRange) bool {
	return r != nil && r.Start.Line > 0 && r.Start.Column > 0
}

func renderError(err error, styles *stylespkg.Styles) string {
	sb := strings.Builder{}
	sb.WriteString(styles.Error.MarginLeft(2).Render(err.Error()))
	sb.WriteString("\n")
	return sb.String()
}

func checkForValidationFailure(diagnostics []*types.BlueprintValidationEvent) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Diagnostic.Level == bpcore.DiagnosticLevelError {
			return true
		}
	}
	return false
}
