package stateio

import "github.com/newstack-cloud/bluelink/libs/blueprint/state"

// Counts each distinct instance ID, children included, since a file can nest instances
// and repeat them.
func countInstances(instances []state.InstanceState) int {
	seen := map[string]bool{}
	for i := range instances {
		addInstanceIDs(&instances[i], seen)
	}
	return len(seen)
}

func addInstanceIDs(instance *state.InstanceState, seen map[string]bool) {
	if instance == nil || seen[instance.InstanceID] {
		return
	}
	seen[instance.InstanceID] = true
	for _, child := range instance.ChildBlueprints {
		addInstanceIDs(child, seen)
	}
}
