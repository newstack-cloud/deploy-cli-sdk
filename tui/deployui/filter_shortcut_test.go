package deployui

import (
	"bytes"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type FilterShortcutTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestFilterShortcutTestSuite(t *testing.T) {
	suite.Run(t, new(FilterShortcutTestSuite))
}

func (s *FilterShortcutTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// Returns a deploy model with every shortcut live, the deployment
// is finished and instance state is available.
func (s *FilterShortcutTestSuite) finishedModel() tea.Model {
	model := NewDeployModel(DeployModelConfig{
		Logger:           zap.NewNop(),
		ChangesetID:      "cs",
		InstanceName:     "app",
		Styles:           s.styles,
		HeadlessWriter:   &bytes.Buffer{},
		ChangesetChanges: harnessChangeset(),
	})

	var m tea.Model = model
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 45})
	m, _ = m.Update(DeployEventMsg(*finishEvent(core.InstanceStatusDeployed)))
	m, _ = m.Update(PostDeployInstanceStateFetchedMsg{InstanceState: harnessInstanceState()})
	return m
}

func typeRunes(m tea.Model, runes string) tea.Model {
	for _, r := range runes {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func (s *FilterShortcutTestSuite) Test_shortcuts_type_into_the_term_instead_of_firing() {
	m := typeRunes(s.finishedModel(), "/")
	s.Require().True(m.(DeployModel).splitPane.IsFiltering())

	// Each of these opens an overlay when not filtering.
	m = typeRunes(m, "eos")

	deployModel := m.(DeployModel)
	s.Equal("eos", deployModel.splitPane.FilterTerm())
	s.False(deployModel.showingExportsView, "e opened the exports view")
	s.False(deployModel.showingOverview, "o opened the overview")
	s.False(deployModel.showingSpecView, "s opened the spec view")
}

func (s *FilterShortcutTestSuite) Test_r_does_not_open_the_pre_rollback_view() {
	m := typeRunes(s.finishedModel(), "/")
	m = typeRunes(m, "r")

	deployModel := m.(DeployModel)
	s.Equal("r", deployModel.splitPane.FilterTerm())
	s.False(deployModel.showingPreRollbackState)
}

func (s *FilterShortcutTestSuite) Test_shortcuts_work_again_once_the_term_is_committed() {
	m := typeRunes(s.finishedModel(), "/")
	m = typeRunes(m, "lambda")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	deployModel := m.(DeployModel)
	s.Require().False(deployModel.splitPane.IsFiltering())
	s.Require().Equal("lambda", deployModel.splitPane.FilterTerm(), "the term stays applied")

	// With the term committed, keys are shortcuts again.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	s.True(m.(DeployModel).showingOverview, "o should open the overview once committed")
}

func (s *FilterShortcutTestSuite) Test_the_exports_pane_captures_its_own_close_key() {
	m := s.finishedModel()

	// Open exports, then start filtering inside it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	s.Require().True(m.(DeployModel).showingExportsView)

	m = typeRunes(m, "/")
	s.Require().True(m.(DeployModel).exportsModel.IsFiltering())

	// "e" closes the exports view when not filtering.
	m = typeRunes(m, "e")
	s.True(m.(DeployModel).showingExportsView, "e closed the exports view while filtering")
}
