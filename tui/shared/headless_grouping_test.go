package shared

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type HeadlessGroupingTestSuite struct {
	suite.Suite
}

func TestHeadlessGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(HeadlessGroupingTestSuite))
}

func (s *HeadlessGroupingTestSuite) Test_no_resources_returns_empty() {
	groups, ungrouped := GroupHeadlessResources(nil)
	s.Empty(groups)
	s.Empty(ungrouped)
}

func (s *HeadlessGroupingTestSuite) Test_all_ungrouped_returns_no_groups() {
	resources := []HeadlessResourceInfo{
		{Path: "bucket", Name: "bucket", Metadata: nil},
		{Path: "queue", Name: "queue", Metadata: nil},
	}
	groups, ungrouped := GroupHeadlessResources(resources)
	s.Empty(groups)
	s.Len(ungrouped, 2)
}

func (s *HeadlessGroupingTestSuite) Test_grouped_resources_partitioned() {
	meta := headlessGroupMeta("myFunc", "celerity/function")
	resources := []HeadlessResourceInfo{
		{Path: "lambda", Name: "lambda", Metadata: meta},
		{Path: "role", Name: "role", Metadata: meta},
		{Path: "bucket", Name: "bucket", Metadata: nil},
	}
	groups, ungrouped := GroupHeadlessResources(resources)
	s.Require().Len(groups, 1)
	s.Equal("myFunc", groups[0].Group.GroupName)
	s.Len(groups[0].Resources, 2)
	s.Len(ungrouped, 1)
}

func (s *HeadlessGroupingTestSuite) Test_multiple_groups_preserved_order() {
	metaA := headlessGroupMeta("myFunc", "celerity/function")
	metaB := headlessGroupMeta("myApi", "celerity/api")
	resources := []HeadlessResourceInfo{
		{Path: "lambda", Name: "lambda", Metadata: metaA},
		{Path: "apigw", Name: "apigw", Metadata: metaB},
		{Path: "role", Name: "role", Metadata: metaA},
	}
	groups, ungrouped := GroupHeadlessResources(resources)
	s.Require().Len(groups, 2)
	s.Equal("myFunc", groups[0].Group.GroupName)
	s.Equal("myApi", groups[1].Group.GroupName)
	s.Len(groups[0].Resources, 2)
	s.Len(groups[1].Resources, 1)
	s.Empty(ungrouped)
}

func (s *HeadlessGroupingTestSuite) Test_group_resources_sorted_by_name() {
	meta := headlessGroupMeta("myFunc", "celerity/function")
	resources := []HeadlessResourceInfo{
		{Path: "zeta", Name: "zeta", Metadata: meta},
		{Path: "alpha", Name: "alpha", Metadata: meta},
	}
	groups, _ := GroupHeadlessResources(resources)
	s.Require().Len(groups, 1)
	s.Equal("alpha", groups[0].Resources[0].Name)
	s.Equal("zeta", groups[0].Resources[1].Name)
}

func (s *HeadlessGroupingTestSuite) Test_split_top_level_resources() {
	resources := []HeadlessResourceInfo{
		{Path: "myFunc", Name: "myFunc"},
		{Path: "childA/nested", Name: "nested"},
		{Path: "childA/childB/deep", Name: "deep"},
	}
	atLevel, nested := SplitResourcesByPathLevel(resources, "")
	s.Len(atLevel, 1)
	s.Equal("myFunc", atLevel[0].Name)
	s.Len(nested, 2)
}

func (s *HeadlessGroupingTestSuite) Test_split_child_level_resources() {
	resources := []HeadlessResourceInfo{
		{Path: "myFunc", Name: "myFunc"},
		{Path: "childA/nested", Name: "nested"},
		{Path: "childA/childB/deep", Name: "deep"},
	}
	atLevel, nested := SplitResourcesByPathLevel(resources, "childA")
	s.Len(atLevel, 1)
	s.Equal("nested", atLevel[0].Name)
	s.Len(nested, 1)
}

func headlessGroupMeta(name, typ string) *state.ResourceMetadataState {
	return &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationSourceAbstractName: {Scalar: &core.ScalarValue{StringValue: &name}},
			AnnotationSourceAbstractType: {Scalar: &core.ScalarValue{StringValue: &typ}},
		},
	}
}
