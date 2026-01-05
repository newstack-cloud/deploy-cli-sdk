package headless

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// isHeadlessOverride allows tests to override headless detection
var isHeadlessOverride *bool

// IsHeadless returns true if not running in an interactive terminal
func IsHeadless() bool {
	if isHeadlessOverride != nil {
		return *isHeadlessOverride
	}
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// SetHeadlessForTesting overrides IsHeadless for testing purposes.
// Returns a cleanup function that restores the original behavior.
func SetHeadlessForTesting(headless bool) func() {
	isHeadlessOverride = &headless
	return func() { isHeadlessOverride = nil }
}

// Flag represents a flag value with its default status
type Flag struct {
	Name      string // Flag name for error messages (e.g., "blueprint-file")
	Value     string // Current value
	IsDefault bool   // True if using default value (not explicitly set)
}

// Requirement defines a validation rule
type Requirement interface {
	Validate() error
}

// Required creates a rule that the flag must be explicitly provided
func Required(f Flag) Requirement {
	return &requiredRule{flag: f}
}

type requiredRule struct {
	flag Flag
}

func (r *requiredRule) Validate() error {
	if r.flag.IsDefault || r.flag.Value == "" {
		return fmt.Errorf("required flag --%s must be provided in non-interactive mode", r.flag.Name)
	}
	return nil
}

// OneOf creates a rule that at least one of the flags must be provided
func OneOf(flags ...Flag) Requirement {
	return &oneOfRule{flags: flags}
}

type oneOfRule struct {
	flags []Flag
}

func (r *oneOfRule) Validate() error {
	for _, f := range r.flags {
		if !f.IsDefault && f.Value != "" {
			return nil // At least one is set
		}
	}
	names := make([]string, len(r.flags))
	for i, f := range r.flags {
		names[i] = "--" + f.Name
	}
	return fmt.Errorf("one of %s must be provided in non-interactive mode", strings.Join(names, " or "))
}

// Condition represents a predicate that can be evaluated
type Condition interface {
	IsMet() bool
	Description() string
}

// FlagPresent returns a condition that is true when the flag is explicitly set
func FlagPresent(f Flag) Condition {
	return &flagPresentCondition{flag: f}
}

type flagPresentCondition struct {
	flag Flag
}

func (c *flagPresentCondition) IsMet() bool {
	return !c.flag.IsDefault && c.flag.Value != ""
}

func (c *flagPresentCondition) Description() string {
	return "--" + c.flag.Name + " is set"
}

// FlagEquals returns a condition that is true when the flag has the specified value
func FlagEquals(f Flag, value string) Condition {
	return &flagEqualsCondition{flag: f, expectedValue: value}
}

type flagEqualsCondition struct {
	flag          Flag
	expectedValue string
}

func (c *flagEqualsCondition) IsMet() bool {
	return c.flag.Value == c.expectedValue
}

func (c *flagEqualsCondition) Description() string {
	return fmt.Sprintf("--%s is %q", c.flag.Name, c.expectedValue)
}

// BoolFlagTrue returns a condition that is true when the bool flag is set to true
func BoolFlagTrue(name string, value bool) Condition {
	return &boolFlagTrueCondition{name: name, value: value}
}

type boolFlagTrueCondition struct {
	name  string
	value bool
}

func (c *boolFlagTrueCondition) IsMet() bool {
	return c.value
}

func (c *boolFlagTrueCondition) Description() string {
	return "--" + c.name + " is set"
}

// RequiredIf creates a rule that the target flag is required when the condition is met
func RequiredIf(condition Condition, target Flag) Requirement {
	return &requiredIfRule{condition: condition, target: target}
}

type requiredIfRule struct {
	condition Condition
	target    Flag
}

func (r *requiredIfRule) Validate() error {
	if r.condition.IsMet() && (r.target.IsDefault || r.target.Value == "") {
		return fmt.Errorf("--%s is required when %s", r.target.Name, r.condition.Description())
	}
	return nil
}

// RequiredIfBool creates a rule that the target bool flag must be true when the condition is met
func RequiredIfBool(condition Condition, targetName string, targetValue bool) Requirement {
	return &requiredIfBoolRule{condition: condition, targetName: targetName, targetValue: targetValue}
}

type requiredIfBoolRule struct {
	condition   Condition
	targetName  string
	targetValue bool
}

func (r *requiredIfBoolRule) Validate() error {
	if r.condition.IsMet() && !r.targetValue {
		return fmt.Errorf("--%s is required when %s", r.targetName, r.condition.Description())
	}
	return nil
}

// Validate checks all requirements, but only if in headless mode.
// In interactive mode, returns nil (TUI will handle missing values).
func Validate(requirements ...Requirement) error {
	if !IsHeadless() {
		return nil
	}

	var errs []string
	for _, req := range requirements {
		if err := req.Validate(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return fmt.Errorf("%s", errs[0])
	}
	return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
}
