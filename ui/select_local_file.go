package ui

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// SelectLocalFileModel provides a file picker for selecting local files.
type SelectLocalFileModel struct {
	filepicker   filepicker.Model
	styles       stylespkg.Styles
	selectedFile string
	prompt       string
	fileTypeName string
	err          error
}

func (m SelectLocalFileModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m SelectLocalFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var fpcmd tea.Cmd
	m.filepicker, fpcmd = m.filepicker.Update(msg)
	cmds = append(cmds, fpcmd)

	if didSelect, file := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = file
		cmds = append(cmds, selectFileCmd(file, consts.FileSourceLocal))
	}

	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		fileTypeName := m.fileTypeName
		if fileTypeName == "" {
			fileTypeName = "file"
		}
		m.err = errors.New(path + " is not a valid " + fileTypeName + ".")
		errCmd := selectFileErrorCmd(m.err)
		m.selectedFile = ""
		return m, tea.Batch(fpcmd, errCmd, clearFileErrorAfter(2*time.Second), clearSelectedFileCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m SelectLocalFileModel) View() string {
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
		s.WriteString("\n")
	} else if m.selectedFile == "" {
		s.WriteString(m.prompt)
		s.WriteString("\n\n")
	} else {
		s.WriteString("Selected file: " + m.styles.Selected.Render(m.selectedFile))
		s.WriteString("\n\n")
	}

	s.WriteString(m.filepicker.View())

	return s.String()
}

// NewSelectLocalFile creates a new local file selection model.
func NewSelectLocalFile(
	fp *filepicker.Model,
	styles *stylespkg.Styles,
	prompt string,
	fileTypeName string,
) (*SelectLocalFileModel, error) {
	if prompt == "" {
		prompt = "Pick a file:"
	}
	return &SelectLocalFileModel{
		filepicker:   *fp,
		styles:       *styles,
		prompt:       prompt,
		fileTypeName: fileTypeName,
	}, nil
}
