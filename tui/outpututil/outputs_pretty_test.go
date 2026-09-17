package outpututil

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type OutputsPrettyTestSuite struct {
	suite.Suite
	specData       *core.MappingNode
	computedFields []string
}

func TestOutputsPrettyTestSuite(t *testing.T) {
	suite.Run(t, new(OutputsPrettyTestSuite))
}

func (s *OutputsPrettyTestSuite) SetupTest() {
	s.specData = core.MappingNodeFields(
		"arn", core.MappingNodeFromString("arn:aws:lambda:eu-west-2:123:function:orders"),
		"environment", core.MappingNodeFields(
			"STAGE", core.MappingNodeFromString("production"),
			"REGION", core.MappingNodeFromString("eu-west-2"),
		),
	)
	s.computedFields = []string{"spec.arn", "spec.environment"}
}

func (s *OutputsPrettyTestSuite) fieldNamed(fields []OutputField, name string) OutputField {
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	s.Failf("field missing", "no output field named %q", name)
	return OutputField{}
}

func (s *OutputsPrettyTestSuite) Test_concise_collapses_nested_values_to_one_line() {
	fields := CollectOutputFields(s.specData, s.computedFields)

	s.Equal("{...}", s.fieldNamed(fields, "environment").Value)
	s.NotContains(s.fieldNamed(fields, "arn").Value, "\n")
}

func (s *OutputsPrettyTestSuite) Test_pretty_renders_nested_values_in_full() {
	fields := CollectOutputFields(s.specData, s.computedFields)
	s.Equal("{...}", s.fieldNamed(fields, "environment").Value)

	pretty := s.fieldNamed(CollectOutputFieldsPretty(s.specData, s.computedFields), "environment")
	s.NotEqual("{...}", pretty.Value, "the full-screen view should not collapse nested outputs")
	s.Contains(pretty.Value, "STAGE")
	s.Contains(pretty.Value, "production")
	s.Contains(pretty.Value, "REGION")
	s.Contains(pretty.Value, "eu-west-2")
}

func (s *OutputsPrettyTestSuite) Test_pretty_leaves_scalars_alone() {
	pretty := s.fieldNamed(CollectOutputFieldsPretty(s.specData, s.computedFields), "arn")
	s.Equal(`"arn:aws:lambda:eu-west-2:123:function:orders"`, pretty.Value)
}

func (s *OutputsPrettyTestSuite) Test_pretty_still_excludes_internal_properties() {
	specData := core.MappingNodeFields(
		"arn", core.MappingNodeFromString("arn:aws:lambda:eu-west-2:123:function:orders"),
		"__ccPrimaryIdentifier", core.MappingNodeFromString("orders-fn"),
	)

	fields := CollectOutputFieldsPretty(
		specData,
		[]string{"spec.arn", "spec.__ccPrimaryIdentifier"},
	)

	s.Require().Len(fields, 1)
	s.Equal("arn", fields[0].Name)
}

func (s *OutputsPrettyTestSuite) Test_pretty_works_without_computed_field_paths() {
	// With no paths given, every top-level field is treated as an output.
	fields := CollectOutputFieldsPretty(s.specData, nil)
	s.Contains(s.fieldNamed(fields, "environment").Value, "STAGE")
}
