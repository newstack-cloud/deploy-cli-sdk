package stateexportui

import (
	"errors"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// The output file of an export usually does not exist yet, so the selector asks for a
// path to write to rather than offering a picker of existing files.
func createSelectFileModel(styles *stylespkg.Styles) (*sharedui.SelectFileModel, error) {
	return sharedui.NewSelectFile(sharedui.SelectFileConfig{
		Action:          "export",
		Styles:          styles,
		FileTypeName:    "output file",
		Mode:            sharedui.SelectFileModeSave,
		DefaultFileName: "state-export.json",
		SupportedLocations: []string{
			consts.FileLocationLocal,
			consts.FileLocationS3,
			consts.FileLocationGCS,
			consts.FileLocationAzureBlob,
		},
	})
}

type stateExportSessionState uint32

const (
	stateExportFileSelect stateExportSessionState = iota
	stateExportRunning
	stateExportComplete
)

// MainModel is the top-level model for the state export command TUI.
type MainModel struct {
	sessionState stateExportSessionState
	filePath     string
	quitting     bool
	selectFile   tea.Model
	exportModel  *ExportModel
	styles       *stylespkg.Styles
	width        int
	Error        error
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	exportCmd := m.exportModel.Init()
	cmds = append(cmds, exportCmd)

	if m.selectFile != nil {
		fileCmd := m.selectFile.Init()
		cmds = append(cmds, fileCmd)
	}

	// If we're starting in running state (auto-export mode), start the export
	if m.sessionState == stateExportRunning {
		cmds = append(cmds, m.exportModel.StartExport())
	}

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sharedui.SelectFileMsg:
		return m.handleSelectFileMsg(msg)
	case sharedui.ClearSelectedFileMsg:
		return m.handleClearSelectedFileMsg()
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTickMsg(msg)
	case ExportCompleteMsg:
		return m.handleExportCompleteMsg(msg)
	}

	return m.handleSessionStateRouting(msg)
}

func (m MainModel) handleSelectFileMsg(msg sharedui.SelectFileMsg) (tea.Model, tea.Cmd) {
	// A completed remote file form keeps sending its selection, and only the first may
	// start an export.
	if m.sessionState != stateExportFileSelect {
		return m, nil
	}
	m.filePath = sharedui.ToFullFilePath(msg.File, msg.Location)
	m.sessionState = stateExportRunning
	m.exportModel.SetFilePath(m.filePath)
	return m, m.exportModel.StartExport()
}

func (m MainModel) handleClearSelectedFileMsg() (tea.Model, tea.Cmd) {
	m.sessionState = stateExportFileSelect
	m.filePath = ""
	return m, nil
}

func (m MainModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.exportModel.SetWidth(msg.Width)
	if m.selectFile != nil {
		var fileCmd tea.Cmd
		m.selectFile, fileCmd = m.selectFile.Update(msg)
		return m, fileCmd
	}
	return m, nil
}

func (m MainModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "q":
		if (m.sessionState == stateExportComplete && m.exportModel.IsFinished()) || m.Error != nil {
			m.quitting = true
			return m, tea.Quit
		}
	}
	// Every other key belongs to the active screen, e.g. the file location selector.
	return m.handleSessionStateRouting(msg)
}

func (m MainModel) handleSpinnerTickMsg(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.exportModel, cmd = m.exportModel.Update(msg)
	return m, cmd
}

func (m MainModel) handleExportCompleteMsg(msg ExportCompleteMsg) (tea.Model, tea.Cmd) {
	m.sessionState = stateExportComplete
	var cmd tea.Cmd
	m.exportModel, cmd = m.exportModel.Update(msg)
	if msg.Err != nil {
		m.Error = msg.Err
	}
	if m.exportModel.headless {
		m.quitting = true
		return m, tea.Quit
	}
	return m, cmd
}

func (m MainModel) handleSessionStateRouting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.sessionState {
	case stateExportFileSelect:
		if m.selectFile != nil {
			newSelectFile, newCmd := m.selectFile.Update(msg)
			selectFileModel, ok := newSelectFile.(sharedui.SelectFileModel)
			if !ok {
				m.Error = errors.New("internal error: unexpected select file model type in state export")
				return m, tea.Quit
			}
			m.selectFile = selectFileModel
			return m, newCmd
		}
	case stateExportRunning, stateExportComplete:
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.Update(msg)
		if m.exportModel.err != nil {
			m.Error = m.exportModel.err
		}
		return m, cmd
	}

	return m, nil
}

func (m MainModel) View() string {
	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("See you next time.")
	}

	switch m.sessionState {
	case stateExportFileSelect:
		if m.selectFile != nil {
			return m.selectFile.View()
		}
		return ""
	case stateExportRunning, stateExportComplete:
		header := sharedui.WrapLabelled("  Exporting to: ", m.styles.Selected.Render(m.filePath), m.width)
		selected := "\n" + header + "\n"
		return selected + m.exportModel.View()
	}

	return ""
}

// StateExportAppConfig holds configuration for creating a new state export app.
type StateExportAppConfig struct {
	FilePath        string
	InstanceFilters []string
	EngineConfig    *stateio.EngineConfig
	Styles          *stylespkg.Styles
	Headless        bool
	HeadlessWriter  io.Writer
	JSONMode        bool
	// FileEncryption is the key the output file is encrypted with, if any.
	FileEncryption *stateio.StateEncryptionKeyConfig
}

// NewStateExportApp creates a new state export application.
func NewStateExportApp(config StateExportAppConfig) (*MainModel, error) {
	// Determine if we're in auto-export mode (headless or file provided)
	autoExport := config.FilePath != "" || config.Headless

	// Determine the initial session state and create appropriate sub-models
	var sessionState stateExportSessionState
	var selectFile tea.Model

	if autoExport {
		// In headless/auto mode, go straight to running
		sessionState = stateExportRunning
	} else {
		// Interactive mode - start with file selection
		sessionState = stateExportFileSelect
		var err error
		selectFile, err = createSelectFileModel(config.Styles)
		if err != nil {
			return nil, err
		}
	}

	exportModel := NewExportModel(ExportModelConfig{
		EngineConfig:    config.EngineConfig,
		FilePath:        config.FilePath,
		InstanceFilters: config.InstanceFilters,
		Styles:          config.Styles,
		Headless:        config.Headless,
		HeadlessWriter:  config.HeadlessWriter,
		JSONMode:        config.JSONMode,
		FileEncryption:  config.FileEncryption,
	})

	return &MainModel{
		sessionState: sessionState,
		filePath:     config.FilePath,
		selectFile:   selectFile,
		exportModel:  exportModel,
		styles:       config.Styles,
		width:        80,
	}, nil
}
