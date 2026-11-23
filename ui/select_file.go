package ui

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// SelectFileModel provides a generalized file selection interface that supports
// both local files and remote storage (S3, GCS, Azure Blob, HTTPS).
type SelectFileModel struct {
	start           tea.Model
	source          tea.Model
	selectLocalFile tea.Model
	inputRemoteFile tea.Model
	selectedSource  string
	selectedFile    string
	autoSelect      bool
	stage           selectFileStage
	quitting        bool
	err             error
	config          SelectFileConfig
	width           int
}

// SelectFileConfig holds configuration for the file selector.
type SelectFileConfig struct {
	// InitialFile is the default file path or URL to pre-select.
	InitialFile string
	// AutoSelect automatically selects the initial file without user interaction.
	AutoSelect bool
	// Action is used in prompts (e.g., "import", "validate", "deploy").
	Action string
	// Styles provides theming for the UI components.
	Styles *stylespkg.Styles
	// FilePicker is the configured file picker for local file selection.
	FilePicker *filepicker.Model
	// FileTypeName is an optional descriptor for the file type (e.g., "blueprint", "state archive").
	// When set, it's interpolated into prompts like "Pick a blueprint file:".
	// If empty, generic "file" terminology is used.
	FileTypeName string
	// RemoteFileConfig provides configuration for remote file input.
	RemoteFileConfig *SelectRemoteFileConfig
	// SupportedSources specifies which file sources are available for selection.
	// If nil or empty, all sources are supported (Local, S3, GCS, Azure Blob, HTTPS).
	SupportedSources []string
	// Mode is SelectFileModeOpen (the default) to choose an existing file, or
	// SelectFileModeSave to choose where a new file will be written.
	Mode SelectFileMode
	// DefaultFileName pre-fills the local file path in SelectFileModeSave.
	DefaultFileName string
}

// SelectFileMode determines whether the selected file is read or written.
type SelectFileMode int

const (
	// SelectFileModeOpen chooses an existing file to read, with a file picker for local files.
	SelectFileModeOpen SelectFileMode = iota
	// SelectFileModeSave chooses where a file will be written, with a path input for local
	// files so the file does not need to exist yet.
	SelectFileModeSave
)

type selectFileStage int

const (
	selectFileStageStart selectFileStage = iota
	selectFileStageSelectSource
	selectFileStageInputFileLocation
	selectFileStageSelectLocalFile
	selectFileStageSelected
)

// SelectFileMsg is sent when a file has been selected.
type SelectFileMsg struct {
	File   string
	Source string
}

// SelectFileSourceMsg is sent when a source has been selected.
type SelectFileSourceMsg struct {
	Source string
}

// ClearSelectedFileMsg is sent to clear the current selection.
type ClearSelectedFileMsg struct{}

// SelectFileErrorMsg is sent when there's an error during file selection.
type SelectFileErrorMsg struct {
	Err error
}

// SelectFileStartMsg is sent when the user makes a selection at the start screen.
type SelectFileStartMsg struct {
	Selection string
}

func (m SelectFileModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	// The save mode path input is started when its screen is shown instead.
	if m.config.Mode != SelectFileModeSave {
		cmds = append(cmds, m.selectLocalFile.Init())
	}
	cmds = append(cmds, m.inputRemoteFile.Init())

	if m.autoSelect {
		cmds = append(cmds, selectFileCmd(m.selectedFile, m.selectedSource))
	}

	return tea.Batch(cmds...)
}

