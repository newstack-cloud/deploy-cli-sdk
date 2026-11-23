package stateexportui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

type ExportModelSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func (s *ExportModelSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *ExportModelSuite) newModel(cfg ExportModelConfig) *ExportModel {
	if cfg.Styles == nil {
		cfg.Styles = s.styles
	}
	return NewExportModel(cfg)
}

func (s *ExportModelSuite) Test_View_shows_a_failed_export() {
	testErr := errors.New("something went wrong during export")
	m := s.newModel(ExportModelConfig{
		FilePath: "/tmp/out.json",
	})
	m.Update(ExportCompleteMsg{Err: testErr})

	view := m.View()
	s.Equal(
		"\n  ✗ Export failed\n\n"+
			"    Source: state\n"+
			"    Error:  something went wrong during export\n\n"+
			"  Press q to quit\n",
		view,
	)
}

func (s *ExportModelSuite) Test_View_shows_an_export_with_no_result() {
	m := s.newModel(ExportModelConfig{
		FilePath: "/tmp/out.json",
	})
	m.Update(ExportCompleteMsg{})

	view := m.View()
	s.Equal("\n  Export completed with no result.\n\n  Press q to quit\n", view)
}

func (s *ExportModelSuite) Test_View_exporting_describes_the_source_state() {
	m := s.newModel(ExportModelConfig{
		FilePath: "/tmp/mystate.json",
	})
	m.Update(ExportStartedMsg{})

	view := m.View()
	s.Equal("\n  ⣾  Exporting state...\n", view)
}

func (s *ExportModelSuite) Test_headless_text_output_with_error() {
	var buf bytes.Buffer
	m := s.newModel(ExportModelConfig{
		Headless:       true,
		HeadlessWriter: &buf,
	})
	m.Update(ExportCompleteMsg{Err: fmt.Errorf("disk full")})

	s.Equal("Export failed: disk full\n  Source: state\n", buf.String())
}

func (s *ExportModelSuite) Test_headless_text_output_with_result() {
	var buf bytes.Buffer
	m := s.newModel(ExportModelConfig{
		Headless:       true,
		HeadlessWriter: &buf,
	})
	m.Update(ExportCompleteMsg{Result: &stateio.ExportResult{
		Success:        true,
		InstancesCount: 3,
		Message:        "Successfully exported 3 instances",
	}})

	s.Equal("Successfully exported 3 instances\nSource: state\n", buf.String())
}

func TestExportModelSuite(t *testing.T) {
	suite.Run(t, new(ExportModelSuite))
}

type MainModelInteractiveSuite struct {
	suite.Suite
	tempDir      string
	stateDir     string
	outputFile   string
	engineConfig *stateio.EngineConfig
	styles       *stylespkg.Styles
}

func (s *MainModelInteractiveSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "stateexport-interactive-test-*")
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

	s.outputFile = filepath.Join(tempDir, "exported.json")
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())

	// Pre-populate state with test data
	inputFile := filepath.Join(tempDir, "input.json")
	instancesJSON := `[{"id":"inst-1","name":"Test Instance","status":2}]`
	err = os.WriteFile(inputFile, []byte(instancesJSON), 0644)
	s.Require().NoError(err)

	_, err = stateio.Import(stateio.ImportParams{
		FilePath:     inputFile,
		EngineConfig: s.engineConfig,
	})
	s.Require().NoError(err)
}

func (s *MainModelInteractiveSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

// Test_View_file_select_state verifies the View delegates to selectFile.
func (s *MainModelInteractiveSuite) Test_View_interactive_mode_shows_file_select() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:     "",
		EngineConfig: s.engineConfig,
		Styles:       s.styles,
		Headless:     false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// The file select view should appear
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Where do you want to save the output file?",
	)

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// Test_export_error_tui verifies the non-headless error display path.
func (s *MainModelInteractiveSuite) Test_export_error_tui_shows_failure() {
	// Use a bad engine config to force an export error.
	// Use an unsupported storage engine to force an export error.
	// Loading a missing memfile dir is no longer an error in
	// blueprint-state v0.8.0+ — it's treated as empty state.
	badConfig := &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine: "unsupported-engine-type",
		},
	}

	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:     s.outputFile,
		EngineConfig: badConfig,
		Styles:       s.styles,
		Headless:     false,
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
		"Export failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

// Test_headless_export_with_error verifies error output in text headless mode.
func (s *MainModelInteractiveSuite) Test_headless_export_with_error_text_output() {
	// Use an unsupported storage engine to force an export error.
	// Loading a missing memfile dir is no longer an error in
	// blueprint-state v0.8.0+ — it's treated as empty state.
	badConfig := &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine: "unsupported-engine-type",
		},
	}

	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   badConfig,
		Styles:         s.styles,
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

	testutils.WaitFor(s.T(), headlessOutput, func(bts []byte) bool {
		return strings.Contains(string(bts), "Export failed")
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// Test_headless_export_with_error_json verifies JSON error output branch.
func (s *MainModelInteractiveSuite) Test_headless_export_with_error_json_output() {
	// Use an unsupported storage engine to force an export error.
	// Loading a missing memfile dir is no longer an error in
	// blueprint-state v0.8.0+ — it's treated as empty state.
	badConfig := &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine: "unsupported-engine-type",
		},
	}

	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   badConfig,
		Styles:         s.styles,
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

// Test_NewStateExportApp_interactive_mode tests createSelectFileModel/createFilePicker code path.
func (s *MainModelInteractiveSuite) Test_NewStateExportApp_interactive_mode_creates_select_model() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:     "",
		EngineConfig: s.engineConfig,
		Styles:       s.styles,
		Headless:     false,
	})
	s.Require().NoError(err)
	s.NotNil(mainModel)
}

// Test_View_quitting shows the quit message.
func (s *MainModelInteractiveSuite) Test_View_quitting_renders_goodbye() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:     s.outputFile,
		EngineConfig: s.engineConfig,
		Styles:       s.styles,
		Headless:     false,
	})
	s.Require().NoError(err)

	quitModel, _ := mainModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	view := quitModel.View()
	s.Contains(view, "See you next time.")
}

func (s *MainModelInteractiveSuite) Test_q_does_not_quit_during_export() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:     s.outputFile,
		EngineConfig: s.engineConfig,
		Styles:       s.styles,
		Headless:     false,
	})
	s.Require().NoError(err)

	// Given an output file, the export starts running straight away.
	_, cmd := mainModel.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("q"),
	})

	s.Nil(cmd)
}

func TestMainModelInteractiveSuite(t *testing.T) {
	suite.Run(t, new(MainModelInteractiveSuite))
}
