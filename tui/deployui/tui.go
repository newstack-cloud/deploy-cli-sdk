package deployui

import (
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"

	"github.com/newstack-cloud/deploy-cli-sdk/tui/driftui"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/preflight"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stageui"
)

type deploySessionState uint32

const (
	// deployPreflight - preflight plugin dependency check
	deployPreflight deploySessionState = iota
	// deployBlueprintSelect - select blueprint file if not provided
	deployBlueprintSelect
	// deployConfigInput - combined config form for instance name, rollback, staging options
	deployConfigInput
	// deployStaging - run change staging (when --stage is set)
	deployStaging
	// deployExecute - main deployment view with split-pane from the start
	deployExecute
)

// MainModel is the top-level model for the deploy command TUI.
// It manages the session state and delegates to sub-models.
type MainModel struct {
	sessionState deploySessionState
	quitting     bool

	// Sub-models
	selectBlueprint  tea.Model
	deployConfigForm *DeployConfigFormModel
	staging          *stageui.StageModel
	deploy           tea.Model

	// Config from flags
	changesetID         string
	instanceID          string
	instanceName        string
	blueprintFile       string
	blueprintSource     string
	isDefaultBlueprint  bool
	autoRollback        bool
	force               bool
	stageFirst          bool
	autoApprove         bool
	autoApproveCodeOnly bool
	skipPrompts         bool

	// Preflight
	preflight          tea.Model
	postPreflightState deploySessionState

	// Runtime state
	headless             bool
	jsonMode             bool
	restartInstructions  string
	installedPlugins     []string
	preflightCommandName string
	engine               engine.DeployEngine
	logger               *zap.Logger

	styles *stylespkg.Styles
	Error  error
}

// GetInstanceID returns the instance ID for the InstanceResolver interface.
func (m MainModel) GetInstanceID() string { return m.instanceID }

// GetInstanceName returns the instance name for the InstanceResolver interface.
func (m MainModel) GetInstanceName() string { return m.instanceName }

// GetEngine returns the engine for the InstanceResolver interface.
func (m MainModel) GetEngine() shared.InstanceLookup { return m.engine }

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	if m.sessionState == deployPreflight && m.preflight != nil {
		return m.preflight.Init()
	}
	bpCmd := m.selectBlueprint.Init()
	configFormCmd := m.deployConfigForm.Init()
	deployCmd := m.deploy.Init()
	var stagingCmd tea.Cmd
	if m.staging != nil {
		stagingCmd = m.staging.Init()
	}
	cmds := []tea.Cmd{bpCmd, configFormCmd, deployCmd, stagingCmd}
	if m.sessionState == deployExecute {
		cmds = append(cmds, func() tea.Msg {
			return StartDeployMsg{}
		})
	}
	return tea.Batch(cmds...)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sessionState == deployPreflight {
		return m.updatePreflight(msg)
	}

	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		return m.handleSelectBlueprintMsg(msg)
	case DeployConfigMsg:
		return m.handleDeployConfigMsg(msg)
	case InstanceResolvedMsg:
		return m.handleInstanceResolvedMsg(msg)
	case stageui.StageCompleteMsg:
		return m.handleStageCompleteMsg(msg)
	case ConfirmDeployMsg:
		return m.handleConfirmDeployMsg(msg)
	case StartDeployMsg:
		return m.handleStartDeployMsg(msg)
	case sharedui.ClearSelectedBlueprintMsg:
		return m.handleClearSelectedBlueprintMsg()
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case tea.KeyMsg:
		newModel, cmd, handled := m.handleKeyMsg(msg)
		if handled {
			return newModel, cmd
		}
	case spinner.TickMsg:
		return m.handleSpinnerTickMsg(msg)
	case stageui.StageStartedMsg, stageui.StageEventMsg, stageui.StageErrorMsg:
		return m.handleStagingEventMsg(msg)
	case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
		return m.handleDriftMsg(msg)
	}

	return m.handleSessionStateRouting(msg)
}

func (m *MainModel) handleStageComplete(
	msg stageui.StageCompleteMsg,
	cmds []tea.Cmd,
) (tea.Model, tea.Cmd) {
	m.changesetID = msg.ChangesetID
	m.propagateChangesetToDeployModel(msg)
	approvalCmds := m.resolveApprovalAction(msg)
	cmds = append(cmds, approvalCmds...)
	return m, tea.Batch(cmds...)
}

func (m *MainModel) propagateChangesetToDeployModel(msg stageui.StageCompleteMsg) {
	deployModel, ok := m.deploy.(DeployModel)
	if !ok {
		return
	}
	deployModel.changesetID = msg.ChangesetID
	deployModel.footerRenderer.ChangesetID = msg.ChangesetID
	deployModel.blueprintFile = m.blueprintFile
	deployModel.blueprintSource = m.blueprintSource
	deployModel.SetPreDeployInstanceState(msg.InstanceState)
	deployModel.SetChangesetChanges(msg.Changes)
	m.deploy = deployModel
}

