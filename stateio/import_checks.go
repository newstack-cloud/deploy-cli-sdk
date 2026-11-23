package stateio

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Statuses an engine sets while it is working on an instance. Importing over one of these
// would replace state the engine is about to write back. Interrupted and failed statuses
// are settled, so they do not block.
var inProgressInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusPreparing:          true,
	core.InstanceStatusDeploying:          true,
	core.InstanceStatusUpdating:           true,
	core.InstanceStatusDestroying:         true,
	core.InstanceStatusDeployRollingBack:  true,
	core.InstanceStatusUpdateRollingBack:  true,
	core.InstanceStatusDestroyRollingBack: true,
}

// The flattened instances of an import, each held once, with every repeated copy of a
// parent wired to the same child pointers.
type flattenedInstances struct {
	byID map[string]*state.InstanceState
	// Children always come before the parents that reference them.
	order []string
	// The instances no other instance in the import has as a child. Saving these saves
	// the rest, since every backend saves an instance's children along with it.
	roots []string
	// The parent and name each child ID is listed under.
	parents map[string]childBlueprintRef
}

type childBlueprintRef struct {
	parentID string
	name     string
}

// A file edited by hand can list an instance more than once, nested and at the top level.
// Only the roots are saved, and every copy of an ID must match the first, or the result
// would depend on the order of the file. The comparison is strict: a map left out in one
// copy and empty in another counts as a difference.
//
// A child listed under two parents, or under two names in one parent, is refused.
//
// Repeated IDs share one copy, so the same instance can be reached from more than one
// place, and the backends walk children without guarding against cycles. An instance that is one of its own ancestors along a
// path in the file is refused as a cycle. One that closes a cycle across two branches has
// copies with different children, and is refused as such.
func flattenInstances(instances []state.InstanceState) (*flattenedInstances, error) {
	flattened := &flattenedInstances{
		byID:    map[string]*state.InstanceState{},
		parents: map[string]childBlueprintRef{},
	}
	for i := range instances {
		if _, err := flattened.add(&instances[i], nil); err != nil {
			return nil, err
		}
	}

	childIDs := map[string]bool{}
	for _, instance := range flattened.byID {
		for _, child := range instance.ChildBlueprints {
			childIDs[child.InstanceID] = true
		}
	}

	isRoot := map[string]bool{}
	for i := range instances {
		id := instances[i].InstanceID
		if !childIDs[id] && !isRoot[id] {
			isRoot[id] = true
			flattened.roots = append(flattened.roots, id)
		}
	}
	return flattened, nil
}

// The ancestors are the IDs from the top-level instance down to the parent of this one.
func (f *flattenedInstances) add(instance *state.InstanceState, ancestors []string) (*state.InstanceState, error) {
	if instance.InstanceID == "" {
		return nil, missingInstanceIDError(ancestors)
	}
	if index := slices.Index(ancestors, instance.InstanceID); index >= 0 {
		return nil, childBlueprintCycleError(append(slices.Clone(ancestors[index:]), instance.InstanceID))
	}
	if existing, seen := f.byID[instance.InstanceID]; seen {
		return existing, f.checkSameAsFirstCopy(existing, instance, ancestors)
	}

	instanceCopy := *instance
	f.byID[instance.InstanceID] = &instanceCopy

	if len(instance.ChildBlueprints) > 0 {
		path := append(slices.Clone(ancestors), instance.InstanceID)
		instanceCopy.ChildBlueprints = make(map[string]*state.InstanceState, len(instance.ChildBlueprints))
		// Sorted so that the order instances are checked in, and the errors, do not
		// depend on map iteration.
		for _, childName := range slices.Sorted(maps.Keys(instance.ChildBlueprints)) {
			child := instance.ChildBlueprints[childName]
			if child == nil {
				return nil, emptyChildBlueprintError(instance.InstanceID, childName)
			}
			added, err := f.add(child, path)
			if err != nil {
				return nil, err
			}
			ref := childBlueprintRef{parentID: instance.InstanceID, name: childName}
			if err = f.addParent(child.InstanceID, ref); err != nil {
				return nil, err
			}
			instanceCopy.ChildBlueprints[childName] = added
		}
	}

	f.order = append(f.order, instance.InstanceID)
	return &instanceCopy, nil
}

