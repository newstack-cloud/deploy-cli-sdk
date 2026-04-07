package inspectui

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

type MainModelTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestMainModelTestSuite(t *testing.T) {
	suite.Run(t, new(MainModelTestSuite))
}

func (s *MainModelTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// helper: build a MainModel via NewInspectApp for TUI tests.
func (s *MainModelTestSuite) newApp(
	instanceState *state.InstanceState,
	events []*types.BlueprintInstanceEvent,
) *MainModel {
	app, err := NewInspectApp(InspectAppConfig{
		DeployEngine:   testutils.NewTestDeployEngineForInspect(instanceState, events),
		Logger:         zap.NewNop(),
		InstanceID:     instanceState.InstanceID,
		InstanceName:   instanceState.InstanceName,
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
	s.Require().NoError(err)
	return app
}

// helper: build a headless MainModel.
func (s *MainModelTestSuite) newHeadlessApp(
	instanceState *state.InstanceState,
	events []*types.BlueprintInstanceEvent,
	output *bytes.Buffer,
) *MainModel {
	app, err := NewInspectApp(InspectAppConfig{
		DeployEngine:   testutils.NewTestDeployEngineForInspect(instanceState, events),
		Logger:         zap.NewNop(),
		InstanceID:     instanceState.InstanceID,
		InstanceName:   instanceState.InstanceName,
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: output,
		JSONMode:       false,
	})
	s.Require().NoError(err)
	return app
}

func (s *MainModelTestSuite) Test_NewInspectApp_returns_model_without_error() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	app := s.newApp(instanceState, nil)
	s.NotNil(app)
}


func (s *MainModelTestSuite) Test_MainModel_static_instance_state_renders_resources() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}
	app := s.newApp(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "myBucket")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *MainModelTestSuite) Test_MainModel_headless_static_outputs_instance_state() {
	output := &bytes.Buffer{}
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
	}
	app := s.newHeadlessApp(instanceState, nil, output)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	s.Contains(output.String(), "test-id")
}

func (s *MainModelTestSuite) Test_MainModel_headless_streaming_outputs_resource_events() {
	output := &bytes.Buffer{}
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeploying,
	}
	events := []*types.BlueprintInstanceEvent{
		{
			DeployEvent: container.DeployEvent{
				ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
					ResourceName:  "streaming-resource",
					ResourceID:    "res-streaming",
					Status:        core.ResourceStatusCreated,
					PreciseStatus: core.PreciseResourceStatusCreated,
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
	app := s.newHeadlessApp(instanceState, events, output)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	s.Contains(output.String(), "streaming-resource")
}

func (s *MainModelTestSuite) Test_MainModel_instance_not_found_sets_error() {
	instanceState := &state.InstanceState{
		InstanceID:   "missing-id",
		InstanceName: "",
		Status:       core.InstanceStatusDeployed,
	}
	app := s.newApp(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	notFoundErr := errInstanceNotFound("missing-id", "")
	testModel.Send(InstanceNotFoundMsg{Err: notFoundErr})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "not found")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

func (s *MainModelTestSuite) Test_MainModel_headless_instance_not_found_outputs_error() {
	output := &bytes.Buffer{}
	instanceState := &state.InstanceState{
		InstanceID:   "missing-id",
		InstanceName: "",
		Status:       core.InstanceStatusDeployed,
	}
	app := s.newHeadlessApp(instanceState, nil, output)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceNotFoundMsg{Err: errInstanceNotFound("missing-id", "")})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	s.Contains(output.String(), "not found")
}

func (s *MainModelTestSuite) Test_MainModel_inspect_error_propagates_to_main() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	app := s.newApp(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InspectErrorMsg{Err: errors.New("upstream error")})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
	s.Contains(finalModel.Error.Error(), "upstream error")
}

func (s *MainModelTestSuite) Test_MainModel_headless_stream_closed_unexpectedly_sets_error() {
	output := &bytes.Buffer{}
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeploying,
	}
	app := s.newHeadlessApp(instanceState, nil, output)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	// Close stream without finishing — should set an error and quit
	testModel.Send(InspectStreamClosedMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

func (s *MainModelTestSuite) Test_MainModel_state_refresh_hydrates_items() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "hydrated-resource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}
	app := s.newApp(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		*app,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "hydrated-resource")

	// Send a state refresh — should not crash or error
	testModel.Send(InstanceStateRefreshedMsg{InstanceState: instanceState})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}
