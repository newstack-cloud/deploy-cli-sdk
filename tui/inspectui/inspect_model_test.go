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

type InspectModelBehaviorTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestInspectModelBehaviorTestSuite(t *testing.T) {
	suite.Run(t, new(InspectModelBehaviorTestSuite))
}

func (s *InspectModelBehaviorTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// helper: create a minimal InspectModel for TUI tests
func (s *InspectModelBehaviorTestSuite) newModel(
	instanceState *state.InstanceState,
	events []*types.BlueprintInstanceEvent,
) InspectModel {
	return *NewInspectModel(InspectModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineForInspect(instanceState, events),
		Logger:         zap.NewNop(),
		InstanceID:     instanceState.InstanceID,
		InstanceName:   instanceState.InstanceName,
		Styles:         s.styles,
		IsHeadless:     false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
}

// helper: create a headless InspectModel that writes to the provided buffer
func (s *InspectModelBehaviorTestSuite) newHeadlessModel(
	instanceState *state.InstanceState,
	events []*types.BlueprintInstanceEvent,
	output *bytes.Buffer,
) InspectModel {
	return *NewInspectModel(InspectModelConfig{
		DeployEngine:   testutils.NewTestDeployEngineForInspect(instanceState, events),
		Logger:         zap.NewNop(),
		InstanceID:     instanceState.InstanceID,
		InstanceName:   instanceState.InstanceName,
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: output,
		JSONMode:       false,
	})
}

func (s *InspectModelBehaviorTestSuite) Test_handleStreamClosed_sets_error_when_not_finished() {
	headlessOutput := &bytes.Buffer{}
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeploying,
	}
	// Use headless mode so the model calls tea.Quit when the stream closes with an error
	model := s.newHeadlessModel(instanceState, nil, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Do NOT send InstanceStateFetchedMsg so finished=false when the stream closes
	testModel.Send(InspectStreamClosedMsg{})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.NotNil(finalModel.Err())
	s.Contains(finalModel.Err().Error(), "closed")
}

func (s *InspectModelBehaviorTestSuite) Test_handleStreamClosed_no_error_when_already_finished() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	// Wait for content to appear (model now in finished state)
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "test")

	// Stream closed after already finished — should NOT set an error
	testModel.Send(InspectStreamClosedMsg{})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Nil(finalModel.Err())
}

func (s *InspectModelBehaviorTestSuite) Test_handleInspectError_propagates_error() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeploying,
	}
	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InspectErrorMsg{Err: errors.New("something went wrong")})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.NotNil(finalModel.Err())
	s.Contains(finalModel.Err().Error(), "something went wrong")
}

func (s *InspectModelBehaviorTestSuite) Test_handleInspectError_nil_error_is_ignored() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InspectErrorMsg{Err: nil})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Nil(finalModel.Err())
}

func (s *InspectModelBehaviorTestSuite) Test_overview_shows_child_and_link_names() {
	prepareDuration := float64(1000)
	totalDuration := float64(5000)
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"child-alpha": {
				InstanceID:   "child-id-1",
				InstanceName: "child-alpha",
				Status:       core.InstanceStatusDeployed,
			},
		},
		Links: map[string]*state.LinkState{
			"res-a::res-b": {
				LinkID: "link-1",
				Status: core.LinkStatusCreated,
			},
		},
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &prepareDuration,
			TotalDuration:   &totalDuration,
		},
	}

	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	// Wait for main view to render before pressing "o"
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "child-alpha")

	// Open overview with "o"
	testutils.Key(testModel, "o")

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-alpha",
		"res-a::res-b",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectModelBehaviorTestSuite) Test_exports_view_opens_and_closes_with_escape() {
	exportVal := "https://example.com"
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"api_url": {
				Value: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &exportVal},
				},
				Type:  "string",
				Field: "url",
			},
		},
	}

	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	// Wait for main view
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "test-instance")

	// Open exports with "e"
	testutils.Key(testModel, "e")

	// Exports view should include export key
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "api_url")

	// Close exports with "esc" and quit
	testutils.KeyEscape(testModel)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Nil(finalModel.Err())
	// showingExports should be closed after esc
	s.False(finalModel.IsInSubView())
}

func (s *InspectModelBehaviorTestSuite) Test_exports_view_opens_and_closes_with_e_key() {
	exportVal := "my-export-value"
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"my_export": {
				Value: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &exportVal},
				},
				Type:  "string",
				Field: "value",
			},
		},
	}

	model := s.newModel(instanceState, nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "test-instance")

	// Open with "e", close with "e" again
	testutils.Key(testModel, "e")
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "my_export")
	testutils.Key(testModel, "e")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Nil(finalModel.Err())
}

