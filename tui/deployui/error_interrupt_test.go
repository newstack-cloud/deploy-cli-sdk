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
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployErrorInterruptSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployErrorInterruptSuite(t *testing.T) {
	suite.Run(t, new(DeployErrorInterruptSuite))
}

func (s *DeployErrorInterruptSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// Runs a deployment that leaves one resource mid-create and one created, and
// then fails on the error channel after the given delay.
func (s *DeployErrorInterruptSuite) runFailingDeployment(
	streamErr error,
	streamErrDelay time.Duration,
) DeployModel {
	// The changes are what tell the model each resource is being created, the
	// same as a real deployment, and so what a create interrupted midway
	// should be reported as.
	changesetChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"done":     {AppliedResourceInfo: provider.ResourceInfo{ResourceName: "done"}},
			"inFlight": {AppliedResourceInfo: provider.ResourceInfo{ResourceName: "inFlight"}},
		},
	}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStreamError(
			[]*types.BlueprintInstanceEvent{
				resourceEvent("done", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
				resourceEvent("done", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
				resourceEvent("inFlight", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			},
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployFailed),
			streamErr,
			streamErrDelay,
		),
		Logger:           zap.NewNop(),
		BlueprintFile:    "test.blueprint.yaml",
		Styles:           s.styles,
		HeadlessWriter:   os.Stdout,
		ChangesetChanges: changesetChanges,
	})

	return s.runModelToError(model, streamErr)
}

// Runs the given model until it reports the given error, and returns the
// final model.
func (s *DeployErrorInterruptSuite) runModelToError(
	model DeployModel,
	streamErr error,
) DeployModel {
	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContains(s.T(), testModel.Output(), streamErr.Error())

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	return testModel.FinalModel(s.T()).(DeployModel)
}

// A deployment can end three ways: a finish event, an instance update carrying
// a failed status, or an error on the error channel. The first two already
// settled in-flight items; the third left them reading "creating", which
// describes work still in progress rather than work that was cut short.
func (s *DeployErrorInterruptSuite) Test_a_stream_error_interrupts_in_flight_items() {
	streamErr := errors.New("run error: layer not resolved")

	finalModel := s.runFailingDeployment(streamErr, 250*time.Millisecond)

	s.Require().Error(finalModel.Err())

	resources := finalModel.ResourcesByName()
	s.Require().Contains(resources, "inFlight")
	s.Require().Contains(resources, "done")

	s.Equal(
		core.ResourceStatusCreateInterrupted,
		resources["inFlight"].Status,
		"an item still creating when the deployment errored was interrupted, not progressing",
	)
	s.Equal(
		core.ResourceStatusCreated,
		resources["done"].Status,
		"an item that finished before the error keeps its result",
	)
}

// The error channel is polled in one second windows, and each window has to
// start the next one. An error arriving after a window has expired only gets
// reported if that happened.
func (s *DeployErrorInterruptSuite) Test_an_error_arriving_after_a_poll_window_is_still_reported() {
	streamErr := errors.New("run error: connection lost mid-deployment")

	finalModel := s.runFailingDeployment(streamErr, 1500*time.Millisecond)

	s.Require().Error(finalModel.Err())
	s.Contains(finalModel.Err().Error(), "connection lost mid-deployment")
	s.Equal(
		core.ResourceStatusCreateInterrupted,
		finalModel.ResourcesByName()["inFlight"].Status,
	)
}

// An item built from deploy events alone has no action, since only a change set
// carries one. The status it last reported is what says which phase it was in.
func (s *DeployErrorInterruptSuite) Test_an_interrupted_create_is_reported_as_such_without_a_changeset() {
	streamErr := errors.New("run error: no changeset actions available")

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStreamError(
			[]*types.BlueprintInstanceEvent{
				resourceEvent("inFlight", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			},
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployFailed),
			streamErr,
			250*time.Millisecond,
		),
		Logger:         zap.NewNop(),
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
	})

	finalModel := s.runModelToError(model, streamErr)

	s.Equal(
		core.ResourceStatusCreateInterrupted,
		finalModel.ResourcesByName()["inFlight"].Status,
		"a resource reported as creating was interrupted mid-create, whatever the action says",
	)
}

// A resource with no changes is normally left alone, but one this run reported
// work on has to be settled like any other, it has an in-progress status and
// no action to explain it, so nothing else would ever move it off "creating".
func (s *DeployErrorInterruptSuite) Test_a_reported_no_change_resource_is_still_interrupted() {
	streamErr := errors.New("run error: unchanged resource left in progress")

	// "unchanged" is in the instance state but not the change set, which is
	// what makes it a no-change item.
	instanceState := &state.InstanceState{
		InstanceID:  "test-instance-id",
		Status:      core.InstanceStatusDeployFailed,
		ResourceIDs: map[string]string{"unchanged": "res-unchanged"},
		Resources: map[string]*state.ResourceState{
			"res-unchanged": {
				ResourceID: "res-unchanged",
				Name:       "unchanged",
				Type:       "test/resource",
			},
		},
	}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStreamError(
			[]*types.BlueprintInstanceEvent{
				reportedResourceEvent(
					"unchanged",
					core.ResourceStatusCreating,
					core.PreciseResourceStatusCreating,
				),
			},
			"test-instance-id",
			instanceState,
			streamErr,
			250*time.Millisecond,
		),
		Logger:         zap.NewNop(),
		InstanceID:     "test-instance-id",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
		ChangesetChanges: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"other": {AppliedResourceInfo: provider.ResourceInfo{ResourceName: "other"}},
			},
		},
	})

	finalModel := s.runModelToError(model, streamErr)

	unchanged := finalModel.ResourcesByName()["unchanged"]
	s.Require().NotNil(unchanged)
	s.Require().Equal(ActionNoChange, unchanged.Action, "the fixture must give it no changes")
	s.Equal(
		core.ResourceStatusCreateInterrupted,
		unchanged.Status,
		"a no-change resource the run reported as creating must not be left there",
	)
}

// Carries a timestamp, which is how the model knows this run reported the item.
func reportedResourceEvent(
	name string,
	status core.ResourceStatus,
	preciseStatus core.PreciseResourceStatus,
) *types.BlueprintInstanceEvent {
	event := resourceEvent(name, status, preciseStatus)
	event.DeployEvent.ResourceUpdateEvent.UpdateTimestamp = time.Now().Unix()
	return event
}
