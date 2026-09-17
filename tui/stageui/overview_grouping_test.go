package stageui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/stretchr/testify/suite"
)

type OverviewGroupingTestSuite struct {
	suite.Suite
}

func TestOverviewGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(OverviewGroupingTestSuite))
}

func (s *OverviewGroupingTestSuite) Test_resources_carry_their_abstract_group() {
	items := assignOverviewGroups([]OverviewItem{
		groupedOverviewResource("lambdaFunction", "myFunc"),
		groupedOverviewResource("lambdaRole", "myFunc"),
		ungroupedOverviewResource("standaloneBucket"),
	})

	s.Require().NotNil(items[0].AbstractGroup)
	s.Equal("myFunc", items[0].AbstractGroup.GroupName)
	s.Require().NotNil(items[1].AbstractGroup)
	s.Equal("myFunc", items[1].AbstractGroup.GroupName)
	s.Nil(items[2].AbstractGroup)
}

func (s *OverviewGroupingTestSuite) Test_link_inherits_group_shared_by_both_endpoints() {
	items := assignOverviewGroups([]OverviewItem{
		groupedOverviewResource("lambdaFunction", "myFunc"),
		groupedOverviewResource("lambdaRole", "myFunc"),
		overviewLink("lambdaFunction::lambdaRole"),
	})

	s.Require().NotNil(items[2].AbstractGroup)
	s.Equal("myFunc", items[2].AbstractGroup.GroupName)
}

func (s *OverviewGroupingTestSuite) Test_cross_group_link_stays_ungrouped() {
	items := assignOverviewGroups([]OverviewItem{
		groupedOverviewResource("lambdaFunction", "myFunc"),
		groupedOverviewResource("apiGateway", "myApi"),
		overviewLink("lambdaFunction::apiGateway"),
	})

	s.Nil(items[2].AbstractGroup)
}

func (s *OverviewGroupingTestSuite) Test_children_are_never_grouped() {
	items := assignOverviewGroups([]OverviewItem{
		groupedOverviewResource("lambdaFunction", "myFunc"),
		{
			Item:        StageItem{Type: ItemTypeChild, Name: "notifications"},
			ElementPath: "children.notifications",
		},
	})

	s.Nil(items[1].AbstractGroup)
}

func groupedOverviewResource(name, groupName string) OverviewItem {
	item := ungroupedOverviewResource(name)
	item.Item.ResourceState = &state.ResourceState{
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

func ungroupedOverviewResource(name string) OverviewItem {
	return OverviewItem{
		Item:        StageItem{Type: ItemTypeResource, Name: name},
		ElementPath: "resources." + name,
	}
}

func overviewLink(linkName string) OverviewItem {
	return OverviewItem{
		Item:        StageItem{Type: ItemTypeLink, Name: linkName},
		ElementPath: "links." + linkName,
	}
}
