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
	start            tea.Model
	selectLocation   tea.Model
	selectLocalFile  tea.Model
	inputRemoteFile  tea.Model
	selectedLocation string
	selectedFile     string
	autoSelect       bool
	stage            selectFileStage
	quitting         bool
	err              error
	config           SelectFileConfig
	width            int
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
	// SupportedLocations specifies which file locations are available for selection.
	// If nil or empty, all locations are supported (Local, S3, GCS, Azure Blob, HTTPS).
	SupportedLocations []string
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
	selectFileStageSelectLocation
	selectFileStageInputRemoteFile
	selectFileStageSelectLocalFile
	selectFileStageSelected
)

// SelectFileMsg is sent when a file has been selected.
type SelectFileMsg struct {
	File     string
	Location string
}

// SelectFileLocationMsg is sent when a location has been selected.
type SelectFileLocationMsg struct {
	Location string
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
		cmds = append(cmds, selectFileCmd(m.selectedFile, m.selectedLocation))
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
			cmds = append(cmds, selectFileCmd(m.selectedFile, m.selectedLocation))
		}
	case SelectFileLocationMsg:
		var locationCmd tea.Cmd
		m, locationCmd = m.handleLocationSelected(msg)
		cmds = append(cmds, locationCmd)
	case SelectFileMsg:
		m.selectedFile = msg.File
		m.selectedLocation = msg.Location
		m.stage = selectFileStageSelected
	case clearErrorMsg:
		m.err = nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		var startCmd tea.Cmd
		m.start, startCmd = m.start.Update(msg)
		cmds = append(cmds, startCmd)

		var locationCmd tea.Cmd
		m.selectLocation, locationCmd = m.selectLocation.Update(msg)
		cmds = append(cmds, locationCmd)

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
	case selectFileStageSelectLocation:
		var locationCmd tea.Cmd
		m.selectLocation, locationCmd = m.selectLocation.Update(msg)
		cmds = append(cmds, locationCmd)
	case selectFileStageInputRemoteFile:
		var inputRemoteFileCmd tea.Cmd
		m.inputRemoteFile, inputRemoteFileCmd = m.inputRemoteFile.Update(msg)
		cmds = append(cmds, inputRemoteFileCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SelectFileModel) handleLocationSelected(msg SelectFileLocationMsg) (SelectFileModel, tea.Cmd) {
	m.stage = fileStageFromLocation(msg.Location)
	m.selectedLocation = msg.Location
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
	case selectFileStageSelectLocation:
		s.WriteString("\n\n" + m.selectLocation.View() + "\n")
	case selectFileStageSelectLocalFile:
		s.WriteString("\n\n" + m.selectLocalFile.View() + "\n")
	case selectFileStageInputRemoteFile:
		s.WriteString("\n\n" + m.inputRemoteFile.View() + "\n")
	case selectFileStageSelected:
		fullPath := ToFullFilePath(m.selectedFile, m.selectedLocation)
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

// SelectedLocation returns the location type of the selected file.
func (m SelectFileModel) SelectedLocation() string {
	return m.selectedLocation
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
	locationPrompt := selectFileLocationPrompt(config, fileTypeName)

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

	locationItems := fileLocationListItems(selectFileLocations(config))
	selectLocation := NewSelect(
		locationPrompt,
		locationItems,
		config.Styles,
		selectFileLocationCmd,
		false,
	)

	inputRemoteFile := NewSelectRemoteFile(config.InitialFile, config.Styles, remoteFileConfig(config))

	defaultFile, defaultLocation := initialFileAndLocation(config.InitialFile)
	selectedFile := defaultFile
	if defaultLocation == consts.FileLocationLocal && !filepath.IsAbs(defaultFile) {
		selectedFile = filepath.Join(workingDir, defaultFile)
	}

	// If there's no initial file, skip the start screen and go directly to location selection
	initialStage := selectFileStageStart
	if config.InitialFile == "" {
		initialStage = selectFileStageSelectLocation
	}

	return &SelectFileModel{
		selectLocalFile:  selectLocalFile,
		inputRemoteFile:  inputRemoteFile,
		autoSelect:       config.AutoSelect,
		start:            start,
		selectLocation:   selectLocation,
		selectedLocation: defaultLocation,
		selectedFile:     selectedFile,
		stage:            initialStage,
		config:           config,
	}, nil
}

func selectFileWorkingDir(fp *filepicker.Model) (string, error) {
	if fp != nil && fp.CurrentDirectory != "" {
		return fp.CurrentDirectory, nil
	}
	return os.Getwd()
}

// A public HTTPS URL can only be read from, so it is never offered as somewhere to save.
func selectFileLocations(config SelectFileConfig) []string {
	if config.Mode != SelectFileModeSave {
		return config.SupportedLocations
	}

	locations := config.SupportedLocations
	if len(locations) == 0 {
		locations = make([]string, 0, len(allFileLocationItems))
		for _, item := range allFileLocationItems {
			locations = append(locations, item.Key)
		}
	}
	return slices.DeleteFunc(slices.Clone(locations), func(location string) bool {
		return location == consts.FileLocationHTTPS
	})
}

func selectFileLocationPrompt(config SelectFileConfig, fileTypeName string) string {
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
// wording for a file being written for each location the caller has not worded itself.
func remoteFileConfig(config SelectFileConfig) *SelectRemoteFileConfig {
	remoteConfig := SelectRemoteFileConfig{}
	if config.RemoteFileConfig != nil {
		remoteConfig = *config.RemoteFileConfig
	}
	if config.Mode != SelectFileModeSave {
		return &remoteConfig
	}

	remoteConfig.BucketNameDescription = withDefaults(remoteConfig.BucketNameDescription, map[string]string{
		consts.FileLocationS3:        "The name of the S3 bucket to save the file to.",
		consts.FileLocationGCS:       "The name of the GCS bucket to save the file to.",
		consts.FileLocationAzureBlob: "The name of the Azure Blob Storage container to save the file to.",
	})
	remoteConfig.ObjectPathDescription = withDefaults(remoteConfig.ObjectPathDescription, map[string]string{
		consts.FileLocationS3:        "The path to save the file to in the S3 bucket.",
		consts.FileLocationGCS:       "The path to save the file to in the GCS bucket.",
		consts.FileLocationAzureBlob: "The path to save the file to in the Azure Blob Storage container.",
	})
	return &remoteConfig
}

// The caller's own wording wins, and defaults only fill the gaps it leaves.
func withDefaults(values map[string]string, defaults map[string]string) map[string]string {
	merged := maps.Clone(defaults)
	maps.Copy(merged, values)
	return merged
}

func initialFileAndLocation(filePath string) (string, string) {
	if strings.HasPrefix(filePath, "https://") {
		return filePath, consts.FileLocationHTTPS
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "s3://"); ok {
		return withoutScheme, consts.FileLocationS3
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "gcs://"); ok {
		return withoutScheme, consts.FileLocationGCS
	}

	if withoutScheme, ok := strings.CutPrefix(filePath, "azureblob://"); ok {
		return withoutScheme, consts.FileLocationAzureBlob
	}

	return filePath, consts.FileLocationLocal
}

func fileStageFromLocation(location string) selectFileStage {
	switch location {
	case consts.FileLocationLocal:
		return selectFileStageSelectLocalFile
	case consts.FileLocationS3, consts.FileLocationGCS, consts.FileLocationAzureBlob, consts.FileLocationHTTPS:
		return selectFileStageInputRemoteFile
	}
	return selectFileStageSelectLocation
}

func fileStageFromStartSelection(selection string) selectFileStage {
	switch selection {
	case "default":
		return selectFileStageSelected
	case "select":
		return selectFileStageSelectLocation
	}
	return selectFileStageStart
}

// ToFullFilePath returns the full path including the scheme for remote locations.
func ToFullFilePath(file, location string) string {
	switch location {
	case consts.FileLocationS3:
		return "s3://" + file
	case consts.FileLocationGCS:
		return "gcs://" + file
	case consts.FileLocationAzureBlob:
		return "azureblob://" + file
	}
	return file
}

func selectFileCmd(file, location string) tea.Cmd {
	return func() tea.Msg {
		return SelectFileMsg{
			File:     file,
			Location: location,
		}
	}
}

func selectFileLocationCmd(location string) tea.Cmd {
	return func() tea.Msg {
		return SelectFileLocationMsg{
			Location: location,
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
		// by returning items that will trigger location selection
		return []list.Item{
			BluelinkListItem{Key: "select", Label: "Select a file"},
		}
	}
	return []list.Item{
		BluelinkListItem{Key: "default", Label: "Use the default file"},
		BluelinkListItem{Key: "select", Label: "Select a different file"},
	}
}

var allFileLocationItems = []BluelinkListItem{
	{Key: consts.FileLocationLocal, Label: "Local file"},
	{Key: consts.FileLocationS3, Label: "AWS S3 Bucket"},
	{Key: consts.FileLocationGCS, Label: "Google Cloud Storage Bucket"},
	{Key: consts.FileLocationAzureBlob, Label: "Azure Blob Storage Container"},
	{Key: consts.FileLocationHTTPS, Label: "Public HTTPS URL"},
}

func fileLocationListItems(supportedLocations []string) []list.Item {
	if len(supportedLocations) == 0 {
		items := make([]list.Item, len(allFileLocationItems))
		for i, item := range allFileLocationItems {
			items[i] = item
		}
		return items
	}

	supportedSet := make(map[string]bool, len(supportedLocations))
	for _, s := range supportedLocations {
		supportedSet[s] = true
	}

	var items []list.Item
	for _, item := range allFileLocationItems {
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
