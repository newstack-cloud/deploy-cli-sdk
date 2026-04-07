package stageui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type StageTUISuite struct {
	suite.Suite
}

func TestStageTUISuite(t *testing.T) {
	suite.Run(t, new(StageTUISuite))
}

// --- Test Event Factories ---

// testStagingType determines the test scenario
type testStagingType string

const (
	stagingSuccessCreate   testStagingType = "success_create"
	stagingSuccessUpdate   testStagingType = "success_update"
	stagingSuccessDelete   testStagingType = "success_delete"
	stagingSuccessRecreate testStagingType = "success_recreate"
	stagingSuccessMixed    testStagingType = "success_mixed"
	stagingSuccessChild    testStagingType = "success_child"
	stagingSuccessLink     testStagingType = "success_link"
)

func testStagingEvents(stagingType testStagingType) []*types.ChangeStagingEvent {
	switch stagingType {
	case stagingSuccessCreate:
		return []*types.ChangeStagingEvent{
			resourceCreateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessUpdate:
		return []*types.ChangeStagingEvent{
			resourceUpdateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessDelete:
		return []*types.ChangeStagingEvent{
			resourceDeleteEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessRecreate:
		return []*types.ChangeStagingEvent{
			resourceRecreateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessMixed:
		return []*types.ChangeStagingEvent{
			resourceCreateEvent("resource-a"),
			resourceUpdateEvent("resource-b"),
			childChangesEvent("child-blueprint"),
			linkChangesEvent("resource-a", "resource-b"),
			completeChangesEvent(),
		}
	case stagingSuccessChild:
		return []*types.ChangeStagingEvent{
			childChangesEvent("child-blueprint"),
			completeChangesEvent(),
		}
	case stagingSuccessLink:
		return []*types.ChangeStagingEvent{
			linkChangesEvent("resource-a", "resource-b"),
			completeChangesEvent(),
		}
	default:
		return []*types.ChangeStagingEvent{completeChangesEvent()}
	}
}

func resourceCreateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          true,
				Changes: provider.Changes{
					NewFields: []provider.FieldChange{
						{FieldPath: "spec.field1", NewValue: stringMappingNode("value1")},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceUpdateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      false,
				Changes: provider.Changes{
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.replicas",
							PrevValue: intMappingNode(2),
							NewValue:  intMappingNode(4),
						},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceDeleteEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      true,
				Changes:      provider.Changes{},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceRecreateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      false,
				Changes: provider.Changes{
					MustRecreate: true,
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.immutableField",
							PrevValue: stringMappingNode("old"),
							NewValue:  stringMappingNode("new"),
						},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func childChangesEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ChildChanges: &types.ChildChangesEventData{
			ChildChangesMessage: container.ChildChangesMessage{
				ChildBlueprintName: name,
				New:                true,
				Changes: changes.BlueprintChanges{
					NewResources: map[string]provider.Changes{
						"child-resource-1": {},
						"child-resource-2": {},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func linkChangesEvent(resourceA, resourceB string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		LinkChanges: &types.LinkChangesEventData{
			LinkChangesMessage: container.LinkChangesMessage{
				ResourceAName: resourceA,
				ResourceBName: resourceB,
				New:           true,
				Changes: provider.LinkChanges{
					NewFields: []*provider.FieldChange{
						{FieldPath: "link.field1", NewValue: stringMappingNode("linked")},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func completeChangesEvent() *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		CompleteChanges: &types.CompleteChangesEventData{
			Changes: &changes.BlueprintChanges{},
		},
	}
}

// Helper functions for MappingNode creation
func stringMappingNode(s string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &s},
	}
}

func intMappingNode(i int) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{IntValue: &i},
	}
}

// --- Interactive Mode Tests ---

func (s *StageTUISuite) Test_successful_staging_with_resource_create() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-123",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Send blueprint selection to trigger staging
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"CREATE",
		"test-changeset-123",
		"bluelink deploy",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
	s.True(finalModel.Finished())
	s.Equal("test-changeset-123", finalModel.ChangesetID())
}

func (s *StageTUISuite) Test_successful_staging_with_resource_update() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessUpdate),
			"test-changeset-456",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"UPDATE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
}

func (s *StageTUISuite) Test_successful_staging_with_resource_delete() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessDelete),
			"test-changeset-789",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"DELETE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
}

func (s *StageTUISuite) Test_successful_staging_with_resource_recreate() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessRecreate),
			"test-changeset-recreate",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"RECREATE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
}

func (s *StageTUISuite) Test_successful_staging_with_child_blueprint() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessChild),
			"test-changeset-child",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
		"Child Blueprints",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
}

