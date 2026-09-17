package deployui

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ChangesetFetchTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestChangesetFetchTestSuite(t *testing.T) {
	suite.Run(t, new(ChangesetFetchTestSuite))
}

func (s *ChangesetFetchTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// Drives the model through the bubbletea runtime the same way
// the deploy command does, and returns the final model once the deployment has
// reported completion.
func (s *ChangesetFetchTestSuite) runToCompletion(model DeployModel) DeployModel {
	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContains(s.T(), testModel.Output(), "complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	return testModel.FinalModel(s.T()).(DeployModel)
}

// Drives the model until it reports an error, and returns the final model.
// The deployment never starts in this case, so there is no completion message
// to wait for.
func (s *ChangesetFetchTestSuite) runToError(model DeployModel, errText string) DeployModel {
	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContains(s.T(), testModel.Output(), errText)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	return testModel.FinalModel(s.T()).(DeployModel)
}

func (s *ChangesetFetchTestSuite) assertGroupedByChangeset(items []DeployItem) {
	s.Require().Len(items, 2)

	for _, item := range items {
		group := item.GetResourceGroup()
		s.Require().NotNilf(group, "%s is ungrouped", item.GetID())
		s.Equal("orderHandler", group.GroupName)
		s.Equal("celerity/handler", group.GroupType)
	}
}

func (s *ChangesetFetchTestSuite) Test_existing_changeset_changes_are_fetched_and_group_resources() {
	// Deploying an existing change set skips staging, so nothing has supplied
	// the changes and they have to come from the engine.
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeploymentAndChangeset(
			// Only a finish event, so any grouping can only have come from the
			// fetched change set and not from deploy events.
			[]*types.BlueprintInstanceEvent{finishEvent(core.InstanceStatusDeployed)},
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
			groupedChangeset(),
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "existing-changeset",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
		// Staging didn't run in the current session.
		ChangesetChanges: nil,
	})
	s.Empty(model.Items(), "no items can exist before the changes are known")

	finalModel := s.runToCompletion(model)

	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
	s.assertGroupedByChangeset(finalModel.Items())
}

func (s *ChangesetFetchTestSuite) Test_staged_deployments_use_the_supplied_changes() {
	// The staged flow supplies the changes through the model config, and the
	// engine offers no change set of its own, so the items can only have been
	// built from what was supplied.
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			[]*types.BlueprintInstanceEvent{finishEvent(core.InstanceStatusDeployed)},
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "already-staged",
		BlueprintFile:    "test.blueprint.yaml",
		Styles:           s.styles,
		HeadlessWriter:   os.Stdout,
		ChangesetChanges: groupedChangeset(),
	})

	finalModel := s.runToCompletion(model)

	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
	s.assertGroupedByChangeset(finalModel.Items())
}

func (s *ChangesetFetchTestSuite) Test_no_changeset_id_builds_items_from_events() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		Logger:         zap.NewNop(),
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
	})

	finalModel := s.runToCompletion(model)

	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
	s.Contains(finalModel.ResourcesByName(), "test-resource")
}

func (s *ChangesetFetchTestSuite) Test_a_changeset_that_cannot_be_found_fails_the_deployment() {
	// Staging is the only thing that produces a change set, so a change set
	// that is not there cannot be rebuilt here and the deployment must stop.
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithMissingChangeset(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "missing-changeset",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
	})

	finalModel := s.runToError(model, "could not be found")

	s.Require().Error(finalModel.Err())
	s.Contains(finalModel.Err().Error(), "missing-changeset")
	s.Empty(finalModel.ResourcesByName(), "the deployment must not have started")
}

func (s *ChangesetFetchTestSuite) Test_a_failed_changeset_fetch_fails_the_deployment() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithChangesetFetchError(
			errors.New("connection refused"),
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "existing-changeset",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
	})

	finalModel := s.runToError(model, "failed to load change set")

	s.Require().Error(finalModel.Err())
	s.Contains(finalModel.Err().Error(), "connection refused")
	s.Empty(finalModel.ResourcesByName(), "the deployment must not have started")
}

func groupedChangeset() *changes.BlueprintChanges {
	newResources := map[string]provider.Changes{}
	for _, name := range []string{"orderHandler_lambda", "orderHandler_role"} {
		newResources[name] = provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceName: name,
				ResourceWithResolvedSubs: &provider.ResolvedResource{
					Metadata: &provider.ResolvedResourceMetadata{
						Annotations: core.MappingNodeFromStringMap(map[string]string{
							shared.AnnotationSourceAbstractName: "orderHandler",
							shared.AnnotationSourceAbstractType: "celerity/handler",
						}),
					},
				},
			},
		}
	}
	return &changes.BlueprintChanges{NewResources: newResources}
}
