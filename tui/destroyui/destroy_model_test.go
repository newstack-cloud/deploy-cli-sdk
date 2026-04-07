package destroyui

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	dectypes "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/driftui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type HeadlessAdditionalSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestHeadlessAdditionalSuite(t *testing.T) {
	suite.Run(t, new(HeadlessAdditionalSuite))
}

func (s *HeadlessAdditionalSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *HeadlessAdditionalSuite) Test_headless_child_events_appear_in_output() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithChild),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-child-headless",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "child-blueprint")
	s.Contains(output, "child")
	s.Contains(output, "Destroy completed")
}

func (s *HeadlessAdditionalSuite) Test_headless_link_events_appear_in_output() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithLink),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-link-headless",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "resource-a::resource-b")
	s.Contains(output, "link")
	s.Contains(output, "Destroy completed")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_hierarchy_resources() {
	headlessOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-hierarchy-resources",
		InstanceID:       "pre-destroy-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Pre-Destroy Instance State")
	s.Contains(output, "Resources:")
	s.Contains(output, "resource-1")
	s.Contains(output, "resource-2")
	s.Contains(output, "res-id-1")
	s.Contains(output, "res-id-2")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_hierarchy_links() {
	headlessOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-hierarchy-links",
		InstanceID:       "pre-destroy-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Links:")
	s.Contains(output, "resource-1::resource-2")
	s.Contains(output, "link-id-1")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_hierarchy_exports() {
	headlessOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-hierarchy-exports",
		InstanceID:       "pre-destroy-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Exports:")
	s.Contains(output, "export-1")
	s.Contains(output, "export-2")
	s.Contains(output, "export-value-1")
	s.Contains(output, "export-value-2")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_hierarchy_child_blueprints() {
	headlessOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-hierarchy-children",
		InstanceID:       "pre-destroy-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Child Blueprints:")
	s.Contains(output, "child-bp")
	s.Contains(output, "child-instance-id")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_hierarchy_spec_and_outputs() {
	headlessOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-hierarchy-spec",
		InstanceID:       "pre-destroy-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	// Spec fields (non-computed) should appear
	s.Contains(output, "bucketName")
	s.Contains(output, "my-bucket-name")
	// Computed/output fields should appear too
	s.Contains(output, "arn")
}

func (s *HeadlessAdditionalSuite) Test_stream_closed_sets_error_when_not_finished() {
	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-stream-closed",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       false,
		HeadlessWriter:   os.Stdout,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	// Directly send the stream closed message to the model without starting destroy
	// so finished remains false
	updatedModel, _ := model.Update(DestroyStreamClosedMsg{})
	resultModel := updatedModel.(DestroyModel)

	s.NotNil(resultModel.Err())
	s.Contains(resultModel.Err().Error(), "stream closed")
}

func (s *HeadlessAdditionalSuite) Test_stream_closed_does_not_overwrite_finished_state() {
	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-already-finished",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       false,
		HeadlessWriter:   os.Stdout,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	// Simulate already finished by setting the finish event first
	finishMsg := DestroyEventMsg(
		*finishEvent(core.InstanceStatusDestroyed),
	)
	model.Update(StartDestroyMsg{})
	afterFinish, _ := model.Update(finishMsg)

	// Now send stream closed - should not overwrite nil error since already finished
	finalResult, _ := afterFinish.(DestroyModel).Update(DestroyStreamClosedMsg{})
	finalModel := finalResult.(DestroyModel)

	// finished was set by finishMsg, so stream closed should be a no-op
	s.Nil(finalModel.Err())
}

func (s *HeadlessAdditionalSuite) Test_headless_stream_closed_outputs_error() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-stream-closed-headless",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(DestroyStreamClosedMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "ERR")
}

func (s *HeadlessAdditionalSuite) Test_headless_validation_error_output() {
	headlessOutput := &bytes.Buffer{}

	validationErr := &engineerrors.ClientError{
		StatusCode: 422,
		Message:    "Validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.myRes", Message: "invalid field value"},
		},
	}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine:    testutils.NewTestDeployEngineWithDestroyError(validationErr),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-validation-err",
		InstanceID:       "test-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "invalid field value")
}

func (s *HeadlessAdditionalSuite) Test_headless_stream_error_output() {
	headlessOutput := &bytes.Buffer{}

	streamErr := &engineerrors.StreamError{
		Event: &dectypes.StreamErrorMessageEvent{
			Message: "stream processing failed",
		},
	}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine:    testutils.NewTestDeployEngineWithDestroyError(streamErr),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-stream-err",
		InstanceID:       "test-instance-id",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "stream processing failed")
}

func (s *HeadlessAdditionalSuite) Test_reconciliation_error_sets_error() {
	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-reconcile-err",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       false,
		HeadlessWriter:   os.Stdout,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.driftReviewMode = true

	updatedModel, _ := model.Update(driftui.ReconciliationErrorMsg{Err: fmt.Errorf("reconciliation failed")})
	resultModel := updatedModel.(DestroyModel)

	s.NotNil(resultModel.Err())
	s.Contains(resultModel.Err().Error(), "reconciliation failed")
}

func (s *HeadlessAdditionalSuite) Test_headless_child_summary_in_destroyed_items() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithChild),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-child-summary",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	// Summary should include child in count
	s.Contains(output, "child")
}

func (s *HeadlessAdditionalSuite) Test_headless_link_summary_in_destroyed_items() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithLink),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-link-summary",
		InstanceID:       "",
		InstanceName:     "test-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	// Summary should include links
	s.Contains(output, "link")
	s.Contains(output, "resource-a::resource-b")
}

func (s *HeadlessAdditionalSuite) Test_headless_pre_destroy_state_with_minimal_instance_state() {
	headlessOutput := &bytes.Buffer{}

	// Minimal state with no resources, links, exports, or children
	minimalState := &state.InstanceState{
		InstanceID: "minimal-instance-id",
		Status:     core.InstanceStatusDeployed,
	}

	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine: testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		Logger:           zap.NewNop(),
		ChangesetID:      "test-changeset-minimal-state",
		InstanceID:       "minimal-instance-id",
		InstanceName:     "minimal-instance",
		Force:            false,
		Styles:           s.styles,
		IsHeadless:       true,
		HeadlessWriter:   headlessOutput,
		ChangesetChanges: nil,
		JSONMode:         false,
	})

	model.SetPreDestroyInstanceState(minimalState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Pre-Destroy Instance State")
	s.Contains(output, "minimal-instance-id")
	// Should not contain section headers when sections are empty
	s.NotContains(output, "Resources:")
	s.NotContains(output, "Links:")
	s.NotContains(output, "Exports:")
	s.NotContains(output, "Child Blueprints:")
}
