package jsonout

import (
	"encoding/json"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
)

// RedactChanges returns a copy of a change set with every value the provider
// declared sensitive replaced by a placeholder.
func RedactChanges(source *changes.BlueprintChanges) *changes.BlueprintChanges {
	if source == nil {
		return nil
	}

	// A round trip rather than a hand-written deep copy as the change set is a
	// large tree that gains fields over time, and a copy that silently misses
	// one would hand back a shared pointer for the caller to mutate.
	encoded, err := json.Marshal(source)
	if err != nil {
		return source
	}

	redacted := &changes.BlueprintChanges{}
	if err := json.Unmarshal(encoded, redacted); err != nil {
		return source
	}

	redactBlueprintChanges(redacted)

	return redacted
}

func redactBlueprintChanges(target *changes.BlueprintChanges) {
	for name, resourceChanges := range target.NewResources {
		target.NewResources[name] = redactResourceChanges(resourceChanges)
	}
	for name, resourceChanges := range target.ResourceChanges {
		target.ResourceChanges[name] = redactResourceChanges(resourceChanges)
	}

	redactFieldChangeMap(target.NewExports)
	redactFieldChangeMap(target.ExportChanges)

	for name, childChanges := range target.ChildChanges {
		redactBlueprintChanges(&childChanges)
		target.ChildChanges[name] = childChanges
	}
}

func redactResourceChanges(resourceChanges provider.Changes) provider.Changes {
	sensitivePaths := []string{}

	for i := range resourceChanges.ModifiedFields {
		if resourceChanges.ModifiedFields[i].Sensitive {
			sensitivePaths = append(sensitivePaths, resourceChanges.ModifiedFields[i].FieldPath)
			redactFieldChange(&resourceChanges.ModifiedFields[i])
		}
	}
	for i := range resourceChanges.NewFields {
		if resourceChanges.NewFields[i].Sensitive {
			sensitivePaths = append(sensitivePaths, resourceChanges.NewFields[i].FieldPath)
			redactFieldChange(&resourceChanges.NewFields[i])
		}
	}

	// The change set also carries the resolved spec the deploy phase will apply,
	// which holds the same values the field changes describe. Redacting only the
	// field changes would leave every secret readable a few lines further down.
	redactSpecPaths(&resourceChanges, sensitivePaths)

	return resourceChanges
}

func redactSpecPaths(resourceChanges *provider.Changes, sensitivePaths []string) {
	if len(sensitivePaths) == 0 {
		return
	}

	resolved := resourceChanges.AppliedResourceInfo.ResourceWithResolvedSubs
	if resolved != nil {
		for _, path := range sensitivePaths {
			redactMappingNodePath(resolved.Spec, path)
		}
	}

	current := resourceChanges.AppliedResourceInfo.CurrentResourceState
	if current != nil {
		redactResourceStateSpec(current, sensitivePaths)
	}
}

func redactResourceStateSpec(resourceState *state.ResourceState, sensitivePaths []string) {
	for _, path := range sensitivePaths {
		redactMappingNodePath(resourceState.SpecData, path)
	}
}

// Replaces the value at a "spec.a.b" field path within a spec mapping node.
//
// Field paths are dotted, so a key containing a dot cannot be addressed and is
// left alone. Rather than leave such a value exposed, the nearest addressable
// ancestor is redacted whole.
func redactMappingNodePath(spec *core.MappingNode, fieldPath string) {
	if spec == nil {
		return
	}

	segments := strings.Split(strings.TrimPrefix(fieldPath, "spec."), ".")
	node := spec
	for i, segment := range segments {
		if node == nil || node.Fields == nil {
			return
		}

		child, ok := node.Fields[segment]
		if !ok {
			// An unaddressable key sits somewhere beneath here, so the whole
			// subtree goes rather than the value it holds.
			node.Fields = map[string]*core.MappingNode{}
			node.Items = nil
			node.Scalar = core.ScalarFromString(headless.RedactedValue)
			return
		}

		if i == len(segments)-1 {
			node.Fields[segment] = core.MappingNodeFromString(headless.RedactedValue)
			return
		}

		node = child
	}
}

func redactFieldChangeMap(fieldChanges map[string]provider.FieldChange) {
	for name, fieldChange := range fieldChanges {
		if !fieldChange.Sensitive {
			continue
		}
		redactFieldChange(&fieldChange)
		fieldChanges[name] = fieldChange
	}
}

func redactFieldChange(fieldChange *provider.FieldChange) {
	if fieldChange.PrevValue != nil {
		fieldChange.PrevValue = core.MappingNodeFromString(headless.RedactedValue)
	}
	if fieldChange.NewValue != nil {
		fieldChange.NewValue = core.MappingNodeFromString(headless.RedactedValue)
	}
}
