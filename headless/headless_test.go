package headless

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HeadlessSuite struct {
	suite.Suite
}

func (s *HeadlessSuite) Test_required_with_explicit_value_passes() {
	rule := Required(Flag{Name: "test-flag", Value: "some-value", IsDefault: false})
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_with_default_value_returns_error() {
	rule := Required(Flag{Name: "test-flag", Value: "default", IsDefault: true})
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--test-flag")
	s.Contains(err.Error(), "non-interactive")
}

func (s *HeadlessSuite) Test_required_with_empty_value_returns_error() {
	rule := Required(Flag{Name: "test-flag", Value: "", IsDefault: false})
	err := rule.Validate()
	s.Error(err)
}

func (s *HeadlessSuite) Test_one_of_with_one_set_passes() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "", IsDefault: true},
		Flag{Name: "flag-b", Value: "value", IsDefault: false},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_one_of_with_none_set_returns_error() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "", IsDefault: true},
		Flag{Name: "flag-b", Value: "default", IsDefault: true},
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--flag-a")
	s.Contains(err.Error(), "--flag-b")
}

func (s *HeadlessSuite) Test_one_of_with_both_set_passes() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "value-a", IsDefault: false},
		Flag{Name: "flag-b", Value: "value-b", IsDefault: false},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_validate_in_interactive_mode_skips_validation() {
	cleanup := SetHeadlessForTesting(false)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "missing", Value: "", IsDefault: true}),
	)
	s.NoError(err)
}

func (s *HeadlessSuite) Test_validate_in_headless_mode_validates_rules() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "missing", Value: "", IsDefault: true}),
	)
	s.Error(err)
}

func (s *HeadlessSuite) Test_validate_aggregates_multiple_errors() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "flag-a", Value: "", IsDefault: true}),
		Required(Flag{Name: "flag-b", Value: "", IsDefault: true}),
	)
	s.Error(err)
	s.Contains(err.Error(), "--flag-a")
	s.Contains(err.Error(), "--flag-b")
}

func (s *HeadlessSuite) Test_validate_with_all_valid_returns_nil() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "flag-a", Value: "value", IsDefault: false}),
	)
	s.NoError(err)
}

// Condition tests

func (s *HeadlessSuite) Test_flag_present_condition_is_met_when_flag_is_set() {
	condition := FlagPresent(Flag{Name: "test-flag", Value: "value", IsDefault: false})
	s.True(condition.IsMet())
	s.Equal("--test-flag is set", condition.Description())
}

func (s *HeadlessSuite) Test_flag_present_condition_not_met_when_flag_is_default() {
	condition := FlagPresent(Flag{Name: "test-flag", Value: "default", IsDefault: true})
	s.False(condition.IsMet())
}

func (s *HeadlessSuite) Test_flag_present_condition_not_met_when_flag_is_empty() {
	condition := FlagPresent(Flag{Name: "test-flag", Value: "", IsDefault: false})
	s.False(condition.IsMet())
}

func (s *HeadlessSuite) Test_flag_equals_condition_is_met_when_values_match() {
	condition := FlagEquals(Flag{Name: "mode", Value: "staging", IsDefault: false}, "staging")
	s.True(condition.IsMet())
	s.Equal(`--mode is "staging"`, condition.Description())
}

func (s *HeadlessSuite) Test_flag_equals_condition_not_met_when_values_differ() {
	condition := FlagEquals(Flag{Name: "mode", Value: "production", IsDefault: false}, "staging")
	s.False(condition.IsMet())
}

func (s *HeadlessSuite) Test_bool_flag_true_condition_is_met_when_true() {
	condition := BoolFlagTrue("stage", true)
	s.True(condition.IsMet())
	s.Equal("--stage is set", condition.Description())
}

func (s *HeadlessSuite) Test_bool_flag_true_condition_not_met_when_false() {
	condition := BoolFlagTrue("stage", false)
	s.False(condition.IsMet())
}

// RequiredIf tests

func (s *HeadlessSuite) Test_required_if_passes_when_condition_not_met() {
	rule := RequiredIf(
		BoolFlagTrue("stage", false),
		Flag{Name: "auto-approve", Value: "", IsDefault: true},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_if_passes_when_condition_met_and_target_set() {
	rule := RequiredIf(
		BoolFlagTrue("stage", true),
		Flag{Name: "auto-approve", Value: "value", IsDefault: false},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_if_returns_error_when_condition_met_and_target_not_set() {
	rule := RequiredIf(
		BoolFlagTrue("stage", true),
		Flag{Name: "target-flag", Value: "", IsDefault: true},
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--target-flag")
	s.Contains(err.Error(), "--stage is set")
}

func (s *HeadlessSuite) Test_required_if_with_flag_present_condition() {
	rule := RequiredIf(
		FlagPresent(Flag{Name: "output", Value: "json", IsDefault: false}),
		Flag{Name: "format-spec", Value: "", IsDefault: true},
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--format-spec")
	s.Contains(err.Error(), "--output is set")
}

func (s *HeadlessSuite) Test_required_if_with_flag_equals_condition() {
	rule := RequiredIf(
		FlagEquals(Flag{Name: "mode", Value: "remote", IsDefault: false}, "remote"),
		Flag{Name: "host", Value: "", IsDefault: true},
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--host")
	s.Contains(err.Error(), `--mode is "remote"`)
}

// RequiredIfBool tests

func (s *HeadlessSuite) Test_required_if_bool_passes_when_condition_not_met() {
	rule := RequiredIfBool(
		BoolFlagTrue("stage", false),
		"auto-approve",
		false,
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_if_bool_passes_when_condition_met_and_target_is_true() {
	rule := RequiredIfBool(
		BoolFlagTrue("stage", true),
		"auto-approve",
		true,
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_if_bool_returns_error_when_condition_met_and_target_is_false() {
	rule := RequiredIfBool(
		BoolFlagTrue("stage", true),
		"auto-approve",
		false,
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--auto-approve")
	s.Contains(err.Error(), "--stage is set")
}

// Integration tests for Validate with RequiredIf

func (s *HeadlessSuite) Test_validate_with_required_if_in_headless_mode() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		RequiredIfBool(
			BoolFlagTrue("stage", true),
			"auto-approve",
			false,
		),
	)
	s.Error(err)
	s.Contains(err.Error(), "--auto-approve")
}

func (s *HeadlessSuite) Test_validate_with_required_if_skips_in_interactive_mode() {
	cleanup := SetHeadlessForTesting(false)
	defer cleanup()

	err := Validate(
		RequiredIfBool(
			BoolFlagTrue("stage", true),
			"auto-approve",
			false,
		),
	)
	s.NoError(err)
}

func TestHeadlessSuite(t *testing.T) {
	suite.Run(t, new(HeadlessSuite))
}
