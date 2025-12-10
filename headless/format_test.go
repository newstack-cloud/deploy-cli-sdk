package headless

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type FormatSuite struct {
	suite.Suite
}

func TestFormatSuite(t *testing.T) {
	suite.Run(t, new(FormatSuite))
}

// Helper to create pointer values
func strPtr(s string) *string       { return &s }
func intPtr(i int) *int             { return &i }
func floatPtr(f float64) *float64   { return &f }
func boolPtr(b bool) *bool          { return &b }

// FormatMappingNode tests

func (s *FormatSuite) Test_format_mapping_node_nil() {
	result := FormatMappingNode(nil)
	s.Equal("null", result)
}

func (s *FormatSuite) Test_format_mapping_node_string_scalar() {
	node := &core.MappingNode{
		Scalar: &core.ScalarValue{
			StringValue: strPtr("hello world"),
		},
	}

	result := FormatMappingNode(node)

	s.Equal("\"hello world\"", result)
}

func (s *FormatSuite) Test_format_mapping_node_int_scalar() {
	node := &core.MappingNode{
		Scalar: &core.ScalarValue{
			IntValue: intPtr(42),
		},
	}

	result := FormatMappingNode(node)

	s.Equal("42", result)
}

func (s *FormatSuite) Test_format_mapping_node_float_scalar() {
	node := &core.MappingNode{
		Scalar: &core.ScalarValue{
			FloatValue: floatPtr(3.14),
		},
	}

	result := FormatMappingNode(node)

	s.Contains(result, "3.14")
}

func (s *FormatSuite) Test_format_mapping_node_bool_scalar_true() {
	node := &core.MappingNode{
		Scalar: &core.ScalarValue{
			BoolValue: boolPtr(true),
		},
	}

	result := FormatMappingNode(node)

	s.Equal("true", result)
}

func (s *FormatSuite) Test_format_mapping_node_bool_scalar_false() {
	node := &core.MappingNode{
		Scalar: &core.ScalarValue{
			BoolValue: boolPtr(false),
		},
	}

	result := FormatMappingNode(node)

	s.Equal("false", result)
}

func (s *FormatSuite) Test_format_mapping_node_empty_array() {
	node := &core.MappingNode{
		Items: []*core.MappingNode{},
	}

	result := FormatMappingNode(node)

	s.Equal("[]", result)
}

func (s *FormatSuite) Test_format_mapping_node_array() {
	node := &core.MappingNode{
		Items: []*core.MappingNode{
			{Scalar: &core.ScalarValue{StringValue: strPtr("a")}},
			{Scalar: &core.ScalarValue{StringValue: strPtr("b")}},
			{Scalar: &core.ScalarValue{StringValue: strPtr("c")}},
		},
	}

	result := FormatMappingNode(node)

	s.Equal("[\"a\", \"b\", \"c\"]", result)
}

func (s *FormatSuite) Test_format_mapping_node_nested_array() {
	node := &core.MappingNode{
		Items: []*core.MappingNode{
			{
				Items: []*core.MappingNode{
					{Scalar: &core.ScalarValue{IntValue: intPtr(1)}},
					{Scalar: &core.ScalarValue{IntValue: intPtr(2)}},
				},
			},
			{Scalar: &core.ScalarValue{StringValue: strPtr("x")}},
		},
	}

	result := FormatMappingNode(node)

	s.Equal("[[1, 2], \"x\"]", result)
}

func (s *FormatSuite) Test_format_mapping_node_fields() {
	node := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key": {Scalar: &core.ScalarValue{StringValue: strPtr("value")}},
		},
	}

	result := FormatMappingNode(node)

	s.Equal("{...}", result)
}

func (s *FormatSuite) Test_format_mapping_node_substitution() {
	node := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{SubstitutionValue: &substitutions.Substitution{}},
			},
		},
	}

	result := FormatMappingNode(node)

	s.Equal("\"${...}\"", result)
}

