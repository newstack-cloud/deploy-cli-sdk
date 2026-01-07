package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// SelectBlueprintModel wraps SelectFileModel with blueprint-specific configuration.
type SelectBlueprintModel struct {
	fileSelector *SelectFileModel
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m SelectBlueprintModel) Init() tea.Cmd {
	cmd := m.fileSelector.Init()
	return wrapFileCmdForBlueprint(cmd)
}

func (m SelectBlueprintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Translate blueprint-specific messages to generic file messages
	switch typedMsg := msg.(type) {
	case SelectBlueprintStartMsg:
		msg = SelectFileStartMsg(typedMsg)
	case SelectBlueprintSourceMsg:
		msg = SelectFileSourceMsg(typedMsg)
	case SelectBlueprintMsg:
		msg = SelectFileMsg{File: typedMsg.BlueprintFile, Source: typedMsg.Source}
	case ClearSelectedBlueprintMsg:
		msg = ClearSelectedFileMsg{}
	case SelectBlueprintFileErrorMsg:
		msg = SelectFileErrorMsg(typedMsg)
	}

	// Forward to the file selector
	newModel, cmd := m.fileSelector.Update(msg)
	fileModel := newModel.(SelectFileModel)
	m.fileSelector = &fileModel

	// Wrap the command to translate file messages back to blueprint messages
	wrappedCmd := wrapFileCmdForBlueprint(cmd)

	return m, wrappedCmd
}

// wrapFileCmdForBlueprint wraps a command to translate SelectFileMsg to SelectBlueprintMsg.
func wrapFileCmdForBlueprint(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		return translateFileMsg(msg)
	}
}

// translateFileMsg converts file messages to blueprint messages.
func translateFileMsg(msg tea.Msg) tea.Msg {
	switch m := msg.(type) {
	case tea.BatchMsg:
		translated := make([]tea.Cmd, len(m))
		for i, cmd := range m {
			translated[i] = wrapFileCmdForBlueprint(cmd)
		}
		return tea.BatchMsg(translated)
	case SelectFileMsg:
		return SelectBlueprintMsg{BlueprintFile: m.File, Source: m.Source}
	case SelectFileSourceMsg:
		return SelectBlueprintSourceMsg(m)
	case SelectFileStartMsg:
		return SelectBlueprintStartMsg(m)
	case ClearSelectedFileMsg:
		return ClearSelectedBlueprintMsg{}
	case SelectFileErrorMsg:
		return SelectBlueprintFileErrorMsg(m)
	}
	return msg
}

func (m SelectBlueprintModel) View() string {
	return m.fileSelector.View()
}

// SelectedFile returns the currently selected blueprint file path.
func (m SelectBlueprintModel) SelectedFile() string {
	return m.fileSelector.SelectedFile()
}

// SelectedSource returns the source type of the selected blueprint file.
func (m SelectBlueprintModel) SelectedSource() string {
	return m.fileSelector.SelectedSource()
}

// NewSelectBlueprint creates a new blueprint selection model.
func NewSelectBlueprint(
	blueprintFile string,
	autoSelect bool,
	action string,
	styles *stylespkg.Styles,
	fp *filepicker.Model,
) (*SelectBlueprintModel, error) {
	fileSelector, err := NewSelectFile(
		SelectFileConfig{
			InitialFile:  blueprintFile,
			AutoSelect:   autoSelect,
			Action:       action,
			Styles:       styles,
			FilePicker:   fp,
			FileTypeName: "blueprint file",
			RemoteFileConfig: &SelectRemoteFileConfig{
				URLTitle:       "HTTPS resource URL blueprint file location",
				URLDescription: "The public URL of the blueprint to download.",
				URLPlaceholder: "https://assets.example.com/project.blueprint.yml",
				BucketNameDescription: map[string]string{
					consts.FileSourceS3:        "The name of the S3 bucket containing the blueprint file.",
					consts.FileSourceGCS:       "The name of the GCS bucket containing the blueprint file.",
					consts.FileSourceAzureBlob: "The name of the Azure Blob Storage container containing the blueprint file.",
				},
				ObjectPathDescription: map[string]string{
					consts.FileSourceS3:        "The path of the blueprint file in the S3 bucket.",
					consts.FileSourceGCS:       "The path of the blueprint file in the GCS bucket.",
					consts.FileSourceAzureBlob: "The path of the blueprint file in the Azure Blob Storage container.",
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &SelectBlueprintModel{
		fileSelector: fileSelector,
	}, nil
}

// ToFullBlueprintPath returns the full path to the blueprint file based on the source.
func ToFullBlueprintPath(blueprintFile, source string) string {
	return ToFullFilePath(blueprintFile, source)
}
