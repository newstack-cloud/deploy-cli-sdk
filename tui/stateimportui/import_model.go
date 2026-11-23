package stateimportui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/jsonout"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stateutil"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// ImportModelConfig holds configuration for the import model.
type ImportModelConfig struct {
	EngineConfig         *stateio.EngineConfig
	FilePath             string
	Styles               *stylespkg.Styles
	Headless             bool
	HeadlessWriter       io.Writer
	JSONMode             bool
	FileEncryption       *stateio.StateEncryptionKeyConfig
	RequireEncryptedFile bool
	Force                bool
}

// ImportModel handles the import progress display.
type ImportModel struct {
	spinner              spinner.Model
	engineConfig         *stateio.EngineConfig
	filePath             string
	fileEncryption       *stateio.StateEncryptionKeyConfig
	requireEncryptedFile bool
	force                bool
	downloading          bool
	importing            bool
	result               *stateio.ImportResult
	err                  error
	finished             bool
	headless             bool
	headlessWriter       io.Writer
	jsonMode             bool
	styles               *stylespkg.Styles
	width                int
}

// NewImportModel creates a new import model.
func NewImportModel(config ImportModelConfig) *ImportModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = config.Styles.Spinner

	return &ImportModel{
		spinner:              s,
		engineConfig:         config.EngineConfig,
		filePath:             config.FilePath,
		fileEncryption:       config.FileEncryption,
		requireEncryptedFile: config.RequireEncryptedFile,
		force:                config.Force,
		headless:             config.Headless,
		headlessWriter:       config.HeadlessWriter,
		jsonMode:             config.JSONMode,
		styles:               config.Styles,
		width:                80, // Default width, will be updated on first WindowSizeMsg
	}
}

func (m *ImportModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ImportModel) Update(msg tea.Msg) (*ImportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case DownloadStartedMsg:
		m.downloading = true
		return m, nil
	case DownloadCompleteMsg:
		m.downloading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.finished = true
			return m, nil
		}
		m.importing = true
		return m, startImportWithDataCmd(m.engineConfig, msg.Data, m.importOptions())
	case ImportStartedMsg:
		m.importing = true
		return m, nil
	case ImportCompleteMsg:
		m.importing = false
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

func (m *ImportModel) View() string {
	if m.headless {
		return ""
	}

	if m.downloading {
		status := fmt.Sprintf("Downloading from %s...", m.filePath)
		return stateutil.RenderProgress(m.spinner.View(), status, m.width)
	}

	if m.importing {
		status := fmt.Sprintf("Importing into %s...", stateutil.DescribeEngineState(m.engineConfig))
		return stateutil.RenderProgress(m.spinner.View(), status, m.width)
	}

	if m.finished {
		return m.renderResult()
	}

	return ""
}

func (m *ImportModel) renderResult() string {
	if m.err != nil {
		// The input file is already shown above the result, in the main view's header.
		details := []stateutil.LabelledValue{{Label: "Target", Value: stateutil.DescribeEngineState(m.engineConfig)}}
		failure := stateutil.RenderFailure(m.styles, "Import failed", details, m.err, m.width)
		return failure + "\n  Press q to quit\n"
	}

	if m.result == nil {
		return "\n  Import completed with no result.\n\n  Press q to quit\n"
	}

	exportedAt := ""
	if !m.result.FileExportedAt.IsZero() {
		exportedAt = fmt.Sprintf("    File exported at: %s\n", formatExportTime(m.result.FileExportedAt))
	}

	return fmt.Sprintf("\n  %s Import complete\n\n%s\n    Instances imported: %d\n%s\n  Press q to quit\n",
		m.styles.Success.Render("✓"),
		sharedui.WrapLabelled("    Target: ", stateutil.DescribeEngineState(m.engineConfig), m.width),
		m.result.InstancesCount,
		exportedAt,
	)
}

func (m *ImportModel) writeHeadlessOutput() {
	if m.headlessWriter == nil {
		return
	}

	if m.jsonMode {
		m.writeJSONOutput()
		return
	}

	m.writeTextOutput()
}

func (m *ImportModel) writeJSONOutput() {
	if m.err != nil {
		jsonout.WriteJSON(m.headlessWriter, jsonout.NewErrorOutput(m.err))
		return
	}

	if m.result != nil {
		output := jsonout.StateImportOutput{
			Success:        m.result.Success,
			Mode:           "import",
			InstancesCount: m.result.InstancesCount,
			Message:        m.result.Message,
			Encrypted:      m.result.Encrypted,
		}
		if !m.result.FileExportedAt.IsZero() {
			output.FileExportedAt = m.result.FileExportedAt.UTC().Format(time.RFC3339)
		}
		jsonout.WriteJSON(m.headlessWriter, output)
	}
}

func (m *ImportModel) writeTextOutput() {
	if m.err != nil {
		details := []stateutil.LabelledValue{{Label: "Target", Value: stateutil.DescribeEngineState(m.engineConfig)}}
		fmt.Fprint(m.headlessWriter, stateutil.FormatFailureText("Import failed", details, m.err))
		return
	}

	if m.result != nil {
		fmt.Fprintf(m.headlessWriter, "%s\n", m.result.Message)
		fmt.Fprintf(m.headlessWriter, "Target: %s\n", stateutil.DescribeEngineState(m.engineConfig))
		if !m.result.FileExportedAt.IsZero() {
			fmt.Fprintf(m.headlessWriter, "File exported at: %s\n", formatExportTime(m.result.FileExportedAt))
		}
	}
}

// Shown so an older export than the one expected can be spotted before it is relied on.
func formatExportTime(exportedAt time.Time) string {
	return exportedAt.UTC().Format("2006-01-02 15:04:05 UTC")
}

// SetFilePath sets the file path for the import.
func (m *ImportModel) SetFilePath(filePath string) {
	m.filePath = filePath
}

// SetWidth sets the terminal width for the import model.
func (m *ImportModel) SetWidth(width int) {
	m.width = width
}

// IsFinished returns whether the import has finished.
func (m *ImportModel) IsFinished() bool {
	return m.finished
}

// StartImport returns a command to start the import process.
func (m *ImportModel) StartImport() tea.Cmd {
	if stateio.IsRemoteFile(m.filePath) {
		return startDownloadCmd(m.filePath)
	}
	return startImportCmd(m.engineConfig, m.filePath, m.importOptions())
}

func (m *ImportModel) importOptions() importOptions {
	return importOptions{
		fileEncryption:       m.fileEncryption,
		requireEncryptedFile: m.requireEncryptedFile,
		force:                m.force,
	}
}
