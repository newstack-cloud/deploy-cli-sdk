package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
)

type testDeployEngine struct {
	validationEvents    []*types.BlueprintValidationEvent
	stagingEvents       []*types.ChangeStagingEvent
	deploymentEvents    []*types.BlueprintInstanceEvent
	changesetID         string
	changesetChanges    *changes.BlueprintChanges
	instanceID          string
	instanceState       *state.InstanceState
	createError         error
	createInstanceErr   error
	updateInstanceErr   error
	destroyInstanceErr  error
	getInstanceStateErr error
}

func NewTestDeployEngine(stubValidationEvents []*types.BlueprintValidationEvent) engine.DeployEngine {
	return &testDeployEngine{
		validationEvents: stubValidationEvents,
	}
}

// NewTestDeployEngineWithStaging creates a test deploy engine with staging event support.
func NewTestDeployEngineWithStaging(
	stubStagingEvents []*types.ChangeStagingEvent,
	changesetID string,
) engine.DeployEngine {
	return &testDeployEngine{
		stagingEvents: stubStagingEvents,
		changesetID:   changesetID,
	}
}

// NewTestDeployEngineWithStagingError creates a test deploy engine that returns an error on CreateChangeset.
func NewTestDeployEngineWithStagingError(err error) engine.DeployEngine {
	return &testDeployEngine{createError: err}
}

// NewTestDeployEngineWithDeployment creates a test deploy engine with deployment event support.
func NewTestDeployEngineWithDeployment(
	stubDeploymentEvents []*types.BlueprintInstanceEvent,
	instanceID string,
	instanceState *state.InstanceState,
) engine.DeployEngine {
	return &testDeployEngine{
		deploymentEvents: stubDeploymentEvents,
		instanceID:       instanceID,
		instanceState:    instanceState,
	}
}

// NewTestDeployEngineWithDeploymentAndChangeset creates a test deploy engine with deployment
// event support and changeset changes that can be fetched via GetChangeset.
func NewTestDeployEngineWithDeploymentAndChangeset(
	stubDeploymentEvents []*types.BlueprintInstanceEvent,
	instanceID string,
	instanceState *state.InstanceState,
	changesetChanges *changes.BlueprintChanges,
) engine.DeployEngine {
	return &testDeployEngine{
		deploymentEvents: stubDeploymentEvents,
		instanceID:       instanceID,
		instanceState:    instanceState,
		changesetChanges: changesetChanges,
	}
}

// NewTestDeployEngineWithDeploymentError creates a test deploy engine that returns an error on
// CreateBlueprintInstance or UpdateBlueprintInstance.
func NewTestDeployEngineWithDeploymentError(err error) engine.DeployEngine {
	return &testDeployEngine{
		createInstanceErr: err,
		updateInstanceErr: err,
	}
}

// NewTestDeployEngineWithNoInstanceState creates a test deploy engine that returns an error
// when GetBlueprintInstance is called (simulating no instance state available).
func NewTestDeployEngineWithNoInstanceState(
	stubDeploymentEvents []*types.BlueprintInstanceEvent,
	instanceID string,
) engine.DeployEngine {
	return &testDeployEngine{
		deploymentEvents:    stubDeploymentEvents,
		instanceID:          instanceID,
		getInstanceStateErr: errInstanceNotFound,
	}
}

// NewTestDeployEngineWithDestroyError creates a test deploy engine that returns an error on
// DestroyBlueprintInstance calls.
func NewTestDeployEngineWithDestroyError(err error) engine.DeployEngine {
	return &testDeployEngine{
		destroyInstanceErr: err,
	}
}

var errInstanceNotFound = fmt.Errorf("instance not found")

func (d *testDeployEngine) CreateBlueprintValidation(
	ctx context.Context,
	payload *types.CreateBlueprintValidationPayload,
	query *types.CreateBlueprintValidationQuery,
) (*types.BlueprintValidationResponse, error) {
	return &types.BlueprintValidationResponse{
		Data: &manage.BlueprintValidation{
			ID:                "test-validation-id",
			Status:            manage.BlueprintValidationStatusStarting,
			BlueprintLocation: payload.BlueprintFile,
			Created:           time.Now().Unix(),
		},
		LastEventID: "",
	}, nil
}