func (s *InspectModelBehaviorTestSuite) Test_GetError_returns_nil_initially() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	model := s.newModel(instanceState, nil)
	s.Nil(model.GetError())
}

func (s *InspectModelBehaviorTestSuite) Test_IsFinished_returns_false_initially() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	model := s.newModel(instanceState, nil)
	s.False(model.IsFinished())
}


func (s *InspectModelBehaviorTestSuite) Test_IsInSubView_returns_false_when_no_subview_active() {
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	model := s.newModel(instanceState, nil)
	s.False(model.IsInSubView())
}

func (s *InspectModelBehaviorTestSuite) Test_headless_durations_printed_when_set() {
	headlessOutput := &bytes.Buffer{}

	prepareDuration := float64(3000)
	totalDuration := float64(12000)
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &prepareDuration,
			TotalDuration:   &totalDuration,
		},
	}

	model := s.newHeadlessModel(instanceState, nil, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Prepare Duration")
	s.Contains(output, "Total Duration")
}

func (s *InspectModelBehaviorTestSuite) Test_headless_child_event_printed_during_streaming() {
	headlessOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeploying,
	}
	events := []*types.BlueprintInstanceEvent{
		{
			DeployEvent: container.DeployEvent{
				ChildUpdateEvent: &container.ChildDeployUpdateMessage{
					ChildName: "child-blueprint-one",
					Status:    core.InstanceStatusDeploying,
				},
			},
		},
		{
			DeployEvent: container.DeployEvent{
				ChildUpdateEvent: &container.ChildDeployUpdateMessage{
					ChildName: "child-blueprint-one",
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

	model := s.newHeadlessModel(instanceState, events, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "child-blueprint-one")
}

func (s *InspectModelBehaviorTestSuite) Test_headless_link_event_printed_during_streaming() {
	headlessOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeploying,
	}
	events := []*types.BlueprintInstanceEvent{
		{
			DeployEvent: container.DeployEvent{
				LinkUpdateEvent: &container.LinkDeployUpdateMessage{
					LinkName:      "service-a::service-b",
					Status:        core.LinkStatusCreating,
					PreciseStatus: core.PreciseLinkStatusUpdatingResourceA,
				},
			},
		},
		{
			DeployEvent: container.DeployEvent{
				LinkUpdateEvent: &container.LinkDeployUpdateMessage{
					LinkName:      "service-a::service-b",
					Status:        core.LinkStatusCreated,
					PreciseStatus: core.PreciseLinkStatusResourceBUpdated,
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

	model := s.newHeadlessModel(instanceState, events, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "service-a::service-b")
}

func (s *InspectModelBehaviorTestSuite) Test_headless_spec_field_single_line_rendered() {
	headlessOutput := &bytes.Buffer{}

	inputVal := "single-line-value"
	outputVal := "computed-output"
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "myResource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"bucketName": {Scalar: &core.ScalarValue{StringValue: &inputVal}},
						"outputKey":  {Scalar: &core.ScalarValue{StringValue: &outputVal}},
					},
				},
				ComputedFields: []string{"outputKey"},
			},
		},
	}

	model := s.newHeadlessModel(instanceState, nil, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "bucketName")
	s.Contains(output, "single-line-value")
}

func (s *InspectModelBehaviorTestSuite) Test_headless_spec_field_multi_line_rendered() {
	headlessOutput := &bytes.Buffer{}

	// A nested MappingNode with multiple fields renders as multi-line JSON via
	// FormatMappingNodeWithOptions(PrettyPrint=true), which triggers the multi-line
	// branch in printHeadlessSpecField (strings.ContainsRune(field.Value, '\n')).
	nestedValA := "nested-a"
	nestedValB := "nested-b"
	instanceState := &state.InstanceState{
		InstanceID:   "test-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "myResource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						// A nested object renders as multi-line JSON
						"tags": {
							Fields: map[string]*core.MappingNode{
								"keyA": {Scalar: &core.ScalarValue{StringValue: &nestedValA}},
								"keyB": {Scalar: &core.ScalarValue{StringValue: &nestedValB}},
							},
						},
					},
				},
				ComputedFields: []string{},
			},
		},
	}

	model := s.newHeadlessModel(instanceState, nil, headlessOutput)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	// The field name "tags" and at least one nested key value should appear
	s.Contains(output, "tags")
}