func (s *FormatSuite) Test_format_mapping_node_mixed_substitution() {
	node := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: strPtr("Hello ")},
				{SubstitutionValue: &substitutions.Substitution{}},
				{StringValue: strPtr("!")},
			},
		},
	}

	result := FormatMappingNode(node)

	s.Equal("\"Hello ${...}!\"", result)
}

func (s *FormatSuite) Test_format_mapping_node_empty_node() {
	node := &core.MappingNode{}

	result := FormatMappingNode(node)

	s.Equal("unknown", result)
}

// FormatScalarValue tests

func (s *FormatSuite) Test_format_scalar_nil() {
	result := FormatScalarValue(nil)
	s.Equal("null", result)
}

func (s *FormatSuite) Test_format_scalar_string() {
	scalar := &core.ScalarValue{
		StringValue: strPtr("test value"),
	}

	result := FormatScalarValue(scalar)

	s.Equal("\"test value\"", result)
}

func (s *FormatSuite) Test_format_scalar_empty_string() {
	scalar := &core.ScalarValue{
		StringValue: strPtr(""),
	}

	result := FormatScalarValue(scalar)

	s.Equal("\"\"", result)
}

func (s *FormatSuite) Test_format_scalar_int() {
	scalar := &core.ScalarValue{
		IntValue: intPtr(12345),
	}

	result := FormatScalarValue(scalar)

	s.Equal("12345", result)
}

func (s *FormatSuite) Test_format_scalar_negative_int() {
	scalar := &core.ScalarValue{
		IntValue: intPtr(-100),
	}

	result := FormatScalarValue(scalar)

	s.Equal("-100", result)
}

func (s *FormatSuite) Test_format_scalar_float() {
	scalar := &core.ScalarValue{
		FloatValue: floatPtr(2.5),
	}

	result := FormatScalarValue(scalar)

	s.Contains(result, "2.5")
}

func (s *FormatSuite) Test_format_scalar_bool_true() {
	scalar := &core.ScalarValue{
		BoolValue: boolPtr(true),
	}

	result := FormatScalarValue(scalar)

	s.Equal("true", result)
}

func (s *FormatSuite) Test_format_scalar_bool_false() {
	scalar := &core.ScalarValue{
		BoolValue: boolPtr(false),
	}

	result := FormatScalarValue(scalar)

	s.Equal("false", result)
}

func (s *FormatSuite) Test_format_scalar_all_nil_fields() {
	scalar := &core.ScalarValue{}

	result := FormatScalarValue(scalar)

	s.Equal("null", result)
}

// DiagnosticLevel tests

func (s *FormatSuite) Test_diagnostic_level_name_error() {
	result := DiagnosticLevelName(DiagnosticLevelError)
	s.Equal("ERROR", result)
}

func (s *FormatSuite) Test_diagnostic_level_name_warning() {
	result := DiagnosticLevelName(DiagnosticLevelWarning)
	s.Equal("WARNING", result)
}

func (s *FormatSuite) Test_diagnostic_level_name_info() {
	result := DiagnosticLevelName(DiagnosticLevelInfo)
	s.Equal("INFO", result)
}

func (s *FormatSuite) Test_diagnostic_level_name_unknown() {
	result := DiagnosticLevelName(DiagnosticLevel(99))
	s.Equal("UNKNOWN", result)
}

func (s *FormatSuite) Test_diagnostic_level_from_core_error() {
	result := DiagnosticLevelFromCore(core.DiagnosticLevelError)
	s.Equal(DiagnosticLevelError, result)
}

func (s *FormatSuite) Test_diagnostic_level_from_core_warning() {
	result := DiagnosticLevelFromCore(core.DiagnosticLevelWarning)
	s.Equal(DiagnosticLevelWarning, result)
}

func (s *FormatSuite) Test_diagnostic_level_from_core_info() {
	result := DiagnosticLevelFromCore(core.DiagnosticLevelInfo)
	s.Equal(DiagnosticLevelInfo, result)
}

func (s *FormatSuite) Test_diagnostic_level_from_core_unknown() {
	// Unknown/invalid core level should default to Error
	result := DiagnosticLevelFromCore(core.DiagnosticLevel(99))
	s.Equal(DiagnosticLevelError, result)
}
