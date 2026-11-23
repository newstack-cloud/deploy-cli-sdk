package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type SelectFileSaveSuite struct {
	suite.Suite
	workingDir string
}

func (s *SelectFileSaveSuite) SetupTest() {
	s.workingDir = s.T().TempDir()
	s.T().Chdir(s.workingDir)
}

func (s *SelectFileSaveSuite) startSaveSelector() *teatest.TestModel {
	return s.startSaveSelectorWithWidth(300)
}

func (s *SelectFileSaveSuite) startSaveSelectorWithWidth(width int) *teatest.TestModel {
	selectModel, err := NewSelectFile(SelectFileConfig{
		Action:          "export",
		Styles:          styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette()),
		FileTypeName:    "output file",
		Mode:            SelectFileModeSave,
		DefaultFileName: "state-export.json",
		SupportedLocations: []string{
			consts.FileLocationLocal,
			consts.FileLocationS3,
		},
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), selectModel, teatest.WithInitialTermSize(width, 100))
	testutils.WaitForContains(s.T(), testModel.Output(), "Where do you want to save the output file?")
	return testModel
}

func (s *SelectFileSaveSuite) chooseLocalFile(testModel *teatest.TestModel) {
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "Path of the output file")
}

func (s *SelectFileSaveSuite) replacePath(testModel *teatest.TestModel, path string) {
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	testModel.Type(path)
}

func (s *SelectFileSaveSuite) finalSelection(testModel *teatest.TestModel) SelectFileModel {
	s.Require().NoError(testModel.Quit())
	finalModel, ok := testModel.FinalModel(s.T()).(SelectFileModel)
	s.Require().True(ok)
	return finalModel
}

func (s *SelectFileSaveSuite) Test_accepts_the_default_file_name_in_the_working_directory() {
	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	testutils.KeyEnter(testModel)

	expected := filepath.Join(s.workingDir, "state-export.json")
	testutils.WaitForContains(s.T(), testModel.Output(), "Output file selected: "+expected)

	finalModel := s.finalSelection(testModel)
	s.Equal(expected, finalModel.SelectedFile())
	s.Equal(consts.FileLocationLocal, finalModel.SelectedLocation())
}

func (s *SelectFileSaveSuite) Test_accepts_a_new_file_in_a_subdirectory() {
	s.Require().NoError(os.Mkdir(filepath.Join(s.workingDir, "exports"), 0755))

	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "exports/prod.json")
	testutils.KeyEnter(testModel)

	expected := filepath.Join(s.workingDir, "exports", "prod.json")
	testutils.WaitForContains(s.T(), testModel.Output(), "Output file selected: "+expected)
	s.Equal(expected, s.finalSelection(testModel).SelectedFile())
}

func (s *SelectFileSaveSuite) Test_warns_that_an_existing_file_will_be_overwritten() {
	// A long working directory catches a warning that wraps and pushes the input out of view.
	longDir := filepath.Join(s.workingDir, "a-fairly-long-directory-name", "and-another-nested-directory-name")
	s.Require().NoError(os.MkdirAll(longDir, 0755))
	s.Require().NoError(os.WriteFile(filepath.Join(longDir, "existing.json"), []byte("[]"), 0644))
	s.T().Chdir(longDir)

	testModel := s.startSaveSelectorWithWidth(100)
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "existing.json")

	testutils.WaitForContains(s.T(), testModel.Output(), "! This file already exists and will be overwritten.")
	view := s.finalSelection(testModel).View()
	s.Contains(view, "Path of the output file")
	s.Contains(view, "! This file already exists and will be overwritten.")
	s.Contains(view, "> existing.json")
}

func (s *SelectFileSaveSuite) Test_rejects_a_path_in_a_missing_directory() {
	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "missing/prod.json")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "does not exist")
	s.NotContains(s.finalSelection(testModel).View(), "Output file selected")
}

func (s *SelectFileSaveSuite) Test_rejects_a_directory() {
	s.Require().NoError(os.Mkdir(filepath.Join(s.workingDir, "exports"), 0755))

	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "exports")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "is a directory")
	s.Require().NoError(testModel.Quit())
}

func (s *SelectFileSaveSuite) Test_describes_a_remote_destination_as_a_file_being_saved() {
	testModel := s.startSaveSelector()
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "The name of the S3 bucket to save the file to.")
	s.Require().NoError(testModel.Quit())
}

func (s *SelectFileSaveSuite) Test_expands_a_home_relative_path() {
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)

	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "~/prod.json")
	testutils.KeyEnter(testModel)

	expected := filepath.Join(home, "prod.json")
	testutils.WaitForContains(s.T(), testModel.Output(), "Output file selected: "+expected)
	s.Equal(expected, s.finalSelection(testModel).SelectedFile())
}

func (s *SelectFileSaveSuite) Test_rejects_a_parent_that_is_a_file() {
	s.Require().NoError(os.WriteFile(filepath.Join(s.workingDir, "existing.json"), []byte("[]"), 0644))

	testModel := s.startSaveSelector()
	s.chooseLocalFile(testModel)
	s.replacePath(testModel, "existing.json/out.json")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "existing.json is a file, not a directory")
	s.Require().NoError(testModel.Quit())
}

func (s *SelectFileSaveSuite) Test_saves_to_a_remote_object_path() {
	testModel := s.startSaveSelector()
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "S3 bucket")
	testModel.Type("state-bucket")
	testutils.KeyEnter(testModel)
	testutils.WaitForContains(s.T(), testModel.Output(), "The path to save the file to in the S3 bucket.")
	testModel.Type("exports/prod.json")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "Output file selected: s3://state-bucket/exports/prod.json")
	finalModel := s.finalSelection(testModel)
	s.Equal("state-bucket/exports/prod.json", finalModel.SelectedFile())
	s.Equal(consts.FileLocationS3, finalModel.SelectedLocation())
}

func (s *SelectFileSaveSuite) Test_never_offers_an_https_url_to_save_to() {
	selectModel, err := NewSelectFile(SelectFileConfig{
		Action:       "export",
		Styles:       styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette()),
		FileTypeName: "output file",
		Mode:         SelectFileModeSave,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), selectModel, teatest.WithInitialTermSize(300, 100))
	testutils.WaitForContains(s.T(), testModel.Output(), "Azure Blob Storage Container")
	s.Require().NoError(testModel.Quit())
	s.NotContains(s.finalSelection(testModel).View(), "Public HTTPS URL")
}

func (s *SelectFileSaveSuite) Test_fills_in_save_wording_for_locations_the_caller_did_not_word() {
	selectModel, err := NewSelectFile(SelectFileConfig{
		Action:       "export",
		Styles:       styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette()),
		FileTypeName: "output file",
		Mode:         SelectFileModeSave,
		RemoteFileConfig: &SelectRemoteFileConfig{
			BucketNameDescription: map[string]string{consts.FileLocationS3: "The S3 bucket for exports."},
		},
		SupportedLocations: []string{consts.FileLocationS3, consts.FileLocationGCS},
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), selectModel, teatest.WithInitialTermSize(300, 100))
	testutils.WaitForContains(s.T(), testModel.Output(), "Where do you want to save the output file?")
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(s.T(), testModel.Output(), "The name of the GCS bucket to save the file to.")
	s.Require().NoError(testModel.Quit())
}

func TestSelectFileSaveSuite(t *testing.T) {
	suite.Run(t, new(SelectFileSaveSuite))
}
