// Package commands provides shared command factory functions and configuration
// types for CLIs that use the Deploy CLI SDK.
package commands

import (
	"context"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/precommand"
)

// Flag name constants shared across command files to avoid duplicate string literals.
const (
	flagInstanceName = "instance-name"
	flagInstanceID   = "instance-id"
	flagChangeSetID  = "change-set-id"
	flagAutoApprove  = "auto-approve"
)

// newTUIProgramOptions returns the standard tea.ProgramOption slice
// for headless vs interactive mode. The context is bound to the program so the
// TUI (and the engine calls it drives) is cancelled when ctx is cancelled
// (e.g. on Ctrl+C via the root signal context).
func newTUIProgramOptions(ctx context.Context, headlessMode bool) []tea.ProgramOption {
	if headlessMode {
		return []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(nil), tea.WithoutRenderer()}
	}
	return []tea.ProgramOption{tea.WithContext(ctx), tea.WithAltScreen(), tea.WithMouseCellMotion()}
}

// createPreflight builds the preflight model when a PreflightFactory is
// configured and the user has not opted out via --skip-plugin-check. The
// context is passed to the factory so preflight operations (e.g. plugin
// dependency resolution and installation) can be cancelled when the command
// context is cancelled.
func createPreflight(
	ctx context.Context,
	cfg *CLIConfig,
	confProvider *config.Provider,
	commandName string,
	s *styles.Styles,
	headless bool,
	jsonMode bool,
) tea.Model {
	if cfg.PreflightFactory == nil {
		return nil
	}
	skipCheck, _ := confProvider.GetBool("skipPluginCheck")
	if skipCheck {
		return nil
	}
	return cfg.PreflightFactory.CreatePreflight(
		ctx, confProvider, commandName, s, headless, os.Stdout, jsonMode,
	)
}

// CLIConfig holds the configuration that differentiates one CLI from another
// when using shared deployment command factories.
type CLIConfig struct {
	// CLIName is used in help text and log file naming (e.g. "celerity", "bluelink").
	CLIName string

	// EnvVarPrefix is prepended to environment variable names for config binding
	// (e.g. "CELERITY_CLI", "BLUELINK_CLI").
	EnvVarPrefix string

	// DefaultBlueprintFile is the default blueprint file name
	// (e.g. "app.blueprint.yaml", "project.blueprint.yaml").
	DefaultBlueprintFile string

	// DefaultDeployConfig is the default deploy configuration file name
	// (e.g. "celerity.deploy.json", "bluelink.deploy.json").
	DefaultDeployConfig string

	// DefaultConfigFile is the default CLI configuration file name
	// (e.g. "celerity.config.toml", "bluelink.config.toml").
	DefaultConfigFile string

	// Palette provides the color scheme for TUI styling.
	Palette styles.ColorPalette

	// PreflightFactory optionally creates a preflight check model
	// (e.g. plugin dependency verification). Nil to skip preflight checks.
	PreflightFactory PreflightFactory

	// PreCommandStep is called before stage/deploy commands interact with
	// the deploy engine. For example, Celerity uses this to trigger a build
	// step and inject build artifact context variables.
	// Nil to skip pre-command steps.
	PreCommandStep precommand.Step

	// EnableCodeOnlyApproval enables the --auto-approve-code-only flag on deploy.
	// When true, the flag is registered and the code-only approval logic is available.
	// This is intended for CLIs that use transformer plugins with resource category annotations.
	EnableCodeOnlyApproval bool
}

// PreflightFactory creates a preflight check TUI model for plugin dependency
// verification or other pre-command checks.
type PreflightFactory interface {
	// CreatePreflight creates a bubbletea model for preflight checks.
	// Returns nil if no preflight checks are needed. The context should be
	// bound to any long-running work the preflight model performs (e.g. plugin
	// dependency resolution/installation) so it is cancellable.
	CreatePreflight(
		ctx context.Context,
		confProvider *config.Provider,
		commandName string,
		styles *styles.Styles,
		headless bool,
		writer io.Writer,
		jsonMode bool,
	) tea.Model
}
