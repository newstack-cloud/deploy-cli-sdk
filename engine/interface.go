package engine

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// DeployEngine is an interface used to allow the CLI to interact
// with the deploy engine that implements the v1 API.
type DeployEngine interface {

	// CreateBlueprintValidation creates a new blueprint validation
	// for the provided blueprint document and starts the validation process.
	// Returns a response containing the validation resource and a LastEventID
	// that should be passed to StreamBlueprintValidationEvents to avoid stale events.
	CreateBlueprintValidation(
		ctx context.Context,
		payload *types.CreateBlueprintValidationPayload,
		query *types.CreateBlueprintValidationQuery,
	) (*types.BlueprintValidationResponse, error)

	// GetBlueprintValidation retrieves metadata and status information
	// about a blueprint validation.
	// To get validation events (diagnostics), use the `StreamBlueprintValidationEvents`
	// method.
	GetBlueprintValidation(
		ctx context.Context,
		validationID string,
	) (*manage.BlueprintValidation, error)

	// StreamBlueprintValidationEvents streams events from a blueprint
	// validation process.
	// This will produce a stream of events as they occur or that have
	// recently occurred.
	// Any HTTP errors that occur when estabilishing a connection will be sent
	// to the provided error channel.
	//
	// The lastEventID parameter follows SSE Last-Event-ID semantics - it is exclusive,
	// meaning the event with this ID will NOT be included, only events after it.
	// Use the LastEventID from CreateBlueprintValidation to avoid stale events.
	StreamBlueprintValidationEvents(
		ctx context.Context,
		validationID string,
		lastEventID string,
		streamTo chan<- types.BlueprintValidationEvent,
		errChan chan<- error,
	) error

	// CleanupBlueprintValidations cleans up blueprint validations that are
	// older than the retention period configured for the Deploy Engine instance.
	// Returns a CleanupOperation that can be polled for completion status.
	CleanupBlueprintValidations(
		ctx context.Context,
	) (*manage.CleanupOperation, error)

	// CreateChangeset creates a change set for a blueprint deployment.
	// This will start a change staging process for the provided blueprint
	// document and return an ID that can be used to retrieve the change set
	// or stream change staging events.
	//
	// If a valid instance ID or name is provided, a change set will be created
	// by comparing the provided blueprint document with the current state of the
	// existing blueprint instance.
	//
	// Creating a change set should be carried out in preparation for deploying new blueprint
	// instances or updating existing blueprint instances.
	//
	// Returns a response containing the changeset and a LastEventID
	// that should be passed to StreamChangeStagingEvents to avoid stale events.
	CreateChangeset(
		ctx context.Context,
		payload *types.CreateChangesetPayload,
	) (*types.ChangesetResponse, error)

	// GetChangeset retrieves a change set for a blueprint deployment.
	// This will return the current status of the change staging process.
	// If complete, the response will include a full set of changes that
	// will be applied when deploying the change set.
	GetChangeset(
		ctx context.Context,
		changesetID string,
	) (*manage.Changeset, error)

	// StreamChangeStagingEvents streams events from the change staging process
	// for the given change set ID.
	// This will produce a stream of events as they occur or that have recently occurred.
	// Any HTTP errors that occur when estabilishing a connection will be sent
	// to the provided error channel.
	//
	// The lastEventID parameter follows SSE Last-Event-ID semantics - it is exclusive,
	// meaning the event with this ID will NOT be included, only events after it.
	// Use the LastEventID from CreateChangeset to avoid stale events from previous operations.
	StreamChangeStagingEvents(
		ctx context.Context,
		changesetID string,
		lastEventID string,
		streamTo chan<- types.ChangeStagingEvent,
		errChan chan<- error,
	) error

	// CleanupChangesets cleans up change sets that are older than the retention
	// period configured for the Deploy Engine instance.
	// Returns a CleanupOperation that can be polled for completion status.
	CleanupChangesets(
		ctx context.Context,
	) (*manage.CleanupOperation, error)

	// CreateBlueprintInstance (Deploy New) creates a new blueprint deployment instance.
	// This will start the deployment process for the provided blueprint
	// document and change set.
	// Returns a response containing the instance state and a LastEventID
	// that should be passed to StreamBlueprintInstanceEvents to avoid stale events.
	CreateBlueprintInstance(
		ctx context.Context,
		payload *types.BlueprintInstancePayload,
	) (*types.BlueprintInstanceResponse, error)

	// UpdateBlueprintInstance (Deploy Existing) updates an existing blueprint
	// deployment instance.
	// This will start the deployment process for the provided blueprint
	// document and change set.
	// Returns a response containing the instance state and a LastEventID
	// that should be passed to StreamBlueprintInstanceEvents to avoid stale events.
	UpdateBlueprintInstance(
		ctx context.Context,
		instanceID string,
		payload *types.BlueprintInstancePayload,
	) (*types.BlueprintInstanceResponse, error)

	// GetBlueprintInstance retrieves a blueprint deployment instance.
	// This will return the current status of the deployment along with
	// the current state of the blueprint intance.
	GetBlueprintInstance(
		ctx context.Context,
		instanceID string,
	) (*state.InstanceState, error)

	// ListBlueprintInstances retrieves blueprint instances with pagination and optional filtering.
	// The params allow filtering by name and paginating results.
	ListBlueprintInstances(
		ctx context.Context,
		params state.ListInstancesParams,
	) (state.ListInstancesResult, error)

	// GetBlueprintInstanceExports retrieves the exports from a blueprint
	// deployment instance.
	// This will return the exported fields from the blueprint instance.
	GetBlueprintInstanceExports(
		ctx context.Context,
		instanceID string,
	) (map[string]*state.ExportState, error)

	// DestroyBlueprintInstance destroys a blueprint deployment instance.
	// This will start the destroy process for the provided change set.
	// Returns a response containing the instance state and a LastEventID
	// that should be passed to StreamBlueprintInstanceEvents to avoid stale events.
	DestroyBlueprintInstance(
		ctx context.Context,
		instanceID string,
		payload *types.DestroyBlueprintInstancePayload,
	) (*types.BlueprintInstanceResponse, error)

	// StreamBlueprintInstanceEvents streams events from the current deployment
	// process for the given blueprint instance ID.
	//
	// This will stream events for new deployments, updates and for destroying
	// a blueprint instance.
	//
	// This will produce a stream of events as they occur or that have recently occurred.
	//
	// For a blueprint instance that has been destroyed, this stream will no longer be available
	// to new connections once the destroy process has been successfully completed.
	//
	// Any HTTP errors that occur when estabilishing a connection or unexpected failures
	// in the deployment process will be sent to the provided error channel.
	//
	// The lastEventID parameter follows SSE Last-Event-ID semantics - it is exclusive,
	// meaning the event with this ID will NOT be included, only events after it.
	// Use the LastEventID from Create/Update/DestroyBlueprintInstance to avoid stale events.
	StreamBlueprintInstanceEvents(
		ctx context.Context,
		instanceID string,
		lastEventID string,
		streamTo chan<- types.BlueprintInstanceEvent,
		errChan chan<- error,
	) error

	// CleanupEvents cleans up events that are older than the retention
	// period configured for the Deploy Engine instance.
	//
	// This will clean up events for all processes including blueprint validations,
	// change staging and deployments. This will not clean up the resources themselves,
	// only the events that are associated with them.
	// You can clean up change sets and blueprint validations using the dedicated methods.
	// Returns a CleanupOperation that can be polled for completion status.
	CleanupEvents(
		ctx context.Context,
	) (*manage.CleanupOperation, error)

	// CleanupReconciliationResults cleans up reconciliation check results that are
	// older than the retention period configured for the Deploy Engine instance.
	// Returns a CleanupOperation that can be polled for completion status.
	CleanupReconciliationResults(
		ctx context.Context,
	) (*manage.CleanupOperation, error)

	// GetCleanupOperation retrieves the status of a cleanup operation by ID.
	GetCleanupOperation(
		ctx context.Context,
		cleanupType manage.CleanupType,
		operationID string,
	) (*manage.CleanupOperation, error)

	// WaitForCleanupCompletion polls until the cleanup operation completes or fails.
	WaitForCleanupCompletion(
		ctx context.Context,
		cleanupType manage.CleanupType,
		operationID string,
		pollInterval time.Duration,
	) (*manage.CleanupOperation, error)

	// ApplyReconciliation applies reconciliation actions to resolve drift or interrupted state.
	// This is a synchronous operation that returns the result of applying the reconciliation actions.
	//
	// The instanceID parameter can be either the unique instance ID or
	// the user-defined instance name.
	ApplyReconciliation(
		ctx context.Context,
		instanceID string,
		payload *types.ApplyReconciliationPayload,
	) (*container.ApplyReconciliationResult, error)
}