func (s *StageTUISuite) Test_successful_staging_with_link() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessLink),
			"test-changeset-link",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
		"Links",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
}

func (s *StageTUISuite) Test_successful_staging_with_mixed_items() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-mixed",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a",
		"resource-b",
		"child-blueprint",
		"resource-a::resource-b",
		"Resources",
		"Child Blueprints",
		"Links",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.Err())
	s.Len(finalModel.Items(), 4)
}

// --- Headless Mode Tests ---

func (s *StageTUISuite) Test_successful_staging_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-headless",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"[stage]",
		"test-resource",
		"CREATE",
		"test-changeset-headless",
		"bluelink deploy",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_progress() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-progress",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"✓ resource: resource-a - CREATE",
		"✓ resource: resource-b - UPDATE",
		"✓ child: child-blueprint - CREATE",
		"✓ link: resource-a::resource-b - CREATE",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_summary() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-summary",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Change staging complete",
		"2 resources",
		"1 child",
		"1 link",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_field_changes() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessUpdate),
			"test-changeset-fields",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"~",           // Modified field indicator
		"spec.replicas",
		"2",
		"4",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Error Handling Tests ---

func (s *StageTUISuite) Test_staging_validation_error() {
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Resource 'test-resource' is invalid",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{Position: source.Position{Line: 10, Column: 5}},
				},
			},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(validationErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Failed to create changeset",
		"Resource 'test-resource' is invalid",
		"[line 10, col 5]",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.NotNil(finalModel.Err())
}

