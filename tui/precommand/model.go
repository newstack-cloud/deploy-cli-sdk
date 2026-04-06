// Package precommand provides a bubbletea sub-model that runs a
// PreCommandStep with a spinner and progress display, integrating
// into the deploy/stage TUI as an initial phase.
package precommand

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// CompleteMsg indicates the pre-command step finished successfully.
type CompleteMsg struct{}

// ErrorMsg indicates the pre-command step failed.
type ErrorMsg struct {
	Err error
}

// ProgressUpdateMsg carries a progress update from the step to the TUI.
type ProgressUpdateMsg struct {
	Phase  string
	Detail string
}

type stepDoneMsg struct {
	err error
}

// Model is a bubbletea sub-model that runs a PreCommandStep and
// displays progress with a spinner.
type Model struct {
	step         Step
	confProvider *config.Provider
	commandName  string
	styles       *stylespkg.Styles
	headless     bool
	writer       io.Writer

	spinner      spinner.Model
	currentPhase string
	phaseDetail  string
	progressCh   <-chan ProgressMsg
	done         bool
	err          error

	// Err is set if the pre-command step failed. Check after the program exits.
	Err error
}

// Options for creating a new pre-command step model.
type Options struct {
	Step         Step
	ConfProvider *config.Provider
	CommandName  string
	Styles       *stylespkg.Styles
	Headless     bool
	Writer       io.Writer
}

func NewModel(opts Options) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	if opts.Styles != nil {
		s.Style = opts.Styles.Selected
	}

	return &Model{
		step:         opts.Step,
		confProvider: opts.ConfProvider,
		commandName:  opts.CommandName,
		styles:       opts.Styles,
		headless:     opts.Headless,
		writer:       opts.Writer,
		spinner:      s,
		currentPhase: "Preparing...",
	}
}

func (m Model) Init() tea.Cmd {
	if m.headless && m.writer != nil {
		fmt.Fprintf(m.writer, "Running pre-command step...\n")
	}

	progressCh := make(chan ProgressMsg, 16)
	m.progressCh = progressCh

	return tea.Batch(
		m.spinner.Tick,
		m.startStep(progressCh),
		m.waitForProgress(progressCh),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressUpdateMsg:
		m.currentPhase = msg.Phase
		m.phaseDetail = msg.Detail
		if m.headless && m.writer != nil {
			if msg.Detail != "" {
				fmt.Fprintf(m.writer, "  %s: %s\n", msg.Phase, msg.Detail)
			} else {
				fmt.Fprintf(m.writer, "  %s\n", msg.Phase)
			}
		}
		// Wait for the next progress message.
		if m.progressCh != nil {
			return m, m.waitForProgress(m.progressCh)
		}
		return m, nil

	case stepDoneMsg:
		m.done = true
		m.progressCh = nil
		if msg.err != nil {
			m.err = msg.err
			m.Err = msg.err
			if m.headless && m.writer != nil {
				fmt.Fprintf(m.writer, "Pre-command step failed: %v\n", msg.err)
			}
			return m, tea.Quit
		}
		if m.headless && m.writer != nil {
			fmt.Fprintf(m.writer, "Pre-command step complete.\n")
		}
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.headless || m.done {
		return ""
	}

	phase := m.currentPhase
	if m.phaseDetail != "" {
		phase = fmt.Sprintf("%s: %s", m.currentPhase, m.phaseDetail)
	}

	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), phase)
}

func (m Model) startStep(progressCh chan ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		err := m.step.Run(
			context.TODO(),
			m.confProvider,
			m.commandName,
			progressCh,
		)
		close(progressCh)
		return stepDoneMsg{err: err}
	}
}

func (m Model) waitForProgress(ch <-chan ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return ProgressUpdateMsg(msg)
	}
}
