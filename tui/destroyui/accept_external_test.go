package destroyui

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DestroyAcceptExternalSuite struct {
	suite.Suite
}

func TestDestroyAcceptExternalSuite(t *testing.T) {
	suite.Run(t, new(DestroyAcceptExternalSuite))
}

func (s *DestroyAcceptExternalSuite) Test_accepting_drift_sends_an_operation_config() {
	destroyEngine := testutils.NewTestDeployEngineWithDestroyError(createDriftBlockedError())
	model := NewDestroyModel(DestroyModelConfig{
		DestroyEngine:  destroyEngine,
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset-drift",
		InstanceID:     "test-instance-id",
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		HeadlessWriter: os.Stdout,
	})

	testModel := teatest.NewTestModel(s.T(), model, teatest.WithInitialTermSize(300, 100))
	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContainsAll(s.T(), testModel.Output(), "Drift Detected", "drifted-resource")

	testModel.Type("a")
	s.Eventually(func() bool {
		return testutils.LastApplyReconciliationPayloadFromEngine(destroyEngine) != nil
	}, 5*time.Second, 20*time.Millisecond)

	// No deploy config file is configured, so the engine's required config is sent empty.
	payload := testutils.LastApplyReconciliationPayloadFromEngine(destroyEngine)
	s.Equal(&types.BlueprintOperationConfig{}, payload.Config)
	s.Len(payload.ResourceActions, 1)

	s.Require().NoError(testModel.Quit())
}

func (s *DestroyAcceptExternalSuite) Test_staged_destroy_stops_when_the_instance_lookup_fails() {
	destroyEngine := testutils.NewTestDeployEngineWithInstanceLookupError(errors.New("connection refused"))
	headlessOutput := &bytes.Buffer{}
	app, err := NewDestroyApp(DestroyAppConfig{
		DestroyEngine:  destroyEngine,
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		StageFirst:     true,
		AutoApprove:    true,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), app, teatest.WithInitialTermSize(300, 100))
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	s.Contains(headlessOutput.String(), `looking up instance "test-instance": connection refused`)
	s.Nil(testutils.LastChangesetPayloadFromEngine(destroyEngine), "nothing should be staged")
}
