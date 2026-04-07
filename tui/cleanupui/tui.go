package cleanupui

import (
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

var errCleanupFailed = errors.New("cleanup failed")

type cleanupSessionState uint32

const (
	cleanupOptionsForm cleanupSessionState = iota
	cleanupExecuting
	cleanupComplete
)

// MainModel is the top-level model for the cleanup TUI.
type MainModel struct {
	sessionState cleanupSessionState
	quitting     bool

	optionsForm *CleanupOptionsFormModel
	cleanup     *CleanupModel

	cleanupValidations           bool
	cleanupChangesets            bool
	cleanupReconciliationResults bool
	cleanupEvents                bool
	showOptionsForm              bool

	styles *stylespkg.Styles
	Error  error

	engine         engine.DeployEngine
	logger         *zap.Logger
	headless       bool
	headlessWriter io.Writer
}

func (m MainModel) Init() tea.Cmd {
	if m.showOptionsForm {
		return m.optionsForm.Init()
	}
	return m.cleanup.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case CleanupOptionsSelectedMsg:
		m.cleanupValidations = msg.Validations
		m.cleanupChangesets = msg.Changesets
		m.cleanupReconciliationResults = msg.ReconciliationResults
		m.cleanupEvents = msg.Events

		m.cleanup = NewCleanupModel(CleanupModelConfig{
			Engine:                       m.engine,
			Logger:                       m.logger,
			CleanupValidations:           m.cleanupValidations,
			CleanupChangesets:            m.cleanupChangesets,
			CleanupReconciliationResults: m.cleanupReconciliationResults,
			CleanupEvents:                m.cleanupEvents,
			Headless:                     m.headless,
			HeadlessWriter:               m.headlessWriter,
			Styles:                       m.styles,
		})
		m.sessionState = cleanupExecuting
		return m, m.cleanup.Init()

	case AllCleanupsDoneMsg:
		m.sessionState = cleanupComplete
		if m.cleanup.err != nil {
			m.Error = m.cleanup.err
		} else if m.cleanup.hasFailures {
			// Set Error to indicate some operations failed (for non-zero exit code)
			m.Error = errCleanupFailed
		}
		// In headless mode, quit immediately after completion
		if m.headless {
			return m, tea.Quit
		}
		return m, nil

	case CleanupErrorMsg:
		m.sessionState = cleanupComplete
		m.Error = msg.Err
		if m.cleanup != nil {
			m.cleanup.err = msg.Err
			m.cleanup.done = true
		}
		// In headless mode, quit immediately after error
		if m.headless {
			return m, tea.Quit
		}
		return m, nil
	}

	switch m.sessionState {
	case cleanupOptionsForm:
		newForm, cmd := m.optionsForm.Update(msg)
		m.optionsForm = newForm
		cmds = append(cmds, cmd)

	case cleanupExecuting:
		newCleanup, cmd := m.cleanup.Update(msg)
		if cm, ok := newCleanup.(*CleanupModel); ok {
			m.cleanup = cm
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Cleanup cancelled.")
	}

	switch m.sessionState {
	case cleanupOptionsForm:
		return m.optionsForm.View()
	case cleanupExecuting, cleanupComplete:
		return m.cleanup.View()
	}

	return ""
}

// CleanupAppConfig holds configuration for creating a new cleanup application.
type CleanupAppConfig struct {
	Engine                       engine.DeployEngine
	Logger                       *zap.Logger
	CleanupValidations           bool
	CleanupChangesets            bool
	CleanupReconciliationResults bool
	CleanupEvents                bool
	ShowOptionsForm              bool
	Styles                       *stylespkg.Styles
	Headless                     bool
	HeadlessWriter               io.Writer
}

// NewCleanupApp creates a new cleanup TUI application.
func NewCleanupApp(cfg CleanupAppConfig) (*MainModel, error) {
	model := &MainModel{
		cleanupValidations:           cfg.CleanupValidations,
		cleanupChangesets:            cfg.CleanupChangesets,
		cleanupReconciliationResults: cfg.CleanupReconciliationResults,
		cleanupEvents:                cfg.CleanupEvents,
		showOptionsForm:              cfg.ShowOptionsForm,
		styles:                       cfg.Styles,
		engine:                       cfg.Engine,
		logger:                       cfg.Logger,
		headless:                     cfg.Headless,
		headlessWriter:               cfg.HeadlessWriter,
	}

	if cfg.ShowOptionsForm {
		model.sessionState = cleanupOptionsForm
		model.optionsForm = NewCleanupOptionsFormModel(cfg.Styles)
	} else {
		model.sessionState = cleanupExecuting
		model.cleanup = NewCleanupModel(CleanupModelConfig{
			Engine:                       cfg.Engine,
			Logger:                       cfg.Logger,
			CleanupValidations:           cfg.CleanupValidations,
			CleanupChangesets:            cfg.CleanupChangesets,
			CleanupReconciliationResults: cfg.CleanupReconciliationResults,
			CleanupEvents:                cfg.CleanupEvents,
			Headless:                     cfg.Headless,
			HeadlessWriter:               cfg.HeadlessWriter,
			Styles:                       cfg.Styles,
		})
	}

	return model, nil
}
