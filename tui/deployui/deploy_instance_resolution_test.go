package deployui

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployInstanceResolutionSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployInstanceResolutionSuite(t *testing.T) {
	suite.Run(t, new(DeployInstanceResolutionSuite))
}

func (s *DeployInstanceResolutionSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *DeployInstanceResolutionSuite) runHeadlessDeploy(
	deployEngine engine.DeployEngine,
	instanceName string,
) string {
	headlessOutput := &bytes.Buffer{}
	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   deployEngine,
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset",
		InstanceName:   instanceName,
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
	return headlessOutput.String()
}

func (s *DeployInstanceResolutionSuite) Test_deploy_by_name_updates_the_existing_instance() {
	deployEngine := testutils.NewTestDeployEngineWithDeployment(
		testDeployEvents(deploySuccessUpdate),
		"test-instance-id",
		testInstanceState(core.InstanceStatusDeployed),
	)

	output := s.runHeadlessDeploy(deployEngine, "test-instance")

	s.Contains(output, "Update completed")
	s.Equal([]string{"test-instance-id"}, testutils.InstanceUpdatesFromEngine(deployEngine))
	s.Equal(0, testutils.InstanceCreatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) Test_deploy_by_a_new_name_creates_the_instance() {
	deployEngine := testutils.NewTestDeployEngineWithUnknownInstanceName(
		testDeployEvents(deploySuccessCreate),
		"test-instance-id",
		testInstanceState(core.InstanceStatusDeployed),
		"new-instance",
	)

	output := s.runHeadlessDeploy(deployEngine, "new-instance")

	s.Contains(output, "Deployment completed")
	s.Equal(1, testutils.InstanceCreatesFromEngine(deployEngine))
	s.Empty(testutils.InstanceUpdatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) Test_deploy_reports_a_failed_instance_lookup_without_creating() {
	deployEngine := testutils.NewTestDeployEngineWithInstanceLookupError(errors.New("connection refused"))

	output := s.runHeadlessDeploy(deployEngine, "test-instance")

	s.Contains(output, `looking up instance "test-instance"`)
	s.Equal(0, testutils.InstanceCreatesFromEngine(deployEngine))
	s.Empty(testutils.InstanceUpdatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) runHeadlessStagedDeploy(deployEngine engine.DeployEngine) string {
	headlessOutput := &bytes.Buffer{}
	app, err := NewDeployApp(DeployAppConfig{
		DeployEngine:   deployEngine,
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		StageFirst:     true,
		AutoApprove:    true,
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), app, teatest.WithInitialTermSize(300, 100))
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
	return headlessOutput.String()
}

func (s *DeployInstanceResolutionSuite) Test_staged_deploy_by_name_stages_and_updates_the_existing_instance() {
	deployEngine := testutils.NewTestDeployEngineWithStagingAndDeployment(
		testStagingEventsForDeploy(stagingSuccessUpdateDeploy),
		"test-changeset",
		testDeployEvents(deploySuccessUpdate),
		testInstanceState(core.InstanceStatusDeployed),
	)

	output := s.runHeadlessStagedDeploy(deployEngine)

	s.Contains(output, "Update completed")
	s.Equal("test-instance-id", testutils.LastChangesetPayloadFromEngine(deployEngine).InstanceID)
	s.Equal([]string{"test-instance-id"}, testutils.InstanceUpdatesFromEngine(deployEngine))
	s.Equal(0, testutils.InstanceCreatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) Test_staged_deploy_creates_the_instance_it_staged_as_new() {
	// Another deploy takes the name after staging found it free, so the change set was
	// staged for a new instance and must not be applied to the one that now exists.
	deployEngine := testutils.NewTestDeployEngineWithInstanceCreatedAfterLookup(
		testStagingEventsForDeploy(stagingSuccessUpdateDeploy),
		"test-changeset",
		testDeployEvents(deploySuccessCreate),
		testInstanceState(core.InstanceStatusDeployed),
		"test-instance",
	)

	s.runHeadlessStagedDeploy(deployEngine)

	s.Equal("", testutils.LastChangesetPayloadFromEngine(deployEngine).InstanceID)
	s.Equal(1, testutils.InstanceCreatesFromEngine(deployEngine))
	s.Empty(testutils.InstanceUpdatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) Test_staged_deploy_stops_when_the_instance_lookup_fails() {
	deployEngine := testutils.NewTestDeployEngineWithInstanceLookupError(errors.New("connection refused"))

	output := s.runHeadlessStagedDeploy(deployEngine)

	s.Contains(output, `looking up instance "test-instance": connection refused`)
	s.Nil(testutils.LastChangesetPayloadFromEngine(deployEngine), "nothing should be staged")
	s.Equal(0, testutils.InstanceCreatesFromEngine(deployEngine))
	s.Empty(testutils.InstanceUpdatesFromEngine(deployEngine))
}

func (s *DeployInstanceResolutionSuite) Test_accepting_drift_sends_an_operation_config() {
	deployEngine := testutils.NewTestDeployEngineWithDeploymentError(newDriftBlockedError())
	model := NewDeployModel(DeployModelConfig{
		DeployEngine:   deployEngine,
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset",
		InstanceID:     "test-instance-id",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(StartDeployMsg{})
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "Drift Detected", "drifted-resource")

	testModel.Type("a")
	s.Eventually(func() bool {
		return testutils.LastApplyReconciliationPayloadFromEngine(deployEngine) != nil
	}, 5*time.Second, 20*time.Millisecond)

	// No deploy config file is configured, so the engine's required config is sent empty.
	payload := testutils.LastApplyReconciliationPayloadFromEngine(deployEngine)
	s.Equal(&types.BlueprintOperationConfig{}, payload.Config)
	s.Len(payload.ResourceActions, 1)

	s.Require().NoError(testModel.Quit())
}

func newDriftBlockedError() error {
	return &engineerrors.ClientError{
		StatusCode: http.StatusConflict,
		Message:    "Drift detected: resources have changed externally",
		DriftBlockedResponse: &types.DriftBlockedResponse{
			Message:     "Drift detected: resources have changed externally",
			InstanceID:  "test-instance-id",
			ChangesetID: "test-changeset-drift",
			ReconciliationResult: &container.ReconciliationCheckResult{
				InstanceID: "test-instance-id",
				Resources: []container.ResourceReconcileResult{
					{
						ResourceID:        "drifted-resource-id",
						ResourceName:      "drifted-resource",
						ResourceType:      "aws/s3/bucket",
						Type:              container.ReconciliationTypeDrift,
						RecommendedAction: container.ReconciliationActionAcceptExternal,
					},
				},
				HasDrift: true,
			},
		},
	}
}
