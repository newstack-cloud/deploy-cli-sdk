package deployui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg routes keyboard input for the MainModel based on current state.
// Returns a bool indicating whether the key was fully handled (caller should return early).
func (m MainModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit, true
	case "y", "Y":
		return m.handleConfirmKeyPress()
	case "n", "N":
		return m.handleRejectKeyPress()
	case "q":
		return m.handleQuitKeyPress()
	}

	return m, nil, false
}

// handleConfirmKeyPress handles y/Y key press for staging confirmation.
func (m MainModel) handleConfirmKeyPress() (tea.Model, tea.Cmd, bool) {
	// Handle confirmation when staging is finished (deploy flow)
	if m.sessionState == deployStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
		return m, func() tea.Msg {
			return ConfirmDeployMsg{Confirmed: true}
		}, true
	}
	return m, nil, false
}

// handleRejectKeyPress handles n/N key press for staging rejection.
func (m MainModel) handleRejectKeyPress() (tea.Model, tea.Cmd, bool) {
	// Handle rejection when staging is finished (deploy flow)
	if m.sessionState == deployStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
		return m, func() tea.Msg {
			return ConfirmDeployMsg{Confirmed: false}
		}, true
	}
	return m, nil, false
}

// handleQuitKeyPress handles q key press based on current session state.
func (m MainModel) handleQuitKeyPress() (tea.Model, tea.Cmd, bool) {
	// Only quit if we're in the deploy view and deployment is finished
	if m.sessionState == deployExecute {
		deployModel, ok := m.deploy.(DeployModel)
		if ok && deployModel.finished {
			m.quitting = true
			return m, tea.Quit, true
		}
	}
	// Allow quit during staging error
	if m.sessionState == deployStaging && m.staging != nil && m.staging.GetError() != nil {
		m.quitting = true
		return m, tea.Quit, true
	}
	return m, nil, false
}
