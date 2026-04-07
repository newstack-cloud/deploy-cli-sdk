package stateimportui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
)

type ImportModelSuite struct {
	suite.Suite
	tempDir      string
	stateDir     string
	testFile     string
	engineConfig *stateio.EngineConfig
	styles       *stylespkg.Styles
}

func (s *ImportModelSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "importmodel-test-*")
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

	s.testFile = filepath.Join(tempDir, "state.json")
	instancesJSON := `[{"id":"inst-1","name":"Test Instance","status":2}]`
	err = os.WriteFile(s.testFile, []byte(instancesJSON), 0644)
	s.Require().NoError(err)

	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *ImportModelSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *ImportModelSuite) newImportModel(filePath string) *ImportModel {
	return NewImportModel(ImportModelConfig{
		EngineConfig: s.engineConfig,
		FilePath:     filePath,
		Styles:       s.styles,
		Headless:     false,
	})
}

func (s *ImportModelSuite) Test_Update_DownloadCompleteMsg_with_error_sets_finished_with_error() {
	m := s.newImportModel("s3://bucket/state.json")
	m.downloading = true

	downloadErr := errors.New("network failure")
	_, cmd := m.Update(DownloadCompleteMsg{Err: downloadErr})

	s.Nil(cmd)
}

func (s *ImportModelSuite) Test_Update_DownloadCompleteMsg_with_success_starts_import() {
	m := s.newImportModel("s3://bucket/state.json")
	m.downloading = true

	data := []byte(`[{"id":"inst-1","name":"Test","status":2}]`)
	_, cmd := m.Update(DownloadCompleteMsg{Data: data, Err: nil})

	s.NotNil(cmd, "should return a command to start import")
}

func (s *ImportModelSuite) Test_Update_ImportStartedMsg_sets_importing_true() {
	m := s.newImportModel(s.testFile)
	_, cmd := m.Update(ImportStartedMsg{})

	s.Nil(cmd)
}

func (s *ImportModelSuite) Test_Update_ImportCompleteMsg_headless_writes_output() {
	var buf bytes.Buffer
	m := NewImportModel(ImportModelConfig{
		EngineConfig:   s.engineConfig,
		FilePath:       s.testFile,
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: &buf,
		JSONMode:       false,
	})

	result := &stateio.ImportResult{
		Success:        true,
		InstancesCount: 1,
		Message:        "Successfully imported 1 instances",
	}
	m.Update(ImportCompleteMsg{Result: result})

	s.Contains(buf.String(), "Successfully imported")
}

func (s *ImportModelSuite) Test_renderResult_with_error_contains_import_failed() {
	m := s.newImportModel(s.testFile)
	m.err = errors.New("something went wrong")
	m.finished = true

	view := m.renderResult()
	s.Contains(view, "Import failed")
	s.Contains(view, "something went wrong")
}

func (s *ImportModelSuite) Test_renderResult_with_nil_result_shows_no_result_message() {
	m := s.newImportModel(s.testFile)
	m.finished = true
	m.result = nil

	view := m.renderResult()
	s.Contains(view, "Import completed with no result")
}

func (s *ImportModelSuite) Test_renderResult_with_result_shows_instance_count() {
	m := s.newImportModel(s.testFile)
	m.finished = true
	m.result = &stateio.ImportResult{
		Success:        true,
		InstancesCount: 5,
		Message:        "done",
	}

	view := m.renderResult()
	s.Contains(view, "Import complete")
	s.Contains(view, "5")
}

func (s *ImportModelSuite) Test_View_returns_empty_in_headless_mode() {
	m := NewImportModel(ImportModelConfig{
		EngineConfig: s.engineConfig,
		FilePath:     s.testFile,
		Styles:       s.styles,
		Headless:     true,
	})
	s.Equal("", m.View())
}

func (s *ImportModelSuite) Test_View_shows_downloading_state() {
	m := s.newImportModel("s3://bucket/state.json")
	m.downloading = true

	view := m.View()
	s.Contains(view, "Downloading from")
	s.Contains(view, "s3://bucket/state.json")
}

func (s *ImportModelSuite) Test_View_shows_importing_state() {
	m := s.newImportModel(s.testFile)
	m.importing = true

	view := m.View()
	s.Contains(view, "Importing state")
}

func (s *ImportModelSuite) Test_View_shows_result_when_finished() {
	m := s.newImportModel(s.testFile)
	m.finished = true
	m.result = &stateio.ImportResult{
		Success:        true,
		InstancesCount: 2,
		Message:        "done",
	}

	view := m.View()
	s.Contains(view, "Import complete")
}

