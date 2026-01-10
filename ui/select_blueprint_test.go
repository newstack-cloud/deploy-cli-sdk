package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type SelectBlueprintSuite struct {
	suite.Suite
	tempDir string
}

func (s *SelectBlueprintSuite) SetupSuite() {
	s.tempDir = s.T().TempDir()

	files, err := os.ReadDir("__testdata")
	s.NoError(err)

	for _, file := range files {
		fileData, err := os.ReadFile(filepath.Join("__testdata", file.Name()))
		s.NoError(err)
		os.WriteFile(filepath.Join(s.tempDir, file.Name()), fileData, 0644)
	}
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_local_file() {
	styles := styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	// Make it so the file picker is in a temporary directory
	// with a text file and a blueprint file.
	fp.CurrentDirectory = s.tempDir
	fp.DirAllowed = false
	selectModel, err := NewSelectBlueprint(
		"",
		/* autoValidate */ false,
		"validate",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Since no initial file is provided, the UI should skip the start screen
	// and go directly to source selection
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Where is the blueprint file that you want to validate stored?",
	)

	// Select the first option that is for picking a local file.
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Pick a blueprint file:",
	)

	// Try to select the first file which is not a valid blueprint file.
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"other.txt is not a valid blueprint file.",
	)

	// Select the second option which is the valid blueprint file.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	blueprintPath := filepath.Join(s.tempDir, "project.blueprint.yml")
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		fmt.Sprintf("Blueprint file selected: %s", blueprintPath),
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	s.Assert().Equal(blueprintPath, finalModel.SelectedFile())
	s.Assert().Equal(consts.BlueprintSourceFile, finalModel.SelectedSource())
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_s3_file() {
	styles := styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	selectModel, err := NewSelectBlueprint(
		"",
		/* autoValidate */ false,
		"validate",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Since no initial file is provided, the UI should skip the start screen
	// and go directly to source selection
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Where is the blueprint file that you want to validate stored?",
	)

	// Select the second option that is for picking a remote file from S3.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"S3 bucket",
	)

	testModel.Type("test-bucket")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		// "enter submit" is the help text displayed when on the final step of the form.
		"enter submit",
	)

	testModel.Type("project1/project.blueprint.yml")
	testutils.KeyEnter(testModel)

	blueprintPath := "test-bucket/project1/project.blueprint.yml"
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		fmt.Sprintf("Blueprint file selected: s3://%s", blueprintPath),
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	s.Assert().Equal(blueprintPath, finalModel.SelectedFile())
	s.Assert().Equal(consts.BlueprintSourceS3, finalModel.SelectedSource())
}

func (s *SelectBlueprintSuite) Test_use_default_blueprint_file() {
	styles := styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	// Set the current directory to the temp directory
	fp.CurrentDirectory = s.tempDir

	// Set up with an initial/default file that exists in the temp directory
	defaultBlueprintFile := "project.blueprint.yml"
	selectModel, err := NewSelectBlueprint(
		defaultBlueprintFile,
		/* autoValidate */ false,
		"deploy",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Verify the option to use the default file is shown
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Use the default file",
	)

	// Select the first option (Use the default file)
	testutils.KeyEnter(testModel)

	// Verify the default file was selected (will show absolute path)
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Blueprint file selected:",
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	// The selected file will be the absolute path
	expectedPath := filepath.Join(s.tempDir, defaultBlueprintFile)
	s.Assert().Equal(expectedPath, finalModel.SelectedFile())
	s.Assert().Equal(consts.BlueprintSourceFile, finalModel.SelectedSource())
}

func (s *SelectBlueprintSuite) Test_select_different_file_when_default_exists() {
	styles := styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	// Make it so the file picker is in a temporary directory
	fp.CurrentDirectory = s.tempDir
	fp.DirAllowed = false

	// Set up with an initial/default file (using a different name than what's in temp dir)
	defaultBlueprintFile := "project.blueprint.yaml"
	selectModel, err := NewSelectBlueprint(
		defaultBlueprintFile,
		/* autoSelect */ false,
		"deploy",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Verify the start screen shows the default file (will be converted to absolute path)
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"The default blueprint file is:",
	)

	// Select the second option (Select a different file)
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	// Should now be at source selection
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Where is the blueprint file that you want to deploy stored?",
	)

	// Select local file
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Pick a blueprint file:",
	)

	// Select the valid blueprint file in the temp directory
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	blueprintPath := filepath.Join(s.tempDir, "project.blueprint.yml")
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		fmt.Sprintf("Blueprint file selected: %s", blueprintPath),
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	s.Assert().Equal(blueprintPath, finalModel.SelectedFile())
	s.Assert().Equal(consts.BlueprintSourceFile, finalModel.SelectedSource())
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_gcs_file() {
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_https_file() {
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_azure_blob_file() {
}

func TestSelectBlueprintSuite(t *testing.T) {
	suite.Run(t, new(SelectBlueprintSuite))
}
