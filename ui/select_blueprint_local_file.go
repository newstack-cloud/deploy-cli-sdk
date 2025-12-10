package ui

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type SelectBlueprintLocalFileModel struct {
	filepicker   filepicker.Model
	styles       stylespkg.Styles
	selectedFile string
	err          error
}

func (m SelectBlueprintLocalFileModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m SelectBlueprintLocalFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var fpcmd tea.Cmd
	m.filepicker, fpcmd = m.filepicker.Update(msg)
	cmds = append(cmds, fpcmd)

	// Did the user select a file?
	if didSelect, file := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = file
		// Dispatch comamand with the path of the selected file.
		cmds = append(cmds, selectBlueprintCmd(file, consts.BlueprintSourceFile))
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(path + " is not a valid blueprint file.")
		errCmd := selectBlueprintFileErrorCmd(m.err)
		m.selectedFile = ""
		return m, tea.Batch(fpcmd, errCmd, clearErrorAfter(2*time.Second), clearSelectedBlueprintCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m SelectBlueprintLocalFileModel) View() string {
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
		s.WriteString("\n")
	} else if m.selectedFile == "" {
		s.WriteString("Pick a blueprint file:")
		s.WriteString("\n\n")
	} else {
		s.WriteString("Selected blueprint file: " + m.styles.Selected.Render(m.selectedFile))
		s.WriteString("\n\n")
	}

	s.WriteString(m.filepicker.View())

	return s.String()
}

func NewSelectBlueprintLocalFile(
	fp *filepicker.Model,
	styles *stylespkg.Styles,
) (*SelectBlueprintLocalFileModel, error) {
	return &SelectBlueprintLocalFileModel{
		filepicker: *fp,
		styles:     *styles,
	}, nil
}

func customFilePickerStyles(styles *stylespkg.Styles) filepicker.Styles {
	fpStyles := filepicker.DefaultStyles()
	fpStyles.Selected = styles.Selected
	fpStyles.File = styles.Selectable
	fpStyles.Directory = styles.Selectable
	fpStyles.Cursor = styles.Selected
	return fpStyles
}

// BlueprintLocalFilePicker creates a new filepicker model for selecting a local blueprint file
// relative to the current working directory.
func BlueprintLocalFilePicker(styles *stylespkg.Styles) (filepicker.Model, error) {
	fp := filepicker.New()
	fp.Styles = customFilePickerStyles(styles)
	fp.AllowedTypes = []string{".yaml", ".yml", ".json", ".jsonc"}

	currentDir, err := os.Getwd()
	if err != nil {
		return filepicker.Model{}, err
	}
	fp.CurrentDirectory = currentDir

	return fp, nil
}
