package destroyui

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// The deploy target (and other plugin config) reaches the engine via the
// Config field of the destroy and change staging payloads. Without it, provider
// plugins run with an empty deploy target when staging destroy changes and when
// destroying an instance.
type DestroyOperationConfigSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDestroyOperationConfigSuite(t *testing.T) {
	suite.Run(t, new(DestroyOperationConfigSuite))
}

func (s *DestroyOperationConfigSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *DestroyOperationConfigSuite) Test_destroy_payload_carries_operation_config() {
	opConfig := testOperationConfig()
	captureEngine := newPayloadCaptureEngine(
		newMockDestroyEngineWithFullFlow(
			testStagingEventsForDestroy(stagingSuccessDeleteDestroy),
			testDestroyEvents(destroySuccess),
			"test-changeset-op-config",
			"test-instance-id",
		),
	)

	app, err := NewDestroyApp(DestroyAppConfig{
		Context:         context.Background(),
		DestroyEngine:   captureEngine,
		Logger:          zap.NewNop(),
		ChangesetID:     "test-changeset-op-config",
		InstanceName:    "op-config-instance",
		Styles:          s.styles,
		Headless:        true,
		HeadlessWriter:  testutils.NewSaveBuffer(),
		OperationConfig: opConfig,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), app, teatest.WithInitialTermSize(300, 100))
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(10*time.Second))

	s.Same(opConfig, captureEngine.capturedDestroyPayload().Config)
}

func (s *DestroyOperationConfigSuite) Test_destroy_change_staging_payload_carries_operation_config() {
	opConfig := testOperationConfig()
	captureEngine := newPayloadCaptureEngine(
		newMockDestroyEngineWithFullFlow(
			testStagingEventsForDestroy(stagingSuccessDeleteDestroy),
			testDestroyEvents(destroySuccess),
			"test-changeset-staging-op-config",
			"test-instance-id",
		),
	)

	app, err := NewDestroyApp(DestroyAppConfig{
		Context:         context.Background(),
		DestroyEngine:   captureEngine,
		Logger:          zap.NewNop(),
		InstanceName:    "op-config-instance",
		BlueprintFile:   s.writeTestBlueprint(),
		StageFirst:      true,
		AutoApprove:     true,
		Styles:          s.styles,
		Headless:        true,
		HeadlessWriter:  testutils.NewSaveBuffer(),
		OperationConfig: opConfig,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(s.T(), app, teatest.WithInitialTermSize(300, 100))
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(10*time.Second))

	s.Same(opConfig, captureEngine.capturedChangesetPayload().Config)
}

func (s *DestroyOperationConfigSuite) writeTestBlueprint() string {
	path := filepath.Join(s.T().TempDir(), "test.blueprint.yaml")
	err := os.WriteFile(path, []byte("version: 2025-05-12\nresources: {}\n"), 0644)
	s.Require().NoError(err)
	return path
}

func testOperationConfig() *types.BlueprintOperationConfig {
	return &types.BlueprintOperationConfig{
		ContextVariables: map[string]*core.ScalarValue{
			"deployTarget": core.ScalarFromString("aws-serverless"),
		},
	}
}

type payloadCaptureEngine struct {
	engine.DeployEngine

	mu               sync.Mutex
	destroyPayload   *types.DestroyBlueprintInstancePayload
	changesetPayload *types.CreateChangesetPayload
}

func newPayloadCaptureEngine(base engine.DeployEngine) *payloadCaptureEngine {
	return &payloadCaptureEngine{DeployEngine: base}
}

func (e *payloadCaptureEngine) DestroyBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.DestroyBlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	e.mu.Lock()
	e.destroyPayload = payload
	e.mu.Unlock()
	return e.DeployEngine.DestroyBlueprintInstance(ctx, instanceID, payload)
}

func (e *payloadCaptureEngine) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*types.ChangesetResponse, error) {
	e.mu.Lock()
	e.changesetPayload = payload
	e.mu.Unlock()
	return e.DeployEngine.CreateChangeset(ctx, payload)
}

func (e *payloadCaptureEngine) capturedDestroyPayload() *types.DestroyBlueprintInstancePayload {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.destroyPayload
}

func (e *payloadCaptureEngine) capturedChangesetPayload() *types.CreateChangesetPayload {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.changesetPayload
}
