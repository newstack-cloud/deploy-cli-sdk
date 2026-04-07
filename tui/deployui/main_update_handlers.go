package deployui

import (
	"errors"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/newstack-cloud/deploy-cli-sdk/tui/driftui"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stageui"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

func (m MainModel) handleSelectBlueprintMsg(msg sharedui.SelectBlueprintMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
	m.blueprintSource = msg.Source

	// If we're already in deployExecute state, just pass the message to the deploy model
	// without recalculating state. This prevents switching back to staging when
	// SelectBlueprintMsg is received during deployment.
	if m.sessionState == deployExecute {
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Determine next state based on whether we need instance name
	nextState := m.determineNextStateAfterBlueprintSelect()
	m.sessionState = nextState

	switch nextState {
	case deployStaging:
		// Resolve instance identifiers before starting staging
		// This handles the case where instance name is provided but instance doesn't exist yet
		m.staging.SetBlueprintFile(m.blueprintFile)
		m.staging.SetBlueprintSource(m.blueprintSource)
		cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
	case deployExecute:
		// Pass the message to the deploy model to start deployment
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleDeployConfigMsg(msg DeployConfigMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	// Update config from form
	m.instanceName = msg.InstanceName
	m.instanceID = msg.InstanceID
	m.changesetID = msg.ChangesetID
	m.stageFirst = msg.StageFirst
	m.autoApprove = msg.AutoApprove
	m.autoRollback = msg.AutoRollback

	// Update deploy model with the new values
	deployModel, ok := m.deploy.(DeployModel)
	if ok {
		deployModel.instanceName = msg.InstanceName
		deployModel.instanceID = msg.InstanceID
		deployModel.changesetID = msg.ChangesetID
		deployModel.autoRollback = msg.AutoRollback
		deployModel.footerRenderer.InstanceName = msg.InstanceName
		deployModel.footerRenderer.InstanceID = msg.InstanceID
		deployModel.footerRenderer.ChangesetID = msg.ChangesetID
		m.deploy = deployModel
	}

	// Update staging model
	if m.staging != nil {
		m.staging.SetInstanceName(msg.InstanceName)
		m.staging.SetInstanceID(msg.InstanceID)
	}

	// Determine next state - staging if stageFirst is set, otherwise deploy
	if msg.StageFirst {
		m.sessionState = deployStaging
		m.staging.SetBlueprintFile(m.blueprintFile)
		m.staging.SetBlueprintSource(m.blueprintSource)
		// Start the staging spinner animation
		cmds = append(cmds, m.staging.Init())
		// Resolve instance identifiers before starting staging
		// This handles the case where instance name is provided but instance doesn't exist yet
		cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
	} else {
		m.sessionState = deployExecute
		// Trigger deployment start with the provided changeset ID
		cmds = append(cmds, func() tea.Msg {
			return sharedui.SelectBlueprintMsg{
				BlueprintFile: m.blueprintFile,
				Source:        m.blueprintSource,
			}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleInstanceResolvedMsg(msg InstanceResolvedMsg) (tea.Model, tea.Cmd) {
	// Instance identifiers have been resolved - set them on staging model and start staging
	m.staging.SetInstanceID(msg.InstanceID)
	m.staging.SetInstanceName(msg.InstanceName)
	return m, m.staging.StartStaging()
}

func (m MainModel) handleStageCompleteMsg(msg stageui.StageCompleteMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	// Guard: If we're already in deployExecute, ignore this message.
	// This can happen if a stale StageCompleteMsg arrives after deployment has started.
	if m.sessionState == deployExecute {
		return m, tea.Batch(cmds...)
	}
	return m.handleStageComplete(msg, cmds)
}

func (m MainModel) handleConfirmDeployMsg(msg ConfirmDeployMsg) (tea.Model, tea.Cmd) {
	if msg.Confirmed {
		m.sessionState = deployExecute
		return m, m.triggerDeploymentWithChangeset(m.changesetID, nil)
	}
	// User cancelled
	m.quitting = true
	return m, tea.Quit
}

func (m MainModel) handleStartDeployMsg(msg StartDeployMsg) (tea.Model, tea.Cmd) {
	// Forward StartDeployMsg to the deploy model to initiate deployment
	var cmd tea.Cmd
	m.deploy, cmd = m.deploy.Update(msg)
	return m, cmd
}

func (m MainModel) handleClearSelectedBlueprintMsg() (tea.Model, tea.Cmd) {
	// Guard: Don't reset state if we're in deployStaging or deployExecute.
	// This prevents unexpected state resets during active operations.
	if m.sessionState == deployStaging || m.sessionState == deployExecute {
		return m, nil
	}
	m.sessionState = deployBlueprintSelect
	m.blueprintFile = ""
	m.blueprintSource = ""
	return m, nil
}

func (m MainModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var bpCmd tea.Cmd
	m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
	var configFormCmd tea.Cmd
	var configFormModel tea.Model
	configFormModel, configFormCmd = m.deployConfigForm.Update(msg)
	if cfm, ok := configFormModel.(DeployConfigFormModel); ok {
		m.deployConfigForm = &cfm
	}
	var deployCmd tea.Cmd
	m.deploy, deployCmd = m.deploy.Update(msg)
	cmds = append(cmds, bpCmd, configFormCmd, deployCmd)
	if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleSpinnerTickMsg(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var cmd tea.Cmd
	m.deploy, cmd = m.deploy.Update(msg)
	cmds = append(cmds, cmd)
	if m.staging != nil && m.sessionState == deployStaging {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleStagingEventMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	// Route staging-specific messages to staging model.
	if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
		if m.staging.GetError() != nil {
			m.Error = m.staging.GetError()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleDriftMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	// Route drift messages based on current session state.
	// During deployment (deployExecute), drift is handled by the deploy model.
	// During staging (deployStaging), drift is handled by the staging model.
	if m.sessionState == deployExecute {
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
		if m.staging.GetError() != nil {
			m.Error = m.staging.GetError()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleSessionStateRouting(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch m.sessionState {
	case deployBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in deploy")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)

	case deployStaging:
		// Route non-staging-specific messages (like KeyMsg, MouseMsg) to staging model.
		// Staging-specific messages, drift messages, and spinner ticks are already handled
		// in the type switch above, so skip them here to avoid duplicate processing.
		switch msg.(type) {
		case stageui.StageStartedMsg, stageui.StageEventMsg, stageui.StageErrorMsg, stageui.StageCompleteMsg,
			driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg,
			spinner.TickMsg:
			// Already handled above
		default:
			if m.staging != nil {
				var stagingModel tea.Model
				var stagingCmd tea.Cmd
				stagingModel, stagingCmd = m.staging.Update(msg)
				if sm, ok := stagingModel.(stageui.StageModel); ok {
					m.staging = &sm
				}
				cmds = append(cmds, stagingCmd)
			}
		}

	case deployConfigInput:
		var configFormModel tea.Model
		var configFormCmd tea.Cmd
		configFormModel, configFormCmd = m.deployConfigForm.Update(msg)
		if cfm, ok := configFormModel.(DeployConfigFormModel); ok {
			m.deployConfigForm = &cfm
		}
		cmds = append(cmds, configFormCmd)

	case deployExecute:
		// Skip drift messages since they're already handled in the type switch above.
		switch msg.(type) {
		case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
			// Already handled above
		default:
			newDeploy, newCmd := m.deploy.Update(msg)
			deployModel, ok := newDeploy.(DeployModel)
			if !ok {
				panic("failed to perform assertion on deploy model")
			}
			m.deploy = deployModel
			cmds = append(cmds, newCmd)
			if deployModel.err != nil {
				m.Error = deployModel.err
			} else if deployModel.finished && IsFailedStatus(deployModel.finalStatus) {
				// Deployment completed with a failed status - set error for non-zero exit code
				m.Error = errors.New("deployment failed with status: " + deployModel.finalStatus.String())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