func (m SelectFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.stage == selectFileStageSelectLocalFile {
			var selectLocalFileCmd tea.Cmd
			m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
			cmds = append(cmds, selectLocalFileCmd)
		}
	case SelectFileStartMsg:
		m.stage = fileStageFromStartSelection(msg.Selection)
		if m.stage == selectFileStageSelected {
			cmds = append(cmds, selectFileCmd(m.selectedFile, m.selectedSource))
		}
	case SelectFileSourceMsg:
		var sourceCmd tea.Cmd
		m, sourceCmd = m.handleSourceSelected(msg)
		cmds = append(cmds, sourceCmd)
	case SelectFileMsg:
		m.selectedFile = msg.File
		m.selectedSource = msg.Source
		m.stage = selectFileStageSelected
	case clearErrorMsg:
		m.err = nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		var startCmd tea.Cmd
		m.start, startCmd = m.start.Update(msg)
		cmds = append(cmds, startCmd)

		var sourceCmd tea.Cmd
		m.source, sourceCmd = m.source.Update(msg)
		cmds = append(cmds, sourceCmd)

		var selectLocalFileCmd tea.Cmd
		m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
		cmds = append(cmds, selectLocalFileCmd)
	default:
		if m.localFileTakesAllMessages() {
			var selectLocalFileCmd tea.Cmd
			m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
			cmds = append(cmds, selectLocalFileCmd)
		}
	}

	switch m.stage {
	case selectFileStageStart:
		var startCmd tea.Cmd
		m.start, startCmd = m.start.Update(msg)
		cmds = append(cmds, startCmd)
	case selectFileStageSelectSource:
		var sourceCmd tea.Cmd
		m.source, sourceCmd = m.source.Update(msg)
		cmds = append(cmds, sourceCmd)
	case selectFileStageInputFileLocation:
		var inputRemoteFileCmd tea.Cmd
		m.inputRemoteFile, inputRemoteFileCmd = m.inputRemoteFile.Update(msg)
		cmds = append(cmds, inputRemoteFileCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SelectFileModel) handleSourceSelected(msg SelectFileSourceMsg) (SelectFileModel, tea.Cmd) {
	m.stage = fileStageFromSource(msg.Source)
	m.selectedSource = msg.Source
	cmds := []tea.Cmd{}
	if m.config.Mode == SelectFileModeSave && m.stage == selectFileStageSelectLocalFile {
		cmds = append(cmds, m.selectLocalFile.Init())
	}
	var inputRemoteFileCmd tea.Cmd
	m.inputRemoteFile, inputRemoteFileCmd = m.inputRemoteFile.Update(msg)
	cmds = append(cmds, inputRemoteFileCmd)
	return m, tea.Batch(cmds...)
}

// The file picker loads its directory listing in the background, so it takes every
// message. The save mode path input is a huh form, and huh's messages are not tied to a
// form: one sent while the remote file form is in use would submit the path input as well,
// so it only takes messages while its screen is shown.
func (m SelectFileModel) localFileTakesAllMessages() bool {
	return m.config.Mode != SelectFileModeSave || m.stage == selectFileStageSelectLocalFile
}

func (m SelectFileModel) View() string {
	if m.quitting {
		return ""
	}

	fileTypeName := m.config.FileTypeName
	if fileTypeName == "" {
		fileTypeName = "file"
	}

	var s strings.Builder

	switch m.stage {
	case selectFileStageStart:
		s.WriteString("\n\n")
		// Only show default file message if there's actually an initial file configured
		if m.config.InitialFile != "" {
			prefix := fmt.Sprintf("    The default %s is: ", fileTypeName)
			// Leave room for trailing space + newlines
			maxPathLen := SafeWidth(m.width) - len(prefix) - 1
			displayPath := TruncatePath(m.selectedFile, maxPathLen)
			s.WriteString(fmt.Sprintf("%s%s \n\n", prefix, displayPath))
		}
		s.WriteString(m.start.View() + "\n")
	case selectFileStageSelectSource:
		s.WriteString("\n\n" + m.source.View() + "\n")
	case selectFileStageSelectLocalFile:
		s.WriteString("\n\n" + m.selectLocalFile.View() + "\n")
	case selectFileStageInputFileLocation:
		s.WriteString("\n\n" + m.inputRemoteFile.View() + "\n")
	case selectFileStageSelected:
		fullPath := ToFullFilePath(m.selectedFile, m.selectedSource)
		capitalizedTypeName := strings.ToUpper(fileTypeName[:1]) + fileTypeName[1:]
		prefix := fmt.Sprintf("%s selected: ", capitalizedTypeName)
		maxPathLen := SafeWidth(m.width) - len(prefix)
		displayPath := TruncatePath(fullPath, maxPathLen)
		s.WriteString(fmt.Sprintf("\n\n%s%s\n", prefix, displayPath))
	}

	return s.String()
}

// SelectedFile returns the currently selected file path.
func (m SelectFileModel) SelectedFile() string {
	return m.selectedFile
}

// SelectedSource returns the source type of the selected file.
func (m SelectFileModel) SelectedSource() string {
	return m.selectedSource
}

// NewSelectFile creates a new file selection model.
func NewSelectFile(config SelectFileConfig) (*SelectFileModel, error) {
	fileTypeName := config.FileTypeName
	if fileTypeName == "" {
		fileTypeName = "file"
	}

	workingDir, err := selectFileWorkingDir(config.FilePicker)
	if err != nil {
		return nil, err
	}

	startPrompt := fmt.Sprintf("Get started by choosing a %s", fileTypeName)
	sourcePrompt := selectFileSourcePrompt(config, fileTypeName)

	selectLocalFile, err := newLocalFileModel(config, fileTypeName, workingDir)
	if err != nil {
		return nil, err
	}

	start := NewSelect(
		startPrompt,
		selectFileStartListItems(config.InitialFile != ""),
		config.Styles,
		selectFileStartCmd,
		false,
	)

	sourceItems := fileSourceListItems(selectFileSources(config))
	source := NewSelect(
		sourcePrompt,
		sourceItems,
		config.Styles,
		selectFileSourceCmd,
		false,
	)

	inputRemoteFile := NewSelectRemoteFile(config.InitialFile, config.Styles, remoteFileConfig(config))

	defaultFile, defaultSource := initialFileAndSource(config.InitialFile)
	selectedFile := defaultFile
	if defaultSource == consts.FileSourceLocal && !filepath.IsAbs(defaultFile) {
		selectedFile = filepath.Join(workingDir, defaultFile)
	}

	// If there's no initial file, skip the start screen and go directly to source selection
	initialStage := selectFileStageStart
	if config.InitialFile == "" {
		initialStage = selectFileStageSelectSource
	}

	return &SelectFileModel{
		selectLocalFile: selectLocalFile,
		inputRemoteFile: inputRemoteFile,
		autoSelect:      config.AutoSelect,
		start:           start,
		source:          source,
		selectedSource:  defaultSource,
		selectedFile:    selectedFile,
		stage:           initialStage,
		config:          config,
	}, nil
}

func selectFileWorkingDir(fp *filepicker.Model) (string, error) {
	if fp != nil && fp.CurrentDirectory != "" {
		return fp.CurrentDirectory, nil
	}
	return os.Getwd()
}

// A public HTTPS URL can only be read from, so it is never offered as somewhere to save.
func selectFileSources(config SelectFileConfig) []string {
	if config.Mode != SelectFileModeSave {
		return config.SupportedSources
	}

	sources := config.SupportedSources
	if len(sources) == 0 {
		sources = make([]string, 0, len(allFileSourceItems))
		for _, item := range allFileSourceItems {
			sources = append(sources, item.Key)
		}
	}
	return slices.DeleteFunc(slices.Clone(sources), func(source string) bool {
		return source == consts.FileSourceHTTPS
	})
}

func selectFileSourcePrompt(config SelectFileConfig, fileTypeName string) string {
	if config.Mode == SelectFileModeSave {
		return fmt.Sprintf("Where do you want to save the %s?", fileTypeName)
	}
	return fmt.Sprintf("Where is the %s that you want to %s stored?", fileTypeName, config.Action)
}

func newLocalFileModel(config SelectFileConfig, fileTypeName string, workingDir string) (tea.Model, error) {
	if config.Mode == SelectFileModeSave {
		return NewInputLocalFile(config.Styles, fileTypeName, config.DefaultFileName, workingDir), nil
	}
	localFilePrompt := fmt.Sprintf("Pick a %s:", fileTypeName)
	return NewSelectLocalFile(config.FilePicker, config.Styles, localFilePrompt, fileTypeName)
}

// The remote form's default wording describes a file being read, so save mode fills in
// wording for a file being written for each source the caller has not worded itself.
func remoteFileConfig(config SelectFileConfig) *SelectRemoteFileConfig {
	remoteConfig := SelectRemoteFileConfig{}
	if config.RemoteFileConfig != nil {
		remoteConfig = *config.RemoteFileConfig
	}
	if config.Mode != SelectFileModeSave {
		return &remoteConfig
	}

	remoteConfig.BucketNameDescription = withDefaults(remoteConfig.BucketNameDescription, map[string]string{
		consts.FileSourceS3:        "The name of the S3 bucket to save the file to.",
		consts.FileSourceGCS:       "The name of the GCS bucket to save the file to.",
		consts.FileSourceAzureBlob: "The name of the Azure Blob Storage container to save the file to.",
	})
	remoteConfig.ObjectPathDescription = withDefaults(remoteConfig.ObjectPathDescription, map[string]string{
		consts.FileSourceS3:        "The path to save the file to in the S3 bucket.",
		consts.FileSourceGCS:       "The path to save the file to in the GCS bucket.",
		consts.FileSourceAzureBlob: "The path to save the file to in the Azure Blob Storage container.",
	})
	return &remoteConfig
}

// The caller's own wording wins, and defaults only fill the gaps it leaves.
func withDefaults(values map[string]string, defaults map[string]string) map[string]string {
	merged := maps.Clone(defaults)
	maps.Copy(merged, values)
	return merged
}

func initialFileAndSource(filePath string) (string, string) {
	if strings.HasPrefix(filePath, "https://") {
		return filePath, consts.FileSourceHTTPS
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "s3://"); ok {
		return withoutScheme, consts.FileSourceS3
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "gcs://"); ok {
		return withoutScheme, consts.FileSourceGCS
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "azureblob://"); ok {
		return withoutScheme, consts.FileSourceAzureBlob
	}

	return filePath, consts.FileSourceLocal
}

func fileStageFromSource(source string) selectFileStage {
	switch source {
	case consts.FileSourceLocal:
		return selectFileStageSelectLocalFile
	case consts.FileSourceS3, consts.FileSourceGCS, consts.FileSourceAzureBlob, consts.FileSourceHTTPS:
		return selectFileStageInputFileLocation
	}
	return selectFileStageSelectSource
}

func fileStageFromStartSelection(selection string) selectFileStage {
	switch selection {
	case "default":
		return selectFileStageSelected
	case "select":
		return selectFileStageSelectSource
	}
	return selectFileStageStart
}

// ToFullFilePath returns the full path including the scheme for remote sources.
func ToFullFilePath(file, source string) string {
	switch source {
	case consts.FileSourceS3:
		return "s3://" + file
	case consts.FileSourceGCS:
		return "gcs://" + file
	case consts.FileSourceAzureBlob:
		return "azureblob://" + file
	}
	return file
}

func selectFileCmd(file, source string) tea.Cmd {
	return func() tea.Msg {
		return SelectFileMsg{
			File:   file,
			Source: source,
		}
	}
}

func selectFileSourceCmd(source string) tea.Cmd {
	return func() tea.Msg {
		return SelectFileSourceMsg{
			Source: source,
		}
	}
}

func clearSelectedFileCmd() tea.Cmd {
	return func() tea.Msg {
		return ClearSelectedFileMsg{}
	}
}

func selectFileErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return SelectFileErrorMsg{
			Err: err,
		}
	}
}