func (m *MainModel) resolveApprovalAction(msg stageui.StageCompleteMsg) []tea.Cmd {
	if m.autoApprove || m.headless {
		m.sessionState = deployExecute
		return []tea.Cmd{m.triggerDeploymentWithChangeset(msg.ChangesetID, msg.Changes)}
	}

	if m.autoApproveCodeOnly {
		return m.handleCodeOnlyApproval(msg)
	}

	m.showStagingConfirmationFooter(msg)
	return nil
}

func (m *MainModel) handleCodeOnlyApproval(msg stageui.StageCompleteMsg) []tea.Cmd {
	result := shared.CheckCodeOnlyEligibility(msg.Changes, msg.InstanceState)
	if result.Eligible {
		m.sessionState = deployExecute
		return []tea.Cmd{m.triggerDeploymentWithChangeset(msg.ChangesetID, msg.Changes)}
	}

	m.showStagingConfirmationFooter(msg, WithCodeOnlyDenial(result.Reasons))
	return nil
}

func (m *MainModel) showStagingConfirmationFooter(
	msg stageui.StageCompleteMsg,
	opts ...StagingFooterOption,
) {
	create, update, del, recreate := m.staging.CountChangeSummary()
	footer := &DeployStagingFooterRenderer{
		ChangesetID:      msg.ChangesetID,
		Summary:          ChangeSummary{Create: create, Update: update, Delete: del, Recreate: recreate},
		HasExportChanges: stageui.HasAnyExportChanges(msg.Changes),
	}
	for _, opt := range opts {
		opt(footer)
	}
	m.staging.SetFooterRenderer(footer)
}

// triggerDeploymentWithChangeset starts deployment with the given changeset.
func (m *MainModel) triggerDeploymentWithChangeset(_ string, _ *changes.BlueprintChanges) tea.Cmd {
	return func() tea.Msg {
		return StartDeployMsg{}
	}
}

// determineNextStateAfterBlueprintSelect determines the next session state
// after a blueprint has been selected.
func (m *MainModel) determineNextStateAfterBlueprintSelect() deploySessionState {
	// Check if all required values are provided for skipping prompts
	instanceIdentified := m.instanceID != "" || m.instanceName != ""
	hasDeployPath := m.stageFirst || m.changesetID != ""
	canSkipForm := m.skipPrompts && instanceIdentified && hasDeployPath

	if m.headless || canSkipForm {
		// Skip config form - go straight to staging or deploy
		if m.stageFirst {
			return deployStaging
		}
		return deployExecute
	}

	// In interactive mode, show the config form so users can review settings
	return deployConfigInput
}