// The engine creates a child blueprint instance for one include in one parent, so a child
// listed under a second parent, or twice under one parent, can only come from a file
// edited by hand.
func (f *flattenedInstances) addParent(childID string, ref childBlueprintRef) error {
	if first, seen := f.parents[childID]; seen {
		return childBlueprintMultipleParentsError(childID, first, ref)
	}
	f.parents[childID] = ref
	return nil
}

func childBlueprintMultipleParentsError(childID string, first, second childBlueprintRef) *ImportError {
	return &ImportError{
		Code: ErrCodeChildBlueprintMultipleParents,
		Message: fmt.Sprintf(
			"nothing was imported, instance %s is a child blueprint in more than one place: "+
				"%q of instance %s and %q of instance %s",
			childID, first.name, first.parentID, second.name, second.parentID,
		),
		Hint: "List the instance under only one parent, once.",
	}
}

// The children of a repeated copy are walked as well, so that a difference further down
// is found even though the first copy's children are the ones kept.
func (f *flattenedInstances) checkSameAsFirstCopy(
	first *state.InstanceState,
	instance *state.InstanceState,
	ancestors []string,
) error {
	childNames := slices.Sorted(maps.Keys(instance.ChildBlueprints))
	// Checked first, since the comparison reads every child's ID.
	for _, childName := range childNames {
		if instance.ChildBlueprints[childName] == nil {
			return emptyChildBlueprintError(instance.InstanceID, childName)
		}
	}
	if !sameInstanceContent(first, instance) {
		return instanceCopiesDifferError(instance.InstanceID)
	}

	// Parents are not recorded again here: the comparison has matched the children's names
	// and IDs to the first copy's, which recorded them.
	path := append(slices.Clone(ancestors), instance.InstanceID)
	for _, childName := range childNames {
		if _, err := f.add(instance.ChildBlueprints[childName], path); err != nil {
			return err
		}
	}
	return nil
}

// Children are compared by name and ID only, since each child's content is compared
// when it is reached in its own right.
func sameInstanceContent(a *state.InstanceState, b *state.InstanceState) bool {
	sameChildIDs := maps.EqualFunc(a.ChildBlueprints, b.ChildBlueprints, func(x, y *state.InstanceState) bool {
		return x.InstanceID == y.InstanceID
	})
	if !sameChildIDs {
		return false
	}
	ownA, ownB := *a, *b
	ownA.ChildBlueprints, ownB.ChildBlueprints = nil, nil
	return reflect.DeepEqual(ownA, ownB)
}

func instanceCopiesDifferError(instanceID string) *ImportError {
	return &ImportError{
		Code: ErrCodeInstanceCopiesDiffer,
		Message: fmt.Sprintf(
			"nothing was imported, instance %s appears more than once in the input file and the copies differ",
			instanceID,
		),
		Hint: "Export the state again, or make every copy of the instance the same.",
	}
}

func missingInstanceIDError(ancestors []string) *ImportError {
	message := "nothing was imported, an instance in the input file has no id"
	if len(ancestors) > 0 {
		message = fmt.Sprintf(
			"nothing was imported, a child blueprint of instance %s has no id", ancestors[len(ancestors)-1],
		)
	}
	return &ImportError{Code: ErrCodeInvalidJSON, Message: message}
}

func emptyChildBlueprintError(parentID string, childName string) *ImportError {
	return &ImportError{
		Code: ErrCodeInvalidJSON,
		Message: fmt.Sprintf(
			"nothing was imported, child blueprint %q of instance %s is empty", childName, parentID,
		),
	}
}

func childBlueprintCycleError(cycle []string) *ImportError {
	return &ImportError{
		Code: ErrCodeChildBlueprintCycle,
		Message: "nothing was imported, these instances are children of each other in a cycle: " +
			strings.Join(cycle, " -> "),
	}
}

// The top-level instances to save, carrying their checked and version-bumped children.
func (f *flattenedInstances) rootValues() []state.InstanceState {
	values := make([]state.InstanceState, len(f.roots))
	for i, id := range f.roots {
		values[i] = *f.byID[id]
	}
	return values
}

type importProblems struct {
	nameConflicts []string
	renames       []string
	inProgress    []string
}

func (p *importProblems) empty() bool {
	return len(p.nameConflicts) == 0 && len(p.renames) == 0 && len(p.inProgress) == 0
}