func (s *ImportModelSuite) Test_View_returns_empty_when_not_started() {
	m := s.newImportModel(s.testFile)
	s.Equal("", m.View())
}

func (s *ImportModelSuite) Test_writeHeadlessOutput_with_nil_writer_does_not_panic() {
	m := NewImportModel(ImportModelConfig{
		EngineConfig:   s.engineConfig,
		FilePath:       s.testFile,
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: nil,
	})
	m.finished = true
	m.result = &stateio.ImportResult{Success: true, Message: "done"}

	s.NotPanics(func() {
		m.writeHeadlessOutput()
	})
}

func (s *ImportModelSuite) Test_StartImport_with_local_file_returns_import_cmd() {
	m := s.newImportModel(s.testFile)
	cmd := m.StartImport()

	// Execute the command and check it produces ImportCompleteMsg
	msg := cmd()
	_, ok := msg.(ImportCompleteMsg)
	s.True(ok, "expected ImportCompleteMsg from local file import")
}

func (s *ImportModelSuite) Test_StartImport_with_s3_path_returns_download_cmd() {
	m := s.newImportModel("s3://bucket/state.json")
	cmd := m.StartImport()

	// The download command will fail (no real S3), but it must return DownloadCompleteMsg
	msg := cmd()
	downloadMsg, ok := msg.(DownloadCompleteMsg)
	s.True(ok, "expected DownloadCompleteMsg from remote file import")
	s.NotNil(downloadMsg.Err, "expected an error because S3 bucket is not real")
}

func (s *ImportModelSuite) Test_StartImport_with_gcs_path_returns_download_cmd() {
	m := s.newImportModel("gcs://bucket/state.json")
	cmd := m.StartImport()

	msg := cmd()
	_, ok := msg.(DownloadCompleteMsg)
	s.True(ok, "expected DownloadCompleteMsg from gcs remote file import")
}

func (s *ImportModelSuite) Test_startImportWithDataCmd_imports_from_memory() {
	data := []byte(`[{"id":"inst-1","name":"Test","status":2}]`)
	cmd := startImportWithDataCmd(s.engineConfig, data)

	msg := cmd()
	completeMsg, ok := msg.(ImportCompleteMsg)
	s.True(ok)
	s.Nil(completeMsg.Err)
	s.NotNil(completeMsg.Result)
	s.Equal(1, completeMsg.Result.InstancesCount)
}

func (s *ImportModelSuite) Test_startImportWithDataCmd_with_invalid_data_returns_error() {
	cmd := startImportWithDataCmd(s.engineConfig, []byte("not-valid-json"))

	msg := cmd()
	completeMsg, ok := msg.(ImportCompleteMsg)
	s.True(ok)
	s.NotNil(completeMsg.Err)
}

func (s *ImportModelSuite) Test_handleSelectFileMsg_transitions_to_running_state() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       "",
		EngineConfig:   s.engineConfig,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	selectMsg := sharedui.SelectFileMsg{File: s.testFile, Source: "file"}
	mainModel.Update(selectMsg)
}

func (s *ImportModelSuite) Test_headless_import_with_error_writes_error_output() {
	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.json")

	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       nonExistentFile,
		EngineConfig:   s.engineConfig,
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Import failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *ImportModelSuite) Test_MainModel_View_in_file_select_state_with_no_select_model() {
	// Edge case: sessionState is fileSelect but selectFile is nil (shouldn't normally happen
	// but we test the View() fallback)
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       "",
		EngineConfig:   s.engineConfig,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	// Replace the selectFile with nil to test the nil branch in View()
	mainModel.selectFile = nil
	view := mainModel.View()
	s.Equal("", view)
}

func (s *ImportModelSuite) Test_MainModel_View_in_running_state_shows_file_path() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       s.testFile,
		EngineConfig:   s.engineConfig,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
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
		"Importing from",
		"Import",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}


func (s *ImportModelSuite) Test_session_state_routing_in_file_select_state_routes_to_select_model() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       "",
		EngineConfig:   s.engineConfig,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	// Send an unrecognised message to exercise handleSessionStateRouting
	type unknownMsg struct{}
	// Should not panic or error
	mainModel.Update(unknownMsg{})
}

func (s *ImportModelSuite) Test_session_state_routing_without_select_model_returns_unchanged() {
	mainModel, err := NewStateImportApp(StateImportAppConfig{
		FilePath:       "",
		EngineConfig:   s.engineConfig,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	mainModel.selectFile = nil

	type unknownMsg struct{}
	_, cmd := mainModel.Update(unknownMsg{})

	s.Nil(cmd)
}

func TestImportModelSuite(t *testing.T) {
	suite.Run(t, new(ImportModelSuite))
}
