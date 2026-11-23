package stateimportui

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
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type StateImportTUISuite struct {
	suite.Suite
	tempDir      string
	stateDir     string
	testFile     string
	engineConfig *stateio.EngineConfig
}

func (s *StateImportTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "stateimport-tui-test-*")
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

	// Create a test JSON file with instances
	// JSON format matches state.InstanceState: id, name, status (int where 2 = deployed)
	s.testFile = filepath.Join(tempDir, "state.json")
	instancesJSON := `[{"id":"inst-1","name":"Test Instance","status":2}]`
	err = os.WriteFile(s.testFile, []byte(instancesJSON), 0644)
	s.Require().NoError(err)
}

func (s *StateImportTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *StateImportTUISuite) Test_successful_import() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
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
		"Import complete",
		"Target: memfile state in",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *StateImportTUISuite) Test_successful_import_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
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
		"Successfully imported",
		"Target: memfile state in",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateImportTUISuite) Test_json_output_mode() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
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
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateImportTUISuite) Test_import_error_displayed() {
	// Use a non-existent file to trigger an error
	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.json")

	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       nonExistentFile,
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
		"Import failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

func (s *StateImportTUISuite) Test_import_error_headless() {
	// Use a non-existent file to trigger an error
	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.json")

	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       nonExistentFile,
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
		"Import failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateImportTUISuite) Test_import_error_json_output() {
	// Use a non-existent file to trigger an error
	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.json")

	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       nonExistentFile,
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
		`"success": false`,
		`"error"`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateImportTUISuite) Test_imports_a_file_chosen_in_the_file_selector() {
	// The file picker lists the working directory, so it holds only the file to choose.
	pickDir := filepath.Join(s.tempDir, "pick")
	s.Require().NoError(os.MkdirAll(pickDir, 0755))
	instancesJSON, err := os.ReadFile(s.testFile)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(filepath.Join(pickDir, "chosen.json"), instancesJSON, 0644))
	s.T().Chdir(pickDir)

	mainModel, err := NewStateImportApp(StateImportAppConfig{
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

	testutils.WaitForContains(s.T(), testModel.Output(), "Local file")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "chosen.json")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Import complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *StateImportTUISuite) Test_wraps_a_long_target_under_its_label() {
	longStateDir := filepath.Join(s.tempDir, "a-fairly-long-directory-name", "and-another-nested-directory", "state")
	s.Require().NoError(os.MkdirAll(longStateDir, 0755))
	s.engineConfig.State.MemFileStateDir = longStateDir

	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	const width = 60
	testModel := teatest.NewTestModel(s.T(), mainModel, teatest.WithInitialTermSize(width, 40))
	testutils.WaitForContains(s.T(), testModel.Output(), "Import complete")
	// Quitting from outside keeps the result on screen, where q replaces it with a goodbye.
	s.Require().NoError(testModel.Quit())

	view := testModel.FinalModel(s.T()).View()
	lines := strings.Split(view, "\n")
	targetLine := slices.IndexFunc(lines, func(line string) bool {
		return strings.HasPrefix(line, "    Target: memfile state in")
	})
	s.Require().GreaterOrEqual(targetLine, 0, view)

	continuation := lines[targetLine+1]
	s.True(strings.HasPrefix(continuation, strings.Repeat(" ", len("    Target: "))), view)
	for _, line := range lines {
		s.LessOrEqual(lipgloss.Width(line), width, line)
	}
}

func (s *StateImportTUISuite) Test_ctrl_c_quits() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
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
		"Import",
	)

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.True(finalModel.quitting)
}

func TestStateImportTUISuite(t *testing.T) {
	suite.Run(t, new(StateImportTUISuite))
}
