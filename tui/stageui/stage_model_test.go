package stageui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type StageModelAccessorsSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestStageModelAccessorsSuite(t *testing.T) {
	suite.Run(t, new(StageModelAccessorsSuite))
}

func (s *StageModelAccessorsSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *StageModelAccessorsSuite) newModel() StageModel {
	return NewStageModel(StageModelConfig{
		DeployEngine: nil,
		Logger:       zap.NewNop(),
		Styles:       s.styles,
	})
}

func (s *StageModelAccessorsSuite) Test_CountChangeSummary_returns_zeros_for_empty_items() {
	m := s.newModel()
	create, update, del, recreate, retain := m.CountChangeSummary()
	s.Equal(0, create)
	s.Equal(0, update)
	s.Equal(0, del)
	s.Equal(0, recreate)
	s.Equal(0, retain)
}

func (s *StageModelAccessorsSuite) Test_CountChangeSummary_counts_all_action_types() {
	m := s.newModel()
	m.items = []StageItem{
		{Action: ActionCreate},
		{Action: ActionCreate},
		{Action: ActionUpdate},
		{Action: ActionDelete},
		{Action: ActionRecreate},
		{Action: ActionRetain},
		{Action: ActionRetain},
		{Action: ActionNoChange},
	}
	create, update, del, recreate, retain := m.CountChangeSummary()
	s.Equal(2, create)
	s.Equal(1, update)
	s.Equal(1, del)
	s.Equal(1, recreate)
	s.Equal(2, retain)
}

func (s *StageModelAccessorsSuite) Test_populateItemsFromCompleteChanges_routes_retained_resources_to_retain() {
	m := s.newModel()
	m.populateItemsFromCompleteChanges(&changes.BlueprintChanges{
		RemovedResources:  []string{"ordersQueue"},
		RetainedResources: []string{"ordersTable"},
	}, nil)

	var deleted, retained *StageItem
	for i := range m.items {
		switch m.items[i].Name {
		case "ordersQueue":
			deleted = &m.items[i]
		case "ordersTable":
			retained = &m.items[i]
		}
	}
	s.Require().NotNil(deleted)
	s.Require().NotNil(retained)

	s.Equal(ActionDelete, deleted.Action)
	s.True(deleted.Removed)
	s.False(deleted.Retained)

	s.Equal(ActionRetain, retained.Action)
	s.True(retained.Removed)
	s.True(retained.Retained)
}

func (s *StageModelAccessorsSuite) Test_GetChanges_returns_complete_changes() {
	m := s.newModel()
	bc := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"res1": {},
		},
	}
	m.completeChanges = bc
	s.Equal(bc, m.GetChanges())
}

func (s *StageModelAccessorsSuite) Test_StageExportsInstanceItem_implements_splitpane_Item() {
	var _ splitpane.Item = (*StageExportsInstanceItem)(nil)
}
