package validateui

import (
	"errors"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/preflight"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"
)

// ValidateStage is an enum that represents the different stages
// of the validation process.
type ValidateStage int

const (
	// ValidateStageConfigStructure is the stage where application configuration
	// and project structure is validated.
	ValidateStageConfigStructure ValidateStage = iota
	// ValidateStageBlueprint is the stage where the blueprint is validated.
	ValidateStageBlueprint
	// ValidateStageSourceCode is the stage where the source code of the
	// application is validated.
	ValidateStageSourceCode
)

type validateSessionState uint32

const (
	validatePreflight validateSessionState = iota
	validateBlueprintSelect
	validateOptionsInput
	validateView
)

type MainModel struct {
	sessionState validateSessionState
	// validateStage   ValidateStage
	blueprintFile        string
	pendingSelectMsg     *sharedui.SelectBlueprintMsg
	quitting             bool
	selectBlueprint      tea.Model
	validate             tea.Model
	validateOptionsForm  *ValidateOptionsFormModel
	preflight            tea.Model
	autoValidate         bool
	needsOptionsInput    bool
	restartInstructions  string
	installedPlugins     []string
	preflightCommandName string
	styles               *stylespkg.Styles
	Error                error
}

func (m MainModel) Init() tea.Cmd {
	if m.sessionState == validatePreflight && m.preflight != nil {
		return m.preflight.Init()
	}
	cmds := []tea.Cmd{m.selectBlueprint.Init(), m.validate.Init()}
	if m.validateOptionsForm != nil {
		cmds = append(cmds, m.validateOptionsForm.Init())
	}
	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sessionState == validatePreflight {
		return m.updatePreflight(msg)
	}
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
		captured := msg
		m.pendingSelectMsg = &captured
		if m.needsOptionsInput {
			m.sessionState = validateOptionsInput
		} else {
			m.sessionState = validateView
			var cmd tea.Cmd
			m.validate, cmd = m.validate.Update(msg)
			cmds = append(cmds, cmd)
		}
	case ValidateOptionsSelectedMsg:
		m.sessionState = validateView
		validateModel := m.validate.(ValidateModel)
		validateModel.SetTransformSpec(msg.TransformSpec)
		validateModel.SetValidateAfterTransform(msg.ValidateAfterTransform)
		m.validate = validateModel
		if m.pendingSelectMsg != nil {
			selectMsg := *m.pendingSelectMsg
			var cmd tea.Cmd
			m.validate, cmd = m.validate.Update(selectMsg)
			cmds = append(cmds, cmd)
		}
	case sharedui.ClearSelectedBlueprintMsg:
		m.sessionState = validateBlueprintSelect
		m.blueprintFile = ""
	case tea.WindowSizeMsg:
		var bpCmd tea.Cmd
		m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
		var validateCmd tea.Cmd
		m.validate, validateCmd = m.validate.Update(msg)
		cmds = append(cmds, bpCmd, validateCmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.validate, cmd = m.validate.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.sessionState {
	case validateBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in validate")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)
	case validateOptionsInput:
		if m.validateOptionsForm != nil {
			var cmd tea.Cmd
			m.validateOptionsForm, cmd = m.validateOptionsForm.Update(msg)
			cmds = append(cmds, cmd)
		}
	case validateView:
		newValidate, newCmd := m.validate.Update(msg)
		validateModel, ok := newValidate.(ValidateModel)
		if !ok {
			panic("failed to perform assertion on validate model")
		}
		m.validate = validateModel
		cmds = append(cmds, newCmd)
		if validateModel.err != nil {
			m.Error = validateModel.err
		}
		if validateModel.validationFailed {
			m.Error = errors.New("validation failed")
		}
	}
	return m, tea.Batch(cmds...)
}

func (m MainModel) updatePreflight(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case preflight.SatisfiedMsg:
		if m.autoValidate {
			m.sessionState = validateView
		} else {
			m.sessionState = validateBlueprintSelect
		}
		cmds := []tea.Cmd{m.selectBlueprint.Init(), m.validate.Init()}
		if m.validateOptionsForm != nil {
			cmds = append(cmds, m.validateOptionsForm.Init())
		}
		return m, tea.Batch(cmds...)
	case preflight.InstalledMsg:
		m.restartInstructions = msg.RestartInstructions
		m.installedPlugins = msg.InstalledPlugins
		m.preflightCommandName = msg.CommandName
		m.quitting = true
		return m, tea.Quit
	case preflight.ErrorMsg:
		m.Error = msg.Err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	if m.preflight != nil {
		updated, cmd := m.preflight.Update(msg)
		m.preflight = updated
		return m, cmd
	}
	return m, nil
}

func (m MainModel) View() string {
	if m.quitting {
		if m.restartInstructions != "" {
			return preflight.RenderInstallSummary(
				m.styles, m.installedPlugins, len(m.installedPlugins),
				m.restartInstructions, m.preflightCommandName,
			)
		}
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Had enough? See you next time.")
	}
	if m.sessionState == validatePreflight {
		if m.preflight != nil {
			return m.preflight.View()
		}
		return ""
	}
	if m.sessionState == validateBlueprintSelect {
		return m.selectBlueprint.View()
	}
	if m.sessionState == validateOptionsInput {
		selected := "\n  You selected blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n\n"
		if m.validateOptionsForm != nil {
			return selected + m.validateOptionsForm.View()
		}
		return selected
	}

	selected := "\n  You selected blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
	return selected + m.validate.View()
}

// ValidateAppConfig holds configuration for creating a new validate application.
type ValidateAppConfig struct {
	Engine                 engine.DeployEngine
	Logger                 *zap.Logger
	BlueprintFile          string
	IsDefaultBlueprintFile bool
	Styles                 *stylespkg.Styles
	Headless               bool
	HeadlessWriter         io.Writer
	Preflight              tea.Model
	// TransformSpec, when non-nil, sets the validation loader's transform-spec
	// option directly and skips the interactive options form. When nil, the
	// SDK default of true is used and (in interactive mode) the form is shown.
	TransformSpec *bool
	// ValidateAfterTransform, when non-nil, sets the validation loader's
	// validate-after-transform option directly and skips the interactive
	// options form. When nil, the SDK default of false is used and (in
	// interactive mode) the form is shown.
	ValidateAfterTransform *bool
}

func NewValidateApp(cfg ValidateAppConfig) (*MainModel, error) {
	sessionState := validateBlueprintSelect
	// In headless mode, use the default blueprint file
	// if no explicit file is provided.
	autoValidate := (cfg.BlueprintFile != "" && !cfg.IsDefaultBlueprintFile) || cfg.Headless

	if autoValidate {
		sessionState = validateView
	}

	if cfg.Preflight != nil {
		sessionState = validatePreflight
	}

	fp, err := sharedui.BlueprintLocalFilePicker(cfg.Styles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		cfg.BlueprintFile,
		autoValidate,
		"validate",
		cfg.Styles,
		&fp,
	)
	if err != nil {
		return nil, err
	}

	transformSpec := boolFromPtr(cfg.TransformSpec, true)
	validateAfterTransform := boolFromPtr(cfg.ValidateAfterTransform, false)
	needsOptionsInput := !cfg.Headless &&
		cfg.TransformSpec == nil &&
		cfg.ValidateAfterTransform == nil

	validate := NewValidateModel(ValidateModelConfig{
		Engine:                 cfg.Engine,
		Logger:                 cfg.Logger,
		Headless:               cfg.Headless,
		HeadlessWriter:         cfg.HeadlessWriter,
		Styles:                 cfg.Styles,
		TransformSpec:          transformSpec,
		ValidateAfterTransform: validateAfterTransform,
	})

	var optionsForm *ValidateOptionsFormModel
	if needsOptionsInput {
		optionsForm = NewValidateOptionsFormModel(cfg.Styles, ValidateOptionsFormConfig{
			InitialTransformSpec:          transformSpec,
			InitialValidateAfterTransform: validateAfterTransform,
		})
	}

	return &MainModel{
		sessionState:        sessionState,
		blueprintFile:       cfg.BlueprintFile,
		selectBlueprint:     selectBlueprint,
		validate:            validate,
		validateOptionsForm: optionsForm,
		preflight:           cfg.Preflight,
		autoValidate:        autoValidate,
		needsOptionsInput:   needsOptionsInput,
		styles:              cfg.Styles,
	}, nil
}

func boolFromPtr(p *bool, defaultValue bool) bool {
	if p == nil {
		return defaultValue
	}
	return *p
}