func (m MainModel) updatePreflight(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case preflight.SatisfiedMsg:
		m.sessionState = m.postPreflightState
		cmds := []tea.Cmd{
			m.selectBlueprint.Init(),
			m.deployConfigForm.Init(),
			m.deploy.Init(),
		}
		if m.staging != nil {
			cmds = append(cmds, m.staging.Init())
		}
		if m.postPreflightState == deployExecute {
			cmds = append(cmds, func() tea.Msg { return StartDeployMsg{} })
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

// View renders the main model.
func (m MainModel) View() string {
	if m.quitting {
		if m.restartInstructions != "" {
			return preflight.RenderInstallSummary(
				m.styles, m.installedPlugins, len(m.installedPlugins),
				m.restartInstructions, m.preflightCommandName,
			)
		}
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("See you next time.")
	}

	switch m.sessionState {
	case deployPreflight:
		if m.preflight != nil {
			return m.preflight.View()
		}
		return ""
	case deployBlueprintSelect:
		return m.selectBlueprint.View()
	case deployConfigInput:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		return selected + m.deployConfigForm.View()
	case deployStaging:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		if m.instanceName != "" {
			selected += "  Instance: " + m.styles.Selected.Render(m.instanceName) + "\n"
		}
		if m.staging != nil {
			return selected + m.staging.View()
		}
		return selected
	default:
		// In deploy view, always show the split-pane (even during streaming)
		return m.deploy.View()
	}
}

// DeployAppConfig holds the configuration for creating a new deploy application.
type DeployAppConfig struct {
	DeployEngine           engine.DeployEngine
	Logger                 *zap.Logger
	ChangesetID            string
	InstanceID             string
	InstanceName           string
	BlueprintFile          string
	IsDefaultBlueprintFile bool
	AutoRollback           bool
	Force                  bool
	StageFirst             bool
	AutoApprove            bool
	AutoApproveCodeOnly    bool
	SkipPrompts            bool
	Styles                 *stylespkg.Styles
	Headless               bool
	HeadlessWriter         io.Writer
	JSONMode               bool
	Preflight              tea.Model
}

// NewDeployApp creates a new deploy application with the given configuration.
func NewDeployApp(cfg DeployAppConfig) (*MainModel, error) {
	sessionState := deployBlueprintSelect
	// In headless mode or with --skip-prompts, use the default blueprint file if no explicit file is provided.
	// Also auto-start if blueprint file is explicitly provided.
	autoDeploy := (cfg.BlueprintFile != "" && !cfg.IsDefaultBlueprintFile) || cfg.Headless || cfg.SkipPrompts

	if autoDeploy {
		// Check if all required values are provided for skipping prompts
		instanceIdentified := cfg.InstanceID != "" || cfg.InstanceName != ""
		hasDeployPath := cfg.StageFirst || cfg.ChangesetID != ""
		canSkipForm := cfg.SkipPrompts && instanceIdentified && hasDeployPath

		// Flag validation for headless mode is now done at the command level
		// using headless.Validate() in deploy.go

		if cfg.Headless || canSkipForm {
			// Skip config form - go straight to staging or deploy
			if cfg.StageFirst {
				sessionState = deployStaging
			} else {
				sessionState = deployExecute
			}
		} else {
			// In interactive mode, show the config form so users can review settings
			sessionState = deployConfigInput
		}
	}

	fp, err := sharedui.BlueprintLocalFilePicker(cfg.Styles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		cfg.BlueprintFile,
		autoDeploy,
		"deploy",
		cfg.Styles,
		&fp,
	)
	if err != nil {
		return nil, err
	}

	deployConfigForm := NewDeployConfigFormModel(
		DeployConfigFormInitialValues{
			InstanceName: cfg.InstanceName,
			InstanceID:   cfg.InstanceID,
			ChangesetID:  cfg.ChangesetID,
			StageFirst:   cfg.StageFirst,
			AutoApprove:  cfg.AutoApprove,
			AutoRollback: cfg.AutoRollback,
		},
		cfg.Styles,
	)

	// Create staging model for --stage flow (reusing stageui.StageModel)
	stagingModel := stageui.NewStageModel(stageui.StageModelConfig{
		DeployEngine:   cfg.DeployEngine,
		Logger:         cfg.Logger,
		InstanceID:     cfg.InstanceID,
		InstanceName:   cfg.InstanceName,
		Destroy:        false,     // not applicable for deploy staging
		SkipDriftCheck: cfg.Force, // use force flag to skip drift detection during staging
		Styles:         cfg.Styles,
		IsHeadless:     cfg.Headless,
		HeadlessWriter: cfg.HeadlessWriter,
		JSONMode:       cfg.JSONMode,
	})
	staging := &stagingModel
	// Pre-populate blueprint info if available
	staging.SetBlueprintFile(cfg.BlueprintFile)
	// Mark as deploy flow mode so staging doesn't print apply hint or quit
	staging.SetDeployFlowMode(true)

	blueprintSource := shared.BlueprintSourceFromPath(cfg.BlueprintFile)
	deploy := NewDeployModel(DeployModelConfig{
		DeployEngine:     cfg.DeployEngine,
		Logger:           cfg.Logger,
		ChangesetID:      cfg.ChangesetID,
		InstanceID:       cfg.InstanceID,
		InstanceName:     cfg.InstanceName,
		BlueprintFile:    cfg.BlueprintFile,
		BlueprintSource:  blueprintSource,
		AutoRollback:     cfg.AutoRollback,
		Force:            cfg.Force,
		Styles:           cfg.Styles,
		IsHeadless:       cfg.Headless,
		HeadlessWriter:   cfg.HeadlessWriter,
		ChangesetChanges: nil, // will be set when staging completes
		JSONMode:         cfg.JSONMode,
	})

	postPreflightState := sessionState
	if cfg.Preflight != nil {
		sessionState = deployPreflight
	}

	return &MainModel{
		sessionState:        sessionState,
		selectBlueprint:     selectBlueprint,
		deployConfigForm:    deployConfigForm,
		staging:             staging,
		deploy:              deploy,
		preflight:           cfg.Preflight,
		postPreflightState:  postPreflightState,
		changesetID:         cfg.ChangesetID,
		instanceID:          cfg.InstanceID,
		instanceName:        cfg.InstanceName,
		blueprintFile:       cfg.BlueprintFile,
		blueprintSource:     blueprintSource,
		isDefaultBlueprint:  cfg.IsDefaultBlueprintFile,
		autoRollback:        cfg.AutoRollback,
		force:               cfg.Force,
		stageFirst:          cfg.StageFirst,
		autoApprove:         cfg.AutoApprove,
		autoApproveCodeOnly: cfg.AutoApproveCodeOnly,
		skipPrompts:         cfg.SkipPrompts,
		headless:            cfg.Headless,
		jsonMode:            cfg.JSONMode,
		engine:              cfg.DeployEngine,
		logger:              cfg.Logger,
		styles:              cfg.Styles,
	}, nil
}

// Test accessor methods - these provide read-only access for testing purposes.

// StageFirst returns whether staging should happen before deployment.
func (m *MainModel) StageFirst() bool {
	return m.stageFirst
}

// ChangesetID returns the changeset ID.
func (m *MainModel) ChangesetID() string {
	return m.changesetID
}

// InstanceName returns the instance name.
func (m *MainModel) InstanceName() string {
	return m.instanceName
}
