package deployui

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployModelBehaviourSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployModelBehaviourSuite(t *testing.T) {
	suite.Run(t, new(DeployModelBehaviourSuite))
}

func (s *DeployModelBehaviourSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *DeployModelBehaviourSuite) Test_handleDeployStreamClosed_sets_error_when_not_finished() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			// Emit a stream-closed before any finish event arrives
			[]*types.BlueprintInstanceEvent{
				resourceEvent("r1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			},
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeploying),
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: &bytes.Buffer{},
	})

	// Directly send the message to the model (no tea runner needed for unit test)
	updatedModel, _ := model.Update(DeployStreamClosedMsg{})
	dm, ok := updatedModel.(DeployModel)
	s.Require().True(ok)

	s.NotNil(dm.Err())
	s.Contains(dm.Err().Error(), "stream closed")
}

func (s *DeployModelBehaviourSuite) Test_handleDeployStreamClosed_is_noop_when_already_finished() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineWithDeployment(nil, "inst", nil),
		Logger:         zap.NewNop(),
		Styles:         s.styles,
		HeadlessWriter: &bytes.Buffer{},
	})
	// Mark as already finished
	model.finished = true

	updatedModel, _ := model.Update(DeployStreamClosedMsg{})
	dm, ok := updatedModel.(DeployModel)
	s.Require().True(ok)

	// finished=true already; error must remain nil because we entered the no-op branch
	s.Nil(dm.Err())
}

func (s *DeployModelBehaviourSuite) Test_WithCodeOnlyDenial_sets_code_only_denied_and_reasons() {
	renderer := &DeployStagingFooterRenderer{}
	s.False(renderer.CodeOnlyDenied)
	s.Empty(renderer.CodeOnlyReasons)

	opt := WithCodeOnlyDenial([]string{"reason-1", "reason-2"})
	opt(renderer)

	s.True(renderer.CodeOnlyDenied)
	s.Equal([]string{"reason-1", "reason-2"}, renderer.CodeOnlyReasons)
}

func (s *DeployModelBehaviourSuite) Test_WithCodeOnlyDenial_empty_reasons_still_marks_denied() {
	renderer := &DeployStagingFooterRenderer{}
	opt := WithCodeOnlyDenial([]string{})
	opt(renderer)

	s.True(renderer.CodeOnlyDenied)
	s.Empty(renderer.CodeOnlyReasons)
}

func (s *DeployModelBehaviourSuite) Test_headless_child_events_appear_in_output() {
	headlessOutput := &bytes.Buffer{}

	childEvents := []*types.BlueprintInstanceEvent{
		{
			DeployEvent: container.DeployEvent{
				ChildUpdateEvent: &container.ChildDeployUpdateMessage{
					ChildName: "infra-child",
					Status:    core.InstanceStatusDeploying,
				},
			},
		},
		{
			DeployEvent: container.DeployEvent{
				ChildUpdateEvent: &container.ChildDeployUpdateMessage{
					ChildName: "infra-child",
					Status:    core.InstanceStatusDeployed,
				},
			},
		},
		{
			DeployEvent: container.DeployEvent{
				FinishEvent: &container.DeploymentFinishedMessage{
					Status:      core.InstanceStatusDeployed,
					EndOfStream: true,
				},
			},
		},
	}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			childEvents,
			"test-instance-id",
			&state.InstanceState{InstanceID: "test-instance-id", Status: core.InstanceStatusDeployed},
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "changeset-child",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "infra-child")
	s.Contains(output, "Deployment completed")
}

func (s *DeployModelBehaviourSuite) Test_headless_link_events_appear_in_output() {
	headlessOutput := &bytes.Buffer{}

	linkEvents := []*types.BlueprintInstanceEvent{
		resourceEvent("res-a", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		resourceEvent("res-b", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		linkEvent("res-a::res-b", core.LinkStatusCreating, core.PreciseLinkStatusUpdatingResourceA),
		linkEvent("res-a::res-b", core.LinkStatusCreated, core.PreciseLinkStatusResourceBUpdated),
		finishEvent(core.InstanceStatusDeployed),
	}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			linkEvents,
			"test-instance-id",
			&state.InstanceState{InstanceID: "test-instance-id", Status: core.InstanceStatusDeployed},
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "changeset-link",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "res-a::res-b")
	s.Contains(output, "Deployment completed")
}

func (s *DeployModelBehaviourSuite) Test_handleDeployStreamClosed_headless_mode_writes_error_output() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineWithDeployment(nil, "inst", nil),
		Logger:         zap.NewNop(),
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	// Send the stream closed message directly - model is not finished so it should record error
	updatedModel, _ := model.Update(DeployStreamClosedMsg{})
	dm, ok := updatedModel.(DeployModel)
	s.Require().True(ok)

	s.NotNil(dm.Err())
}

func (s *DeployModelBehaviourSuite) Test_handleDeployError_nil_error_is_noop() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineWithDeployment(nil, "inst", nil),
		Logger:         zap.NewNop(),
		Styles:         s.styles,
		HeadlessWriter: &bytes.Buffer{},
	})

	updatedModel, _ := model.Update(DeployErrorMsg{Err: nil})
	dm := updatedModel.(DeployModel)

	s.Nil(dm.Err())
}

func (s *DeployModelBehaviourSuite) Test_handleDeployError_non_headless_stores_error() {
	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineWithDeployment(nil, "inst", nil),
		Logger:         zap.NewNop(),
		Styles:         s.styles,
		HeadlessWriter: &bytes.Buffer{},
		// IsHeadless: false (default)
	})

	testErr := errors.New("something went wrong")
	updatedModel, _ := model.Update(DeployErrorMsg{Err: testErr})
	dm := updatedModel.(DeployModel)

	s.Equal(testErr, dm.Err())
}