func selectFileStartCmd(selection string) tea.Cmd {
	return func() tea.Msg {
		return SelectFileStartMsg{
			Selection: selection,
		}
	}
}

func selectFileStartListItems(hasInitialFile bool) []list.Item {
	if !hasInitialFile {
		// When there's no initial file, skip the start screen entirely
		// by returning items that will trigger source selection
		return []list.Item{
			BluelinkListItem{Key: "select", Label: "Select a file"},
		}
	}
	return []list.Item{
		BluelinkListItem{Key: "default", Label: "Use the default file"},
		BluelinkListItem{Key: "select", Label: "Select a different file"},
	}
}

// allFileSourceItems contains the full list of file source options.
var allFileSourceItems = []BluelinkListItem{
	{Key: consts.FileSourceLocal, Label: "Local file"},
	{Key: consts.FileSourceS3, Label: "AWS S3 Bucket"},
	{Key: consts.FileSourceGCS, Label: "Google Cloud Storage Bucket"},
	{Key: consts.FileSourceAzureBlob, Label: "Azure Blob Storage Container"},
	{Key: consts.FileSourceHTTPS, Label: "Public HTTPS URL"},
}

// fileSourceListItems returns the list items for file source selection.
// If supportedSources is nil or empty, all sources are returned.
func fileSourceListItems(supportedSources []string) []list.Item {
	if len(supportedSources) == 0 {
		items := make([]list.Item, len(allFileSourceItems))
		for i, item := range allFileSourceItems {
			items[i] = item
		}
		return items
	}

	supportedSet := make(map[string]bool, len(supportedSources))
	for _, s := range supportedSources {
		supportedSet[s] = true
	}

	var items []list.Item
	for _, item := range allFileSourceItems {
		if supportedSet[item.Key] {
			items = append(items, item)
		}
	}
	return items
}

// clearFileErrorAfter returns a command that clears errors after a duration.
func clearFileErrorAfter(t time.Duration) tea.Cmd {
	return clearErrorAfter(t)
}
