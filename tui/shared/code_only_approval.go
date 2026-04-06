package shared

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// CodeOnlyResult holds the outcome of a code-only eligibility check.
type CodeOnlyResult struct {
	// Eligible is true when all changes are modifications to code-hosting resources only.
	Eligible bool
	// Reasons lists why auto-approval was denied (empty when eligible).
	Reasons []string
}

// CheckCodeOnlyEligibility analyses a changeset to determine if it contains
// only modifications to code-hosting resources with no structural changes.
// A nil or empty changeset is considered eligible.
func CheckCodeOnlyEligibility(
	bpChanges *changes.BlueprintChanges,
	instanceState *state.InstanceState,
) CodeOnlyResult {
	if bpChanges == nil {
		return CodeOnlyResult{Eligible: true}
	}

	var reasons []string
	reasons = append(reasons, checkStructuralChanges(bpChanges)...)
	reasons = append(reasons, checkResourceCategories(bpChanges.ResourceChanges, instanceState)...)
	reasons = append(reasons, checkChildChanges(bpChanges.ChildChanges, instanceState)...)

	return CodeOnlyResult{
		Eligible: len(reasons) == 0,
		Reasons:  reasons,
	}
}

func checkStructuralChanges(bpChanges *changes.BlueprintChanges) []string {
	var reasons []string

	if n := len(bpChanges.NewResources); n > 0 {
		reasons = append(reasons, fmt.Sprintf("%d new resource(s) would be created", n))
	}
	if n := len(bpChanges.RemovedResources); n > 0 {
		reasons = append(reasons, fmt.Sprintf("%d resource(s) would be removed", n))
	}
	if n := len(bpChanges.NewChildren); n > 0 {
		reasons = append(reasons, fmt.Sprintf("%d new child blueprint(s) would be created", n))
	}
	if n := len(bpChanges.RemovedChildren); n > 0 {
		reasons = append(reasons, fmt.Sprintf("%d child blueprint(s) would be removed", n))
	}
	if n := len(bpChanges.RecreateChildren); n > 0 {
		reasons = append(reasons, fmt.Sprintf("%d child blueprint(s) would be recreated", n))
	}

	return reasons
}

func checkResourceCategories(
	resourceChanges map[string]provider.Changes,
	instanceState *state.InstanceState,
) []string {
	var reasons []string
	for name, resChanges := range resourceChanges {
		category := resolveResourceCategory(name, &resChanges, instanceState)
		if category != ResourceCategoryCodeHosting {
			reasons = append(reasons, fmt.Sprintf(
				"modified resource %q is not code-hosting (category: %s)",
				name, categoryDisplayName(category),
			))
		}
	}
	return reasons
}

func resolveResourceCategory(
	name string,
	resChanges *provider.Changes,
	instanceState *state.InstanceState,
) string {
	// Primary: metadata from the change's current resource state.
	if rs := resChanges.AppliedResourceInfo.CurrentResourceState; rs != nil {
		if category := ExtractResourceCategory(rs.Metadata); category != "" {
			return category
		}
	}

	// Fallback: look up in pre-deployment instance state.
	if rs := FindResourceStateByName(instanceState, name); rs != nil {
		if category := ExtractResourceCategory(rs.Metadata); category != "" {
			return category
		}
	}

	return ""
}

func checkChildChanges(
	childChanges map[string]changes.BlueprintChanges,
	instanceState *state.InstanceState,
) []string {
	var reasons []string
	for name, childBPChanges := range childChanges {
		var childState *state.InstanceState
		if instanceState != nil && instanceState.ChildBlueprints != nil {
			childState = instanceState.ChildBlueprints[name]
		}
		childResult := CheckCodeOnlyEligibility(&childBPChanges, childState)
		for _, reason := range childResult.Reasons {
			reasons = append(reasons, fmt.Sprintf("child %q: %s", name, reason))
		}
	}
	return reasons
}

func categoryDisplayName(category string) string {
	if category == "" {
		return "unclassified"
	}
	return category
}
