package shared

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ExtractGroupingFromResolvedTestSuite struct {
	suite.Suite
}

func TestExtractGroupingFromResolvedTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractGroupingFromResolvedTestSuite))
}

func (s *ExtractGroupingFromResolvedTestSuite) Test_extracts_group_from_resolved_annotations() {
	resolved := resolvedWithAnnotations(map[string]string{
		AnnotationSourceAbstractName: "myFunc",
		AnnotationSourceAbstractType: "celerity/function",
	})
	group := ExtractGroupingFromResolved(resolved)
	s.Require().NotNil(group)
	s.Equal("myFunc", group.GroupName)
	s.Equal("celerity/function", group.GroupType)
}

func (s *ExtractGroupingFromResolvedTestSuite) Test_returns_nil_when_annotations_incomplete() {
	resolved := resolvedWithAnnotations(map[string]string{
		AnnotationSourceAbstractName: "myFunc",
	})
	s.Nil(ExtractGroupingFromResolved(resolved))
}

func (s *ExtractGroupingFromResolvedTestSuite) Test_returns_nil_for_missing_structures() {
	s.Nil(ExtractGroupingFromResolved(nil))
	s.Nil(ExtractGroupingFromResolved(&provider.ResolvedResource{}))
	s.Nil(ExtractGroupingFromResolved(&provider.ResolvedResource{
		Metadata: &provider.ResolvedResourceMetadata{},
	}))
}

func resolvedWithAnnotations(annotations map[string]string) *provider.ResolvedResource {
	return &provider.ResolvedResource{
		Metadata: &provider.ResolvedResourceMetadata{
			Annotations: core.MappingNodeFromStringMap(annotations),
		},
	}
}

// The role is declared by the transformer alongside the grouping annotations,
// so it has to survive both routes the TUI reads them by: persisted state, and
// the resolved resource in a change set.
func (s *ExtractGroupingFromResolvedTestSuite) Test_the_declared_role_is_carried_from_a_resolved_resource() {
	group := ExtractGroupingFromResolved(resolvedWithAnnotations(map[string]string{
		AnnotationSourceAbstractName: "appVpc",
		AnnotationSourceAbstractType: "celerity/vpc",
		AnnotationResourceRole:       ResourceRoleAmbient,
	}))

	s.Require().NotNil(group)
	s.Equal(ResourceRoleAmbient, group.Role)
	s.True(group.IsAmbient())
}

func (s *ExtractGroupingFromResolvedTestSuite) Test_an_absent_role_is_a_component() {
	group := ExtractGroupingFromResolved(resolvedWithAnnotations(map[string]string{
		AnnotationSourceAbstractName: "createOrder",
		AnnotationSourceAbstractType: "celerity/handler",
	}))

	s.Require().NotNil(group)
	s.Empty(group.Role)
	s.False(group.IsAmbient(), "no declared role means a component")
}

type ExtractGroupingRoleTestSuite struct {
	suite.Suite
}

func TestExtractGroupingRoleTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractGroupingRoleTestSuite))
}

func (s *ExtractGroupingRoleTestSuite) Test_the_declared_role_is_carried_from_state() {
	group := ExtractGrouping(&state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationSourceAbstractName: core.MappingNodeFromString("appVpc"),
			AnnotationSourceAbstractType: core.MappingNodeFromString("celerity/vpc"),
			AnnotationResourceRole:       core.MappingNodeFromString(ResourceRoleAmbient),
		},
	})

	s.Require().NotNil(group)
	s.True(group.IsAmbient())
}

func (s *ExtractGroupingRoleTestSuite) Test_state_without_a_role_is_a_component() {
	group := ExtractGrouping(&state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationSourceAbstractName: core.MappingNodeFromString("createOrder"),
			AnnotationSourceAbstractType: core.MappingNodeFromString("celerity/handler"),
		},
	})

	s.Require().NotNil(group)
	s.False(group.IsAmbient())
}
