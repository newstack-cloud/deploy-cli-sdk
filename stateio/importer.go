package stateio

import (
	"context"
	"encoding/json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// StateImporter defines the interface for importing instance state data.
// This abstraction allows different storage backends to implement
// their own import logic with optimal performance characteristics.
type StateImporter interface {
	// ImportInstances imports a slice of instance states.
	// Implementations should handle batch operations efficiently where possible.
	ImportInstances(ctx context.Context, instances []state.InstanceState) error
}

// ContainerStateImporter implements StateImporter using a state.Container.
// This works with any backend that implements the state.Container interface.
// It uses the container's SaveBatch method for efficient bulk operations.
type ContainerStateImporter struct {
	container state.Container
	force     bool
}

// ContainerStateImporterOption configures a ContainerStateImporter.
type ContainerStateImporterOption func(*ContainerStateImporter)

// WithForce allows importing over instances that have an operation in progress.
// It exists for a deployment that was lost or abandoned without its status being updated,
// which would otherwise leave the instance blocked from being imported over.
func WithForce(force bool) ContainerStateImporterOption {
	return func(i *ContainerStateImporter) {
		i.force = force
	}
}

// NewContainerStateImporter creates a new importer that uses the given container.
func NewContainerStateImporter(
	container state.Container,
	opts ...ContainerStateImporterOption,
) *ContainerStateImporter {
	importer := &ContainerStateImporter{container: container}
	for _, opt := range opts {
		opt(importer)
	}
	return importer
}

// ImportInstances saves the instances and their nested child blueprints, checking each
// child as an instance in its own right. Nothing is written unless every instance passes
// the checks: a new instance can't take another's name, an existing one can't be renamed,
// and an existing one with an operation in progress is refused unless the importer was
// created WithForce.
//
// Existing instances get a version past both the stored and the imported one, so an
// objectstore engine holding the old version conflicts rather than writing over the import.
// Stop a memfile engine while importing; on postgres, only the in-progress check protects
// against a running engine.
func (i *ContainerStateImporter) ImportInstances(ctx context.Context, instances []state.InstanceState) error {
	flattened, err := flattenInstances(instances)
	if err != nil {
		return err
	}
	storedVersions, err := checkImport(ctx, i.container.Instances(), flattened, i.force)
	if err != nil {
		return err
	}
	flattened.moveVersionsPast(storedVersions)

	return i.container.Instances().SaveBatch(ctx, flattened.rootValues())
}

// ParseInstancesJSON parses a JSON array of instances from raw bytes.
func ParseInstancesJSON(data []byte) ([]state.InstanceState, error) {
	var instances []state.InstanceState
	if err := json.Unmarshal(data, &instances); err != nil {
		return nil, &ImportError{
			Code:    ErrCodeInvalidJSON,
			Message: "failed to parse instances JSON",
			Err:     err,
		}
	}
	return instances, nil
}

// ImportInstancesResult contains the result of an instances import.
type ImportInstancesResult struct {
	InstancesCount int
}

// ExecuteInstancesImport performs the instances import using the provided importer.
func ExecuteInstancesImport(
	ctx context.Context,
	importer StateImporter,
	data []byte,
) (*ImportInstancesResult, error) {
	instances, err := ParseInstancesJSON(data)
	if err != nil {
		return nil, err
	}

	if err := importer.ImportInstances(ctx, instances); err != nil {
		return nil, err
	}

	return &ImportInstancesResult{
		InstancesCount: countInstances(instances),
	}, nil
}