func (d *testDeployEngine) GetBlueprintValidation(
	ctx context.Context,
	validationID string,
) (*manage.BlueprintValidation, error) {
	return &manage.BlueprintValidation{
		ID:      "test-validation-id",
		Status:  manage.BlueprintValidationStatusValidated,
		Created: time.Now().Unix(),
	}, nil
}

func (d *testDeployEngine) StreamBlueprintValidationEvents(
	ctx context.Context,
	validationID string,
	lastEventID string,
	streamTo chan<- types.BlueprintValidationEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, validationEvent := range d.validationEvents {
			streamTo <- *validationEvent
		}
	}()

	return nil
}

func (d *testDeployEngine) CleanupBlueprintValidations(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CleanupChangesets(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CleanupEvents(
	ctx context.Context,
) error {
	return nil
}

func (d *testDeployEngine) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*types.ChangesetResponse, error) {
	if d.createError != nil {
		return nil, d.createError
	}
	return &types.ChangesetResponse{
		Data: &manage.Changeset{
			ID:                d.changesetID,
			Status:            manage.ChangesetStatusStagingChanges,
			BlueprintLocation: payload.BlueprintFile,
			Created:           time.Now().Unix(),
		},
		LastEventID: "",
	}, nil
}

func (d *testDeployEngine) GetChangeset(
	ctx context.Context,
	changesetID string,
) (*manage.Changeset, error) {
	if d.changesetChanges != nil {
		return &manage.Changeset{
			ID:      changesetID,
			Changes: d.changesetChanges,
		}, nil
	}
	return nil, nil
}

func (d *testDeployEngine) StreamChangeStagingEvents(
	ctx context.Context,
	changesetID string,
	lastEventID string,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, event := range d.stagingEvents {
			streamTo <- *event
		}
	}()
	return nil
}

func (d *testDeployEngine) CreateBlueprintInstance(
	ctx context.Context,
	payload *types.BlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	if d.createInstanceErr != nil {
		return nil, d.createInstanceErr
	}
	instanceState := d.instanceState
	if instanceState == nil {
		instanceState = &state.InstanceState{
			InstanceID: d.instanceID,
		}
	}
	return &types.BlueprintInstanceResponse{
		Data:        *instanceState,
		LastEventID: "",
	}, nil
}

func (d *testDeployEngine) UpdateBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.BlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	if d.updateInstanceErr != nil {
		return nil, d.updateInstanceErr
	}
	instanceState := d.instanceState
	if instanceState == nil {
		instanceState = &state.InstanceState{
			InstanceID: instanceID,
		}
	}
	return &types.BlueprintInstanceResponse{
		Data:        *instanceState,
		LastEventID: "",
	}, nil
}

func (d *testDeployEngine) GetBlueprintInstance(
	ctx context.Context,
	instanceID string,
) (*state.InstanceState, error) {
	if d.getInstanceStateErr != nil {
		return nil, d.getInstanceStateErr
	}
	if d.instanceState != nil {
		return d.instanceState, nil
	}
	return &state.InstanceState{
		InstanceID: instanceID,
	}, nil
}

func (d *testDeployEngine) GetBlueprintInstanceExports(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	return nil, nil
}

func (d *testDeployEngine) DestroyBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.DestroyBlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	if d.destroyInstanceErr != nil {
		return nil, d.destroyInstanceErr
	}
	instanceState := d.instanceState
	if instanceState == nil {
		instanceState = &state.InstanceState{
			InstanceID: instanceID,
		}
	}
	return &types.BlueprintInstanceResponse{
		Data:        *instanceState,
		LastEventID: "",
	}, nil
}

func (d *testDeployEngine) StreamBlueprintInstanceEvents(
	ctx context.Context,
	instanceID string,
	lastEventID string,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, event := range d.deploymentEvents {
			streamTo <- *event
		}
	}()
	return nil
}

func (d *testDeployEngine) ApplyReconciliation(
	ctx context.Context,
	instanceID string,
	payload *types.ApplyReconciliationPayload,
) (*container.ApplyReconciliationResult, error) {
	return nil, nil
}
