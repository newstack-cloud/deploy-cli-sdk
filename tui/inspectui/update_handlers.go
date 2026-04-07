package inspectui

import (
	"errors"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m MainModel) handleInstanceInputMsg(msg InstanceInputMsg) (tea.Model, tea.Cmd) {
	m.instanceID = msg.InstanceID
	m.instanceName = msg.InstanceName

	// Update inspect model with the identifiers
	m.inspect.instanceID = msg.InstanceID
	m.inspect.instanceName = msg.InstanceName
	m.inspect.footerRenderer.InstanceID = msg.InstanceID
	m.inspect.footerRenderer.InstanceName = msg.InstanceName

	// Transition to loading state and start fetching
	m.sessionState = inspectLoading
	return m, fetchInstanceStateCmd(*m.inspect)
}

func (m MainModel) handleInstanceStateFetchedMsg(msg InstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	m.sessionState = inspectViewing

	// Update the inspect model with fetched state
	m.inspect.SetInstanceState(msg.InstanceState)

	if msg.IsInProgress {
		// Start streaming events and periodic state refresh
		m.inspect.streaming = true
		m.inspect.footerRenderer.Streaming = true
		cmds = append(cmds, startStreamingCmd(*m.inspect))
		cmds = append(cmds, startStateRefreshTickerCmd())
	} else {
		// Static view - just display the state
		m.inspect.finished = true
		m.inspect.footerRenderer.Finished = true
		m.inspect.detailsRenderer.Finished = true

		// In headless mode, output now and quit
		if m.headless {
			if m.jsonMode {
				m.inspect.outputJSON()
			} else {
				m.inspect.printHeadlessInstanceState()
			}
			return m, tea.Quit
		}
	}
	// Return early to avoid double-handling by InspectModel
	return m, tea.Batch(cmds...)
}

func (m MainModel) handleInstanceNotFoundMsg(msg InstanceNotFoundMsg) (tea.Model, tea.Cmd) {
	m.Error = msg.Err
	if m.headless {
		if m.jsonMode {
			m.inspect.outputJSONError(msg.Err)
		} else {
			m.inspect.printHeadlessError(msg.Err)
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m MainModel) handleInspectStreamStartedMsg() (tea.Model, tea.Cmd) {
	return m, waitForNextEventCmd(*m.inspect)
}

func (m MainModel) handleStateRefreshTickMsg() (tea.Model, tea.Cmd) {
	// Only refresh if still streaming
	if m.inspect.streaming && !m.inspect.finished {
		return m, tea.Batch(
			refreshInstanceStateCmd(*m.inspect),
			startStateRefreshTickerCmd(),
		)
	}
	return m, nil
}

func (m MainModel) handleInstanceStateRefreshedMsg(msg InstanceStateRefreshedMsg) (tea.Model, tea.Cmd) {
	// Hydrate existing items with updated state
	// This runs both during streaming (periodic refresh) and after streaming ends (final refresh)
	if msg.InstanceState != nil {
		m.inspect.RefreshInstanceState(msg.InstanceState)
	}
	return m, nil
}

func (m MainModel) handleInspectEventMsg(msg InspectEventMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	m.inspect.processEvent(&msg)
	m.inspect.splitPane.UpdateItems(ToSplitPaneItems(m.inspect.items))

	finishData, isFinish := msg.AsFinish()
	if isFinish && finishData.EndOfStream {
		m.inspect.finished = true
		m.inspect.streaming = false
		m.inspect.footerRenderer.Streaming = false
		m.inspect.footerRenderer.Finished = true
		m.inspect.footerRenderer.CurrentStatus = finishData.Status
		m.inspect.detailsRenderer.Finished = true

		if m.headless {
			if m.jsonMode {
				m.inspect.outputJSON()
			} else {
				m.inspect.printHeadlessInstanceState()
			}
			return m, tea.Quit
		}

		// Trigger a final state refresh to hydrate all items with ResourceState
		// for resources that completed during streaming
		cmds = append(cmds, refreshInstanceStateCmd(*m.inspect))
		return m, tea.Batch(cmds...)
	}

	cmds = append(cmds, waitForNextEventCmd(*m.inspect))
	return m, tea.Batch(cmds...)
}

func (m MainModel) handleInspectStreamClosedMsg() (tea.Model, tea.Cmd) {
	m.inspect.streaming = false
	m.inspect.footerRenderer.Streaming = false
	if !m.inspect.finished {
		m.inspect.err = errors.New("event stream closed unexpectedly")
		m.Error = m.inspect.err
		if m.headless {
			if m.jsonMode {
				m.inspect.outputJSONError(m.inspect.err)
			} else {
				m.inspect.printHeadlessError(m.inspect.err)
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MainModel) handleInspectErrorMsg(msg InspectErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.inspect.err = msg.Err
		m.Error = msg.Err
		if m.headless {
			if m.jsonMode {
				m.inspect.outputJSONError(msg.Err)
			} else {
				m.inspect.printHeadlessError(msg.Err)
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MainModel) handleSpinnerTickMsg(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	// Always forward spinner ticks to the inspect model regardless of session state
	if m.inspect != nil {
		var cmd tea.Cmd
		m.inspect.spinner, cmd = m.inspect.spinner.Update(msg)
		m.inspect.footerRenderer.SpinnerView = m.inspect.spinner.View()
		return m, cmd
	}
	return m, nil
}

func (m MainModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	if m.instanceForm != nil {
		var formModel tea.Model
		var formCmd tea.Cmd
		formModel, formCmd = m.instanceForm.Update(msg)
		if fm, ok := formModel.(*InstanceInputFormModel); ok {
			m.instanceForm = fm
		}
		cmds = append(cmds, formCmd)
	}

	if m.inspect != nil {
		var inspectCmd tea.Cmd
		var inspectModel tea.Model
		inspectModel, inspectCmd = m.inspect.handleWindowSize(msg)
		if im, ok := inspectModel.(InspectModel); ok {
			m.inspect = &im
		}
		cmds = append(cmds, inspectCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit, true
	case "q":
		if m.sessionState == inspectViewing && m.inspect.finished {
			m.quitting = true
			return m, tea.Quit, true
		}
		if m.Error != nil {
			m.quitting = true
			return m, tea.Quit, true
		}
	}
	return m, nil, false
}

func (m MainModel) handleSessionStateRouting(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	// Route messages to the appropriate sub-model based on session state
	switch m.sessionState {
	case inspectInstanceInput:
		if m.instanceForm != nil {
			var formModel tea.Model
			var formCmd tea.Cmd
			formModel, formCmd = m.instanceForm.Update(msg)
			if fm, ok := formModel.(*InstanceInputFormModel); ok {
				m.instanceForm = fm
			}
			cmds = append(cmds, formCmd)
		}

	case inspectLoading:
		// Just waiting for state fetch

	case inspectViewing:
		if m.inspect != nil {
			var inspectModel tea.Model
			var inspectCmd tea.Cmd
			inspectModel, inspectCmd = m.inspect.Update(msg)
			if im, ok := inspectModel.(InspectModel); ok {
				m.inspect = &im
			}
			cmds = append(cmds, inspectCmd)

			if m.inspect.err != nil {
				m.Error = m.inspect.err
			}
		}
	}

	return m, tea.Batch(cmds...)
}
