package stateexportui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
)

type StateExportTUISuite struct {
	suite.Suite
	tempDir      string
	stateDir     string
	outputFile   string
	engineConfig *stateio.EngineConfig
}

func (s *StateExportTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "stateexport-tui-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.stateDir = filepath.Join(tempDir, "state")
	err = os.MkdirAll(s.stateDir, 0755)
	s.Require().NoError(err)

	s.engineConfig = &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine:   stateio.StorageEngineMemfile,
			MemFileStateDir: s.stateDir,
		},
	}

	// Set up an output file path for export
	s.outputFile = filepath.Join(tempDir, "exported.json")

	// Pre-populate the memfile state with test data by importing
	inputFile := filepath.Join(tempDir, "input.json")
	instancesJSON := `[{"id":"inst-1","name":"Test Instance","status":2}]`
	err = os.WriteFile(inputFile, []byte(instancesJSON), 0644)
	s.Require().NoError(err)

	// Import the test data
	_, err = stateio.Import(stateio.ImportParams{
		FilePath:     inputFile,
		EngineConfig: s.engineConfig,
	})
	s.Require().NoError(err)
}

func (s *StateExportTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *StateExportTUISuite) Test_successful_export() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Export complete",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify the output file was created
	_, err = os.Stat(s.outputFile)
	s.NoError(err)
}

func (s *StateExportTUISuite) Test_successful_export_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       true,
		HeadlessWriter: headlessOutput,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Successfully exported",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_json_output_mode() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       true,
		HeadlessWriter: headlessOutput,
		JSONMode:       true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": true`,
		`"instancesCount"`,
		`"mode": "export"`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_export_with_instance_filter() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:        s.outputFile,
		InstanceFilters: []string{"inst-1"},
		EngineConfig:    s.engineConfig,
		Styles:          stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:        true,
		HeadlessWriter:  headlessOutput,
		JSONMode:        true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": true`,
		`"instancesCount": 1`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_export_error_invalid_instance() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:        s.outputFile,
		InstanceFilters: []string{"nonexistent-instance"},
		EngineConfig:    s.engineConfig,
		Styles:          stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:        true,
		HeadlessWriter:  headlessOutput,
		JSONMode:        true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": false`,
		`"error"`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_exports_to_a_new_file_entered_in_the_file_selector() {
	s.T().Chdir(s.tempDir)

	mainModel, err := NewStateExportApp(StateExportAppConfig{
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContains(s.T(), testModel.Output(), "Where do you want to save the output file?")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Path of the output file")
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	testModel.Type("new-export.json")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Export complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	exported, err := os.ReadFile(filepath.Join(s.tempDir, "new-export.json"))
	s.Require().NoError(err)
	s.Contains(string(exported), "Test Instance")
}

func (s *StateExportTUISuite) Test_ctrl_c_quits() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for some output to ensure the model is running
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Export",
	)

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.True(finalModel.quitting)
}

func (s *StateExportTUISuite) encryptedEngineConfig() *stateio.EngineConfig {
	stateDir := filepath.Join(s.tempDir, "encrypted-state")
	s.Require().NoError(os.MkdirAll(stateDir, 0700))

	config := &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine:   stateio.StorageEngineMemfile,
			MemFileStateDir: stateDir,
			Encryption: stateio.StateEncryptionConfig{
				Enabled: true,
				StateEncryptionKeyConfig: stateio.StateEncryptionKeyConfig{
					Provider:   "passphrase",
					Passphrase: "correct horse",
				},
			},
		},
	}

	_, err := stateio.Import(stateio.ImportParams{
		FilePath:     "input.json",
		FileData:     []byte(`[{"id":"inst-1","name":"Test Instance","status":2}]`),
		EngineConfig: config,
	})
	s.Require().NoError(err)

	return config
}

func (s *StateExportTUISuite) runEncryptedExport(headless bool, jsonMode bool, expected ...string) {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.encryptedEngineConfig(),
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       headless,
		HeadlessWriter: headlessOutput,
		JSONMode:       jsonMode,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), mainModel, teatest.WithInitialTermSize(300, 100))
	if headless {
		testutils.WaitForContainsAll(s.T(), headlessOutput, expected...)
		testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
		return
	}

	testutils.WaitForContainsAll(s.T(), testModel.Output(), expected...)
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_warns_that_an_export_of_encrypted_state_is_plaintext() {
	s.runEncryptedExport(
		false, false, "Export complete", "Source: memfile state in", "encrypted at rest", "export file is not",
	)
}

func (s *StateExportTUISuite) Test_warns_that_an_export_of_encrypted_state_is_plaintext_headless() {
	s.runEncryptedExport(
		true, false, "Successfully exported", "Source: memfile state in", "Warning: State is encrypted at rest",
	)
}

func (s *StateExportTUISuite) Test_wraps_a_warning_under_its_marker() {
	const width = 60
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.encryptedEngineConfig(),
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), mainModel, teatest.WithInitialTermSize(width, 40))
	testutils.WaitForContains(s.T(), testModel.Output(), "Export complete")
	s.Require().NoError(testModel.Quit())

	lines := strings.Split(testModel.FinalModel(s.T()).View(), "\n")
	warningLine := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, "  ! ") })
	s.Require().GreaterOrEqual(warningLine, 0)
	s.True(strings.HasPrefix(lines[warningLine+1], "    "), lines[warningLine+1])
	for _, line := range lines {
		s.LessOrEqual(lipgloss.Width(line), width, line)
	}
}

func (s *StateExportTUISuite) Test_starts_one_export_for_a_repeated_file_selection() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)
	secondFile := filepath.Join(s.tempDir, "second.json")

	testModel := teatest.NewTestModel(s.T(), mainModel, teatest.WithInitialTermSize(300, 100))
	testutils.WaitForContains(s.T(), testModel.Output(), "Where do you want to save the output file?")
	testModel.Send(sharedui.SelectFileMsg{File: s.outputFile, Location: consts.FileLocationLocal})
	testModel.Send(sharedui.SelectFileMsg{File: secondFile, Location: consts.FileLocationLocal})
	testutils.WaitForContains(s.T(), testModel.Output(), "Export complete")
	s.Require().NoError(testModel.Quit())

	s.FileExists(s.outputFile)
	s.NoFileExists(secondFile)
}

func (s *StateExportTUISuite) Test_includes_warnings_in_json_output() {
	s.runEncryptedExport(true, true, `"warnings"`, "export file is not")
}

func TestStateExportTUISuite(t *testing.T) {
	suite.Run(t, new(StateExportTUISuite))
}