// Every instance is checked before anything is written. Only the postgres backend saves a
// batch atomically, so on the others a problem found part-way through would leave the
// import half applied. Returns the stored version of each instance that already exists.
func checkImport(
	ctx context.Context,
	instances state.InstancesContainer,
	flattened *flattenedInstances,
	force bool,
) (map[string]int64, error) {
	problems := &importProblems{}
	importedNames := map[string]string{}
	storedVersions := map[string]int64{}

	for _, id := range flattened.order {
		instance := flattened.byID[id]
		existing, err := instances.Get(ctx, id)
		if err != nil && !state.IsInstanceNotFound(err) {
			return nil, existingStateReadError(err)
		}

		if err != nil {
			err = checkNewInstanceName(ctx, instances, instance, importedNames, problems)
			if err != nil {
				return nil, existingStateReadError(err)
			}
			continue
		}

		checkExistingInstance(instance, &existing, force, problems)
		storedVersions[id] = existing.Version
	}

	if problems.empty() {
		return storedVersions, nil
	}

	return nil, problems.toError()
}

// Reported as a StateAccessError since nothing has been written yet, so an import can tell it
// apart from a failure part way through saving.
func existingStateReadError(err error) *StateAccessError {
	return &StateAccessError{
		Code:    ErrCodeStateUnavailable,
		Message: "the instances already in the state could not be read",
		Err:     err,
	}
}

// An existing instance's version is moved past both the stored and the imported version.
func (f *flattenedInstances) moveVersionsPast(storedVersions map[string]int64) {
	for id, storedVersion := range storedVersions {
		instance := f.byID[id]
		instance.Version = max(storedVersion, instance.Version) + 1
	}
}

func checkNewInstanceName(
	ctx context.Context,
	instances state.InstancesContainer,
	instance *state.InstanceState,
	importedNames map[string]string,
	problems *importProblems,
) error {
	if instance.InstanceName == "" {
		return nil
	}

	if otherID, taken := importedNames[instance.InstanceName]; taken {
		problems.nameConflicts = append(problems.nameConflicts, fmt.Sprintf(
			"%q is used by both %s and %s in the input file", instance.InstanceName, otherID, instance.InstanceID,
		))
		return nil
	}
	importedNames[instance.InstanceName] = instance.InstanceID

	ownerID, err := instances.LookupIDByName(ctx, instance.InstanceName)
	if err != nil {
		if state.IsInstanceNotFound(err) {
			return nil
		}
		return err
	}

	if ownerID != instance.InstanceID {
		problems.nameConflicts = append(problems.nameConflicts, fmt.Sprintf(
			"%q is already used by instance %s, not %s", instance.InstanceName, ownerID, instance.InstanceID,
		))
	}
	return nil
}

func checkExistingInstance(
	instance *state.InstanceState,
	existing *state.InstanceState,
	force bool,
	problems *importProblems,
) {
	if existing.InstanceName != instance.InstanceName {
		problems.renames = append(problems.renames, fmt.Sprintf(
			"%s is named %q, not %q", instance.InstanceID, existing.InstanceName, instance.InstanceName,
		))
	}

	if !force && inProgressInstanceStatuses[existing.Status] {
		problems.inProgress = append(problems.inProgress, fmt.Sprintf(
			"%s (%q) is %s", instance.InstanceID, existing.InstanceName, existing.Status,
		))
	}
}

// Every problem found is reported at once, so a large import is not fixed one error at a
// time. The code is that of the first kind of problem present.
func (p *importProblems) toError() *ImportError {
	sections := []string{}
	code := ImportErrorCode("")
	addSection := func(sectionCode ImportErrorCode, heading string, items []string) {
		if len(items) == 0 {
			return
		}
		if code == "" {
			code = sectionCode
		}
		sections = append(sections, fmt.Sprintf("%s:\n  - %s", heading, strings.Join(items, "\n  - ")))
	}

	addSection(
		ErrCodeInstanceNameConflict,
		"instance names already in use",
		p.nameConflicts,
	)
	addSection(
		ErrCodeInstanceNameImmutable,
		"instances cannot be renamed",
		p.renames,
	)
	addSection(
		ErrCodeInstanceOperationInProgress,
		"instances with an operation in progress",
		p.inProgress,
	)

	hint := ""
	if len(p.inProgress) > 0 {
		hint = "If an operation was abandoned without its status being updated, " +
			"import again with --force to overwrite the instance anyway."
	}

	return &ImportError{
		Code:    code,
		Message: "nothing was imported\n" + strings.Join(sections, "\n"),
		Hint:    hint,
	}
}
