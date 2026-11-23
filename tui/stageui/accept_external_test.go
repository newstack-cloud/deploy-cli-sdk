package stageui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type StageAcceptExternalSuite struct {
	suite.Suite
}

func TestStageAcceptExternalSuite(t *testing.T) {
	suite.Run(t, new(StageAcceptExternalSuite))
}

func (s *StageAcceptExternalSuite) Test_accepting_drift_sends_the_operation_config() {
	driftEvent := &types.ChangeStagingEvent{
		DriftDetected: &types.DriftDetectedEventData{
			ReconciliationResult: &container.ReconciliationCheckResult{
				InstanceID: "test-instance-id",
				Resources: []container.ResourceReconcileResult{
					{
						ResourceID:        "drifted-resource-id",
						ResourceName:      "drifted-resource",
						ResourceType:      "aws/sqs/queue",
						Type:              container.ReconciliationTypeDrift,
						RecommendedAction: container.ReconciliationActionAcceptExternal,
					},
				},
				HasDrift: true,
			},
			Message: "Drift detected in resource state",
		},
	}
	region := "eu-west-2"
	operationConfig := &types.BlueprintOperationConfig{
		ContextVariables: map[string]*core.ScalarValue{"region": {StringValue: &region}},
	}
	deployEngine := testutils.NewTestDeployEngineWithStaging(
		[]*types.ChangeStagingEvent{driftEvent},
		"test-changeset-drift",
	)

	model := NewStageModel(StageModelConfig{
		DeployEngine:    deployEngine,
		Logger:          zap.NewNop(),
		InstanceID:      "test-instance-id",
		InstanceName:    "test-instance",
		Styles:          stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter:  os.Stdout,
		OperationConfig: operationConfig,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "Drift Detected", "drifted-resource")

	testModel.Type("a")
	s.Eventually(func() bool {
		return testutils.LastApplyReconciliationPayloadFromEngine(deployEngine) != nil
	}, 5*time.Second, 20*time.Millisecond)

	payload := testutils.LastApplyReconciliationPayloadFromEngine(deployEngine)
	s.Equal(operationConfig, payload.Config)
	s.Len(payload.ResourceActions, 1)

	s.Require().NoError(testModel.Quit())
}
