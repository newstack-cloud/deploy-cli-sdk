package stateexportui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/jsonout"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stateutil"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// ExportModelConfig holds configuration for the export model.
type ExportModelConfig struct {
	EngineConfig    *stateio.EngineConfig
	FilePath        string
	InstanceFilters []string
	Styles          *stylespkg.Styles
	Headless        bool
	HeadlessWriter  io.Writer
	JSONMode        bool
	FileEncryption  *stateio.StateEncryptionKeyConfig
}

// ExportModel handles the export progress display.
type ExportModel struct {
	spinner         spinner.Model
	engineConfig    *stateio.EngineConfig
	filePath        string
	instanceFilters []string
	fileEncryption  *stateio.StateEncryptionKeyConfig
	exporting       bool
	result          *stateio.ExportResult
	err             error
	finished        bool
	headless        bool
	headlessWriter  io.Writer
	jsonMode        bool
	styles          *stylespkg.Styles
	width           int
}

// NewExportModel creates a new export model.
func NewExportModel(config ExportModelConfig) *ExportModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = config.Styles.Spinner

	return &ExportModel{
		spinner:         s,
		engineConfig:    config.EngineConfig,
		filePath:        config.FilePath,
		instanceFilters: config.InstanceFilters,
		fileEncryption:  config.FileEncryption,
		headless:        config.Headless,
		headlessWriter:  config.HeadlessWriter,
		jsonMode:        config.JSONMode,
		styles:          config.Styles,
		width:           80,
	}
}

func (m *ExportModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ExportModel) Update(msg tea.Msg) (*ExportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ExportStartedMsg:
		m.exporting = true
		return m, nil
	case ExportCompleteMsg:
		m.exporting = false
		m.finished = true
		m.result = msg.Result
		m.err = msg.Err
		if m.headless {
			m.writeHeadlessOutput()
		}
		return m, nil
	}

	return m, nil
}

func (m *ExportModel) View() string {
	if m.headless {
		return ""
	}

	if m.exporting {
		// The output file is already shown above, in the main view's header.
		status := fmt.Sprintf("Exporting %s...", stateutil.DescribeEngineState(m.engineConfig))
		return stateutil.RenderProgress(m.spinner.View(), status, m.width)
	}

	if m.finished {
		return m.renderResult()
	}

	return ""
}

func (m *ExportModel) renderResult() string {
	if m.err != nil {
		// The output file is already shown above the result, in the main view's header.
		details := []stateutil.LabelledValue{{Label: "Source", Value: stateutil.DescribeEngineState(m.engineConfig)}}
		failure := stateutil.RenderFailure(m.styles, "Export failed", details, m.err, m.width)
		return failure + "\n  Press q to quit\n"
	}

	if m.result == nil {
		return "\n  Export completed with no result.\n\n  Press q to quit\n"
	}

	return fmt.Sprintf("\n  %s Export complete\n\n%s\n    Instances exported: %d\n%s\n%s\n  Press q to quit\n",
		m.styles.Success.Render("✓"),
		sharedui.WrapLabelled("    Source: ", stateutil.DescribeEngineState(m.engineConfig), m.width),
		m.result.InstancesCount,
		sharedui.WrapLabelled("    Output file: ", m.result.FilePath+encryptedSuffix(m.result.Encrypted), m.width),
		m.renderWarnings(),
	)
}

func encryptedSuffix(encrypted bool) string {
	if encrypted {
		return " (encrypted)"
	}
	return ""
}

func (m *ExportModel) renderWarnings() string {
	if len(m.result.Warnings) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, warning := range m.result.Warnings {
		prefix := "  " + m.styles.Warning.Render("!") + " "
		builder.WriteString("\n" + sharedui.WrapLabelled(prefix, warning, m.width) + "\n")
	}
	return builder.String()
}

func (m *ExportModel) writeHeadlessOutput() {
	if m.headlessWriter == nil {
		return
	}

	if m.jsonMode {
		m.writeJSONOutput()
		return
	}

	m.writeTextOutput()
}

func (m *ExportModel) writeJSONOutput() {
	if m.err != nil {
		jsonout.WriteJSON(m.headlessWriter, jsonout.NewErrorOutput(m.err))
		return
	}

	if m.result != nil {
		output := jsonout.StateImportOutput{
			Success:        m.result.Success,
			Mode:           "export",
			InstancesCount: m.result.InstancesCount,
			Message:        m.result.Message,
			Encrypted:      m.result.Encrypted,
			Warnings:       m.result.Warnings,
		}
		jsonout.WriteJSON(m.headlessWriter, output)
	}
}

func (m *ExportModel) writeTextOutput() {
	if m.err != nil {
		details := []stateutil.LabelledValue{{Label: "Source", Value: stateutil.DescribeEngineState(m.engineConfig)}}
		fmt.Fprint(m.headlessWriter, stateutil.FormatFailureText("Export failed", details, m.err))
		return
	}

	if m.result != nil {
		fmt.Fprintf(m.headlessWriter, "%s\n", m.result.Message)
		fmt.Fprintf(m.headlessWriter, "Source: %s\n", stateutil.DescribeEngineState(m.engineConfig))
		for _, warning := range m.result.Warnings {
			fmt.Fprintf(m.headlessWriter, "Warning: %s\n", warning)
		}
	}
}

// SetFilePath sets the file path for the export.
func (m *ExportModel) SetFilePath(filePath string) {
	m.filePath = filePath
}

// SetWidth sets the terminal width for the export model.
func (m *ExportModel) SetWidth(width int) {
	m.width = width
}

// IsFinished returns whether the export has finished.
func (m *ExportModel) IsFinished() bool {
	return m.finished
}

// StartExport returns a command to start the export process.
func (m *ExportModel) StartExport() tea.Cmd {
	return startExportCmd(m.engineConfig, m.filePath, m.instanceFilters, m.fileEncryption)
}
