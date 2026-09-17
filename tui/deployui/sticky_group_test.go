package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/stretchr/testify/suite"
)

type StickyGroupTestSuite struct {
	suite.Suite
}

func TestStickyGroupTestSuite(t *testing.T) {
	suite.Run(t, new(StickyGroupTestSuite))
}

func (s *StickyGroupTestSuite) Test_group_is_kept_once_state_has_been_seen() {
	resource := &ResourceDeployItem{Name: "orderHandler_lambda", Changes: &provider.Changes{}}
	item := &DeployItem{Type: ItemTypeResource, Resource: resource}

	// Before the resource is deployed there is nothing to read annotations from.
	s.Nil(item.GetResourceGroup())

	// A state refresh mid-deployment hydrates it.
	resource.ResourceState = groupedResourceState("orderHandler_lambda", "orderHandler")
	group := item.GetResourceGroup()
	s.Require().NotNil(group)
	s.Equal("orderHandler", group.GroupName)

	// A later refresh that arrives without the resource must not evict it from
	// its group, which is what made resources jump in and out mid-deployment.
	resource.ResourceState = nil
	stillGrouped := item.GetResourceGroup()
	s.Require().NotNil(stillGrouped, "resource fell out of its group")
	s.Equal("orderHandler", stillGrouped.GroupName)
}

func (s *StickyGroupTestSuite) Test_group_resolved_from_instance_state_is_kept() {
	resource := &ResourceDeployItem{Name: "orderHandler_lambda", Changes: &provider.Changes{}}
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{"orderHandler_lambda": "res-1"},
		Resources: map[string]*state.ResourceState{
			"res-1": groupedResourceState("orderHandler_lambda", "orderHandler"),
		},
	}
	item := &DeployItem{Type: ItemTypeResource, Resource: resource, InstanceState: instanceState}

	group := item.GetResourceGroup()
	s.Require().NotNil(group)

	// The instance state reference is dropped, but the group is already known.
	item.InstanceState = nil
	s.Require().NotNil(item.GetResourceGroup(), "resource fell out of its group")
}

func (s *StickyGroupTestSuite) Test_ungrouped_resources_stay_ungrouped() {
	resource := &ResourceDeployItem{
		Name:          "standaloneBucket",
		Changes:       &provider.Changes{},
		ResourceState: &state.ResourceState{Name: "standaloneBucket"},
	}
	item := &DeployItem{Type: ItemTypeResource, Resource: resource}

	s.Nil(item.GetResourceGroup())
	s.Nil(item.GetResourceGroup())
}

func groupedResourceState(name, groupName string) *state.ResourceState {
	return &state.ResourceState{
		Name: name,
		Metadata: &state.ResourceMetadataState{
			Annotations: map[string]*core.MappingNode{
				shared.AnnotationSourceAbstractName: core.MappingNodeFromString(groupName),
				shared.AnnotationSourceAbstractType: core.MappingNodeFromString("celerity/handler"),
			},
		},
	}
}
