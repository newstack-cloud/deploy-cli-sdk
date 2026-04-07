package preflight

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type RenderInstallSummaryTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestRenderInstallSummaryTestSuite(t *testing.T) {
	suite.Run(t, new(RenderInstallSummaryTestSuite))
}

func (s *RenderInstallSummaryTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_multiple_plugins_shows_all_plugin_names() {
	plugins := []string{"aws-provider", "gcp-provider", "azure-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 3, "Restart your engine.", "deploy")

	s.Contains(result, "aws-provider")
	s.Contains(result, "gcp-provider")
	s.Contains(result, "azure-provider")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_multiple_plugins_shows_installed_count() {
	plugins := []string{"aws-provider", "gcp-provider", "azure-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 3, "Restart your engine.", "deploy")

	s.Contains(result, "3")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_multiple_plugins_shows_restart_instructions() {
	plugins := []string{"aws-provider", "gcp-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 2, "Run `engine restart` to continue.", "deploy")

	s.Contains(result, "Run `engine restart` to continue.")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_multiple_plugins_shows_command_name_suggestion() {
	plugins := []string{"aws-provider", "gcp-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 2, "Restart your engine.", "bluelink deploy")

	s.Contains(result, "bluelink deploy")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_single_plugin_shows_plugin_name() {
	plugins := []string{"aws-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 1, "Restart your engine.", "deploy")

	s.Contains(result, "aws-provider")
	s.Contains(result, "1")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_single_plugin_shows_restart_instructions() {
	plugins := []string{"aws-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 1, "Run `engine restart` to continue.", "deploy")

	s.Contains(result, "Run `engine restart` to continue.")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_empty_command_name_omits_rerun_instruction() {
	plugins := []string{"aws-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 1, "Restart your engine.", "")

	s.NotContains(result, "Re-run")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_with_command_name_includes_rerun_instruction() {
	plugins := []string{"aws-provider"}
	result := RenderInstallSummary(s.testStyles, plugins, 1, "Restart your engine.", "deploy")

	s.Contains(result, "Re-run")
	s.Contains(result, "deploy")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_empty_plugins_renders_no_bullet_points() {
	result := RenderInstallSummary(s.testStyles, []string{}, 0, "Restart your engine.", "deploy")

	s.NotContains(result, "•")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_empty_plugins_still_shows_installed_count() {
	result := RenderInstallSummary(s.testStyles, []string{}, 0, "Restart your engine.", "deploy")

	s.Contains(result, "0")
}

func (s *RenderInstallSummaryTestSuite) Test_RenderInstallSummary_empty_plugins_still_shows_restart_instructions() {
	result := RenderInstallSummary(s.testStyles, []string{}, 0, "Restart your engine.", "deploy")

	s.Contains(result, "Restart your engine.")
}