func (s *StageTUISuite) Test_staging_validation_error_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Blueprint validation failed",
			},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(validationErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"[stage]",
		"Failed to create changeset",
		"Blueprint validation failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_generic_error() {
	genericErr := &engineerrors.ClientError{
		StatusCode: 500,
		Message:    "Internal server error",
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(genericErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Error during change staging",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.NotNil(finalModel.Err())
}

// --- Additional Headless Mode Tests ---

func (s *StageTUISuite) Test_staging_headless_with_existing_resource() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessUpdate),
			"test-changeset-existing",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	// Existing resource should not show "(new)"
	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"test-resource",
		"UPDATE",
	)
	s.NotContains(headlessOutput.String(), "(new)")

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_existing_link() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		{
			LinkChanges: &types.LinkChangesEventData{
				LinkChangesMessage: container.LinkChangesMessage{
					ResourceAName: "resA",
					ResourceBName: "resB",
					New:           false,
					Changes: provider.LinkChanges{
						ModifiedFields: []*provider.FieldChange{
							{FieldPath: "field1", PrevValue: stringMappingNode("old"), NewValue: stringMappingNode("new")},
						},
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-link-existing"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"resA::resB",
		"UPDATE",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_existing_child() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		{
			ChildChanges: &types.ChildChangesEventData{
				ChildChangesMessage: container.ChildChangesMessage{
					ChildBlueprintName: "existingChild",
					New:                false,
					Changes: changes.BlueprintChanges{
						ResourceChanges: map[string]provider.Changes{
							"res1": {ModifiedFields: []provider.FieldChange{{FieldPath: "spec.field"}}},
						},
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-child-existing"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"existingChild",
		"UPDATE",
		"1 resource",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_destroy_hint() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessDelete),
			"test-changeset-destroy",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Destroy:        true,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"bluelink destroy",
		"--changeset-id",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_instance_id() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-id",
		),
		Logger:         zap.NewNop(),
		InstanceID:     "inst-456",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"--instance-id inst-456",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_placeholder_name() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-placeholder",
		),
		Logger:         zap.NewNop(),
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"--instance-name <name>",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_no_changes() {
	headlessOutput := testutils.NewSaveBuffer()
	// Empty events - just complete changes with no actual changes
	events := []*types.ChangeStagingEvent{
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-no-changes"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"No changes to apply",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_stream_error() {
	headlessOutput := testutils.NewSaveBuffer()
	streamErr := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Stream error during staging",
			Diagnostics: []*core.Diagnostic{
				{Level: core.DiagnosticLevelError, Message: "Diagnostic message for stream error"},
			},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(streamErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Error during change staging",
		"Stream error during staging",
		"Diagnostics",
		"Diagnostic message for stream error",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_validation_errors() {
	headlessOutput := testutils.NewSaveBuffer()
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.myRes", Message: "invalid field value"},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(validationErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Failed to create changeset",
		"resources.myRes",
		"invalid field value",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_generic_error() {
	headlessOutput := testutils.NewSaveBuffer()
	genericErr := &engineerrors.ClientError{
		StatusCode: 500,
		Message:    "Internal server error",
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(genericErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Error during change staging",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_link_field_changes() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		{
			LinkChanges: &types.LinkChangesEventData{
				LinkChangesMessage: container.LinkChangesMessage{
					ResourceAName: "resA",
					ResourceBName: "resB",
					New:           true,
					Changes: provider.LinkChanges{
						NewFields: []*provider.FieldChange{
							{FieldPath: "link.field1", NewValue: stringMappingNode("linked-value")},
						},
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-link-fields"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"link.field1",
		"linked-value",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_child_change_counts() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		{
			ChildChanges: &types.ChildChangesEventData{
				ChildChangesMessage: container.ChildChangesMessage{
					ChildBlueprintName: "myChild",
					New:                true,
					Changes: changes.BlueprintChanges{
						NewResources:    map[string]provider.Changes{"res1": {}, "res2": {}},
						RemovedChildren: []string{}, // explicitly empty
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-child-counts"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"myChild",
		"(new, 2 resources)",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_resource_outbound_links() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		resourceCreateEvent("resource-a"),
		{
			LinkChanges: &types.LinkChangesEventData{
				LinkChangesMessage: container.LinkChangesMessage{
					ResourceAName: "resource-a",
					ResourceBName: "resource-b",
					New:           true,
					Changes: provider.LinkChanges{
						NewFields: []*provider.FieldChange{
							{FieldPath: "outbound.field", NewValue: stringMappingNode("outbound-value")},
						},
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
		completeChangesEvent(),
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-outbound"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"resource-a",
		"resource-a::resource-b",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_export_changes() {
	headlessOutput := testutils.NewSaveBuffer()
	events := []*types.ChangeStagingEvent{
		resourceCreateEvent("test-resource"),
		{
			CompleteChanges: &types.CompleteChangesEventData{
				Changes: &changes.BlueprintChanges{
					NewExports: map[string]provider.FieldChange{
						"myExport": {NewValue: stringMappingNode("exported-value")},
					},
				},
			},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(events, "test-changeset-exports"),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"exports",
		"New Exports",
		"myExport",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_with_diagnostic_location() {
	headlessOutput := testutils.NewSaveBuffer()
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Error with location",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{Position: source.Position{Line: 42, Column: 10}},
				},
			},
		},
	}

	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStagingError(validationErr),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"line 42",
		"col 10",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Navigation Tests ---

func (s *StageTUISuite) Test_quit_with_q() {
	model := NewStageModel(StageModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-quit",
		),
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	// Wait for staging to complete
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-quit",
	)

	// Press q to quit
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}
