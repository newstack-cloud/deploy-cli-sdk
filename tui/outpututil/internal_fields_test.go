package outpututil

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type InternalFieldsTestSuite struct {
	suite.Suite
}

func TestInternalFieldsTestSuite(t *testing.T) {
	suite.Run(t, new(InternalFieldsTestSuite))
}

func (s *InternalFieldsTestSuite) Test_identifies_provider_internal_paths() {
	s.True(IsInternalFieldPath("__ccPrimaryIdentifier"))
	s.True(IsInternalFieldPath("nested.__ccPrimaryIdentifier"))
	s.True(IsInternalFieldPath("__cc.identifier"))

	s.False(IsInternalFieldPath("arn"))
	s.False(IsInternalFieldPath("nested.arn"))
	// A lone underscore is a legitimate, if unusual, field name.
	s.False(IsInternalFieldPath("_arn"))
}

func (s *InternalFieldsTestSuite) Test_computed_field_outputs_exclude_internal_properties() {
	specData := core.MappingNodeFields(
		"arn", core.MappingNodeFromString("arn:aws:lambda:eu-west-2:123:function:orders"),
		"__ccPrimaryIdentifier", core.MappingNodeFromString("orders-fn"),
	)
	computedFields := []string{"spec.arn", "spec.__ccPrimaryIdentifier"}

	fields := CollectOutputFields(specData, computedFields)

	s.Require().Len(fields, 1)
	s.Equal("arn", fields[0].Name)
}

func (s *InternalFieldsTestSuite) Test_top_level_outputs_exclude_internal_properties() {
	specData := core.MappingNodeFields(
		"arn", core.MappingNodeFromString("arn:aws:lambda:eu-west-2:123:function:orders"),
		"__ccPrimaryIdentifier", core.MappingNodeFromString("orders-fn"),
	)

	// With no computed field paths, every top-level field is treated as output.
	fields := CollectOutputFields(specData, nil)

	s.Require().Len(fields, 1)
	s.Equal("arn", fields[0].Name)
}

func (s *InternalFieldsTestSuite) Test_spec_view_and_its_count_exclude_internal_properties() {
	specData := core.MappingNodeFields(
		"handler", core.MappingNodeFromString("index.handler"),
		"memorySize", core.MappingNodeFromString("512"),
		"__ccPrimaryIdentifier", core.MappingNodeFromString("orders-fn"),
	)

	fields := CollectNonComputedFieldsPretty(specData, nil)
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	s.Equal([]string{"handler", "memorySize"}, names)

	// The count on the spec hint has to agree with what the spec view shows.
	s.Equal(2, CountNonComputedFields(specData, nil))
}
