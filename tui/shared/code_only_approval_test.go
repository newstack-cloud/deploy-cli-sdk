package shared

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type CodeOnlyApprovalTestSuite struct {
	suite.Suite
}

func TestCodeOnlyApprovalTestSuite(t *testing.T) {
	suite.Run(t, new(CodeOnlyApprovalTestSuite))
}

func (s *CodeOnlyApprovalTestSuite) Test_nil_changes_is_eligible() {
	result := CheckCodeOnlyEligibility(nil, nil)
	s.True(result.Eligible)
	s.Empty(result.Reasons)
}

func (s *CodeOnlyApprovalTestSuite) Test_empty_changes_is_eligible() {
	result := CheckCodeOnlyEligibility(&changes.BlueprintChanges{}, nil)
	s.True(result.Eligible)
	s.Empty(result.Reasons)
}

func (s *CodeOnlyApprovalTestSuite) Test_all_code_hosting_updates_is_eligible() {
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myFunction": changesWithCategory(ResourceCategoryCodeHosting),
			"myApi":      changesWithCategory(ResourceCategoryCodeHosting),
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.True(result.Eligible)
	s.Empty(result.Reasons)
}

func (s *CodeOnlyApprovalTestSuite) Test_new_resources_denies() {
	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Contains(result.Reasons, "1 new resource(s) would be created")
}

func (s *CodeOnlyApprovalTestSuite) Test_removed_resources_denies() {
	bpChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"oldResource"},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Contains(result.Reasons, "1 resource(s) would be removed")
}

func (s *CodeOnlyApprovalTestSuite) Test_new_children_denies() {
	bpChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"childA": {},
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Contains(result.Reasons, "1 new child blueprint(s) would be created")
}

func (s *CodeOnlyApprovalTestSuite) Test_removed_children_denies() {
	bpChanges := &changes.BlueprintChanges{
		RemovedChildren: []string{"childA"},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Contains(result.Reasons, "1 child blueprint(s) would be removed")
}

func (s *CodeOnlyApprovalTestSuite) Test_recreated_children_denies() {
	bpChanges := &changes.BlueprintChanges{
		RecreateChildren: []string{"childA"},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Contains(result.Reasons, "1 child blueprint(s) would be recreated")
}

func (s *CodeOnlyApprovalTestSuite) Test_infrastructure_resource_denies() {
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myTable": changesWithCategory(ResourceCategoryInfrastructure),
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Require().Len(result.Reasons, 1)
	s.Contains(result.Reasons[0], "myTable")
	s.Contains(result.Reasons[0], "infrastructure")
}

func (s *CodeOnlyApprovalTestSuite) Test_missing_annotation_denies() {
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": changesWithMetadata(&state.ResourceMetadataState{
				Annotations: map[string]*core.MappingNode{},
			}),
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Require().Len(result.Reasons, 1)
	s.Contains(result.Reasons[0], "unclassified")
}

func (s *CodeOnlyApprovalTestSuite) Test_nil_metadata_denies() {
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": changesWithMetadata(nil),
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Require().Len(result.Reasons, 1)
	s.Contains(result.Reasons[0], "unclassified")
}

func (s *CodeOnlyApprovalTestSuite) Test_mixed_categories_collects_all_reasons() {
	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{"newRes": {}},
		ResourceChanges: map[string]provider.Changes{
			"myFunction": changesWithCategory(ResourceCategoryCodeHosting),
			"myTable":    changesWithCategory(ResourceCategoryInfrastructure),
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.GreaterOrEqual(len(result.Reasons), 2)
}

func (s *CodeOnlyApprovalTestSuite) Test_nested_child_infrastructure_denies() {
	childChanges := changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"childTable": changesWithCategory(ResourceCategoryInfrastructure),
		},
	}
	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"childA": childChanges,
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, nil)
	s.False(result.Eligible)
	s.Require().Len(result.Reasons, 1)
	s.Contains(result.Reasons[0], `child "childA"`)
	s.Contains(result.Reasons[0], "childTable")
}

func (s *CodeOnlyApprovalTestSuite) Test_instance_state_fallback_for_category() {
	// Resource changes without metadata in AppliedResourceInfo.
	bpChanges := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myFunction": changesWithMetadata(nil),
		},
	}
	// Instance state provides the metadata.
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{"myFunction": "res-123"},
		Resources: map[string]*state.ResourceState{
			"res-123": {
				Metadata: metadataWithCategory(ResourceCategoryCodeHosting),
			},
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, instanceState)
	s.True(result.Eligible)
	s.Empty(result.Reasons)
}

func (s *CodeOnlyApprovalTestSuite) Test_child_uses_child_instance_state() {
	childChanges := changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"childFunc": changesWithMetadata(nil),
		},
	}
	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"childA": childChanges,
		},
	}
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": {
				ResourceIDs: map[string]string{"childFunc": "child-res-1"},
				Resources: map[string]*state.ResourceState{
					"child-res-1": {
						Metadata: metadataWithCategory(ResourceCategoryCodeHosting),
					},
				},
			},
		},
	}
	result := CheckCodeOnlyEligibility(bpChanges, instanceState)
	s.True(result.Eligible)
}

// --- test helpers ---

func stringMappingNode(val string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &val},
	}
}

func metadataWithCategory(category string) *state.ResourceMetadataState {
	return &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationResourceCategory: stringMappingNode(category),
		},
	}
}

func changesWithCategory(category string) provider.Changes {
	return changesWithMetadata(metadataWithCategory(category))
}

func changesWithMetadata(meta *state.ResourceMetadataState) provider.Changes {
	var currentState *state.ResourceState
	if meta != nil {
		currentState = &state.ResourceState{Metadata: meta}
	}
	return provider.Changes{
		AppliedResourceInfo: provider.ResourceInfo{
			CurrentResourceState: currentState,
		},
	}
}
