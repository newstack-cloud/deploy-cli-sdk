package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/stretchr/testify/suite"
)

type ResultCollectorGroupingTestSuite struct {
	suite.Suite
}

func TestResultCollectorGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(ResultCollectorGroupingTestSuite))
}

func (s *ResultCollectorGroupingTestSuite) Test_successful_resources_carry_their_abstract_group() {
	items := []DeployItem{
		groupedResourceItem("lambdaFunction", "myFunc"),
		groupedResourceItem("lambdaRole", "myFunc"),
		ungroupedResourceItem("standaloneBucket"),
	}

	collector := &ResultCollector{}
	collector.CollectFromItems(items, "")
	collector.AssignLinkGroups()

	s.Require().Len(collector.Successful, 3)
	s.Equal("myFunc", collector.Successful[0].AbstractGroup.GroupName)
	s.Equal("celerity/function", collector.Successful[0].AbstractGroup.GroupType)
	s.Equal("myFunc", collector.Successful[1].AbstractGroup.GroupName)
	s.Nil(collector.Successful[2].AbstractGroup)
}

func (s *ResultCollectorGroupingTestSuite) Test_link_inherits_group_shared_by_both_endpoints() {
	items := []DeployItem{
		groupedResourceItem("lambdaFunction", "myFunc"),
		groupedResourceItem("lambdaRole", "myFunc"),
		successfulLinkItem("lambdaFunction::lambdaRole"),
	}

	collector := &ResultCollector{}
	collector.CollectFromItems(items, "")
	collector.AssignLinkGroups()

	link := s.findSuccessful(collector, "lambdaFunction::lambdaRole")
	s.Require().NotNil(link.AbstractGroup)
	s.Equal("myFunc", link.AbstractGroup.GroupName)
}

func (s *ResultCollectorGroupingTestSuite) Test_cross_group_link_stays_ungrouped() {
	items := []DeployItem{
		groupedResourceItem("lambdaFunction", "myFunc"),
		groupedResourceItem("apiGateway", "myApi"),
		successfulLinkItem("lambdaFunction::apiGateway"),
	}

	collector := &ResultCollector{}
	collector.CollectFromItems(items, "")
	collector.AssignLinkGroups()

	link := s.findSuccessful(collector, "lambdaFunction::apiGateway")
	s.Nil(link.AbstractGroup)
}

func (s *ResultCollectorGroupingTestSuite) Test_link_between_ungrouped_resources_stays_ungrouped() {
	items := []DeployItem{
		groupedResourceItem("lambdaFunction", "myFunc"),
		ungroupedResourceItem("bucketOne"),
		ungroupedResourceItem("bucketTwo"),
		successfulLinkItem("bucketOne::bucketTwo"),
	}

	collector := &ResultCollector{}
	collector.CollectFromItems(items, "")
	collector.AssignLinkGroups()

	link := s.findSuccessful(collector, "bucketOne::bucketTwo")
	s.Nil(link.AbstractGroup)
}

func (s *ResultCollectorGroupingTestSuite) findSuccessful(c *ResultCollector, name string) SuccessfulElement {
	for _, elem := range c.Successful {
		if elem.ElementName == name {
			return elem
		}
	}
	s.Failf("element not collected", "no successful element named %q", name)
	return SuccessfulElement{}
}

func groupedResourceItem(name, groupName string) DeployItem {
	item := ungroupedResourceItem(name)
	item.Resource.ResourceState = &state.ResourceState{
		Name: name,
		Metadata: &state.ResourceMetadataState{
			Annotations: map[string]*core.MappingNode{
				shared.AnnotationSourceAbstractName: core.MappingNodeFromString(groupName),
				shared.AnnotationSourceAbstractType: core.MappingNodeFromString("celerity/function"),
			},
		},
	}
	return item
}

func ungroupedResourceItem(name string) DeployItem {
	return DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:    name,
			Status:  core.ResourceStatusCreated,
			Changes: &provider.Changes{},
		},
	}
}

func successfulLinkItem(linkName string) DeployItem {
	return DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkName:      linkName,
			ResourceAName: ExtractResourceAFromLinkName(linkName),
			ResourceBName: ExtractResourceBFromLinkName(linkName),
			Status:        core.LinkStatusCreated,
		},
	}
}
