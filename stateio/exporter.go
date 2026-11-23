package stateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// StateExporter defines the interface for exporting instance state data.
// This abstraction allows different storage backends to implement
// their own export logic with optimal performance characteristics.
type StateExporter interface {
	// ExportInstances exports instances by their IDs or names.
	// If instanceFilters is empty, exports all instances.
	// Returns the full instance state including child blueprints. Each instance is
	// returned once: one nested under another returned instance is only returned there.
	ExportInstances(ctx context.Context, instanceFilters []string) ([]state.InstanceState, error)
}

// ContainerStateExporter implements StateExporter using a state.Container.
// This works with any backend that implements the state.Container interface.
type ContainerStateExporter struct {
	container state.Container
}

// NewContainerStateExporter creates a new exporter that uses the given container.
func NewContainerStateExporter(container state.Container) *ContainerStateExporter {
	return &ContainerStateExporter{container: container}
}

// ExportInstances retrieves instances using the container's batch get operation.
// If instanceFilters is empty, all instances are exported.
//
// An instance nested under another exported instance is exported only there, and an
// instance the filters name more than once is exported once.
func (e *ContainerStateExporter) ExportInstances(
	ctx context.Context,
	instanceFilters []string,
) ([]state.InstanceState, error) {
	if len(instanceFilters) == 0 {
		instances, err := e.exportAllInstances(ctx)
		if err != nil {
			return nil, err
		}
		return topLevelInstances(instances)
	}

	instances, err := e.container.Instances().GetBatch(ctx, instanceFilters)
	if err != nil {
		return nil, createInstanceNotFoundError(err)
	}
	return topLevelInstances(instances)
}

// Drops every instance nested under another one in the result, and repeats of one the
// filters name twice. An export of everything lists children alongside their parents, and
// postgres GetBatch in blueprint-state v0.12.0 returns a requested instance that is also
// another's descendant without its children, which it wires onto the nested copy only.
//
// Instances in a stored cycle are all nested under each other, so none would be kept.
// That is refused rather than exported as nothing.
func topLevelInstances(instances []state.InstanceState) ([]state.InstanceState, error) {
	nested := map[string]bool{}
	for i := range instances {
		for _, child := range instances[i].ChildBlueprints {
			addInstanceIDs(child, nested)
		}
	}

	included := map[string]bool{}
	topLevel := make([]state.InstanceState, 0, len(instances))
	for i := range instances {
		if !nested[instances[i].InstanceID] && !included[instances[i].InstanceID] {
			addInstanceIDs(&instances[i], included)
			topLevel = append(topLevel, instances[i])
		}
	}

	var inCycle []string
	for i := range instances {
		if !included[instances[i].InstanceID] && !slices.Contains(inCycle, instances[i].InstanceID) {
			inCycle = append(inCycle, instances[i].InstanceID)
		}
	}
	if len(inCycle) > 0 {
		return nil, &ExportError{
			Code: ErrCodeExportFailed,
			Message: "these instances are children of each other in a cycle in the state, " +
				"so none of them can be exported: " + strings.Join(inCycle, ", "),
		}
	}
	return topLevel, nil
}

func createInstanceNotFoundError(err error) *ExportError {
	if notFoundErr, ok := errors.AsType[*state.InstancesNotFoundError](err); ok {
		return &ExportError{
			Code:    ErrCodeInstanceNotFound,
			Message: formatMissingInstancesMessage(notFoundErr.MissingIDsOrNames),
			Err:     err,
		}
	}

	return &ExportError{
		Code:    ErrCodeInstanceNotFound,
		Message: "one or more instances not found",
		Err:     err,
	}
}

func formatMissingInstancesMessage(missing []string) string {
	if len(missing) == 1 {
		return fmt.Sprintf("instance %q not found", missing[0])
	}

	quoted := make([]string, len(missing))
	for i, m := range missing {
		quoted[i] = fmt.Sprintf("%q", m)
	}
	return fmt.Sprintf("instances not found: %s", strings.Join(quoted, ", "))
}

func (e *ContainerStateExporter) exportAllInstances(ctx context.Context) ([]state.InstanceState, error) {
	result, err := e.container.Instances().List(ctx, state.ListInstancesParams{Limit: 0})
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to list instances",
			Err:     err,
		}
	}

	if len(result.Instances) == 0 {
		return []state.InstanceState{}, nil
	}

	ids := make([]string, len(result.Instances))
	for i, summary := range result.Instances {
		ids[i] = summary.InstanceID
	}

	instances, err := e.container.Instances().GetBatch(ctx, ids)
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to retrieve instances",
			Err:     err,
		}
	}

	return instances, nil
}

// SerializeInstancesJSON serializes instances to a JSON byte array.
func SerializeInstancesJSON(instances []state.InstanceState) ([]byte, error) {
	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeExportFailed,
			Message: "failed to serialize instances to JSON",
			Err:     err,
		}
	}
	return data, nil
}

// ExportInstancesResult contains the result of an instances export.
type ExportInstancesResult struct {
	InstancesCount int
	Data           []byte
}

// ExecuteInstancesExport performs the instances export using the provided exporter.
func ExecuteInstancesExport(
	ctx context.Context,
	exporter StateExporter,
	instanceFilters []string,
) (*ExportInstancesResult, error) {
	instances, err := exporter.ExportInstances(ctx, instanceFilters)
	if err != nil {
		return nil, err
	}

	data, err := SerializeInstancesJSON(instances)
	if err != nil {
		return nil, err
	}

	return &ExportInstancesResult{
		InstancesCount: countInstances(instances),
		Data:           data,
	}, nil
}
