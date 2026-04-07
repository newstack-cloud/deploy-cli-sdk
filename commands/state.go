package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/newstack-cloud/deploy-cli-sdk/jsonout"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stateexportui"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stateimportui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errStateImportFailed = errors.New("state import failed")
var errStateExportFailed = errors.New("state export failed")

// SetupStateCommand registers a state command with import and export
// subcommands on the root command, parameterized by CLIConfig for branding.
func SetupStateCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "Manage deploy engine state",
		Long:  `Commands for managing deploy engine state, including import and export operations.`,
	}

	prefix := cfg.EnvVarPrefix

	stateCmd.PersistentFlags().String(
		"engine-config-file", "",
		"Path to deploy engine config file. Used to determine storage backend.",
	)
	confProvider.BindPFlag("stateEngineConfigFile", stateCmd.PersistentFlags().Lookup("engine-config-file"))
	confProvider.BindEnvVar("stateEngineConfigFile", prefix+"_STATE_ENGINE_CONFIG_FILE")

	setupStateImportCommand(stateCmd, confProvider, cfg)
	setupStateExportCommand(stateCmd, confProvider, cfg)

	rootCmd.AddCommand(stateCmd)
}

type stateImportFlags struct {
	filePath          string
	filePathIsDefault bool
	engineConfigFile  string
	jsonMode          bool
}

func readStateImportFlags(confProvider *config.Provider) stateImportFlags {
	filePath, filePathIsDefault := confProvider.GetString("stateImportFile")
	engineConfigFile, _ := confProvider.GetString("stateEngineConfigFile")
	jsonMode, _ := confProvider.GetBool("stateImportJson")

	return stateImportFlags{
		filePath:          filePath,
		filePathIsDefault: filePathIsDefault,
		engineConfigFile:  engineConfigFile,
		jsonMode:          jsonMode,
	}
}

func validateStateImportFlags(flags stateImportFlags) error {
	if flags.jsonMode && (flags.filePathIsDefault || flags.filePath == "") {
		return fmt.Errorf("--file is required when --json is set")
	}
	return headless.Validate(
		headless.Required(headless.Flag{
			Name:      "file",
			Value:     flags.filePath,
			IsDefault: flags.filePathIsDefault,
		}),
	)
}

func loadEngineConfig(engineConfigFile string) (*stateio.EngineConfig, error) {
	path := engineConfigFile
	if path == "" {
		path = stateio.GetDefaultEngineConfigPath()
	}
	cfg, err := stateio.LoadEngineConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load engine config: %w", err)
	}
	return cfg, nil
}

func runStateImportTUI(cmd *cobra.Command, flags stateImportFlags, cfg *CLIConfig) error {
	engineConfig, err := loadEngineConfig(flags.engineConfigFile)
	if err != nil {
		return err
	}

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		cfg.Palette,
	)

	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || flags.jsonMode
	app, err := stateimportui.NewStateImportApp(stateimportui.StateImportAppConfig{
		FilePath:       flags.filePath,
		EngineConfig:   engineConfig,
		Styles:         styles,
		Headless:       headlessMode,
		HeadlessWriter: os.Stdout,
		JSONMode:       flags.jsonMode,
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(app, newTUIProgramOptions(headlessMode)...).Run()
	if err != nil {
		return err
	}
	finalApp := finalModel.(stateimportui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errStateImportFailed
	}

	return nil
}

func setupStateImportCommand(stateCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import state from a file",
		Long: fmt.Sprintf(`Import deploy engine state from a local file or remote object storage.

The input file must be a JSON array of blueprint instances. This format is
backend-agnostic and works with any storage backend (memfile, PostgreSQL, etc.).

Examples:
  # Import state from a local file
  %[1]s state import --file ./backup/state.json

  # Import from S3
  %[1]s state import --file s3://my-bucket/state.json

  # Import from GCS
  %[1]s state import --file gcs://my-bucket/state.json

  # Import from Azure Blob Storage
  %[1]s state import --file azureblob://my-container/state.json

  # Use deploy engine config to determine storage backend (flag inherited from state command)
  %[1]s state --engine-config-file ~/.config/engine/config.json import --file ./state.json`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			flags := readStateImportFlags(confProvider)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateStateImportFlags(flags); err != nil {
				if flags.jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errStateImportFailed
				}
				return err
			}

			return runStateImportTUI(cmd, flags, cfg)
		},
	}

	prefix := cfg.EnvVarPrefix

	importCmd.Flags().String(
		"file", "",
		"Path to input file. Can be local or remote (s3://, gcs://, azureblob://).",
	)
	confProvider.BindPFlag("stateImportFile", importCmd.Flags().Lookup("file"))
	confProvider.BindEnvVar("stateImportFile", prefix+"_STATE_IMPORT_FILE")

	importCmd.Flags().Bool("json", false,
		"Output result as JSON (for headless/CI mode).",
	)
	confProvider.BindPFlag("stateImportJson", importCmd.Flags().Lookup("json"))
	confProvider.BindEnvVar("stateImportJson", prefix+"_STATE_IMPORT_JSON")

	stateCmd.AddCommand(importCmd)
}

type stateExportFlags struct {
	filePath          string
	filePathIsDefault bool
	engineConfigFile  string
	instanceFilters   []string
	jsonMode          bool
}

func readStateExportFlags(confProvider *config.Provider) stateExportFlags {
	filePath, filePathIsDefault := confProvider.GetString("stateExportFile")
	engineConfigFile, _ := confProvider.GetString("stateEngineConfigFile")
	instancesFlag, _ := confProvider.GetString("stateExportInstances")
	jsonMode, _ := confProvider.GetBool("stateExportJson")

	var instanceFilters []string
	if instancesFlag != "" {
		for _, inst := range strings.Split(instancesFlag, ",") {
			trimmed := strings.TrimSpace(inst)
			if trimmed != "" {
				instanceFilters = append(instanceFilters, trimmed)
			}
		}
	}

	return stateExportFlags{
		filePath:          filePath,
		filePathIsDefault: filePathIsDefault,
		engineConfigFile:  engineConfigFile,
		instanceFilters:   instanceFilters,
		jsonMode:          jsonMode,
	}
}

func validateStateExportFlags(flags stateExportFlags) error {
	if flags.jsonMode && (flags.filePathIsDefault || flags.filePath == "") {
		return fmt.Errorf("--file is required when --json is set")
	}
	return headless.Validate(
		headless.Required(headless.Flag{
			Name:      "file",
			Value:     flags.filePath,
			IsDefault: flags.filePathIsDefault,
		}),
	)
}

func runStateExportTUI(cmd *cobra.Command, flags stateExportFlags, cfg *CLIConfig) error {
	engineConfig, err := loadEngineConfig(flags.engineConfigFile)
	if err != nil {
		return err
	}

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		cfg.Palette,
	)

	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || flags.jsonMode
	app, err := stateexportui.NewStateExportApp(stateexportui.StateExportAppConfig{
		FilePath:        flags.filePath,
		InstanceFilters: flags.instanceFilters,
		EngineConfig:    engineConfig,
		Styles:          styles,
		Headless:        headlessMode,
		HeadlessWriter:  os.Stdout,
		JSONMode:        flags.jsonMode,
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(app, newTUIProgramOptions(headlessMode)...).Run()
	if err != nil {
		return err
	}
	finalApp := finalModel.(stateexportui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errStateExportFailed
	}

	return nil
}

func setupStateExportCommand(stateCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export state to a file",
		Long: fmt.Sprintf(`Export deploy engine state to a local file or remote object storage.

The output file is a JSON array of blueprint instances. This format is
backend-agnostic and can be imported into any storage backend (memfile, PostgreSQL, etc.).

Examples:
  # Export all instances to a local file
  %[1]s state export --file ./backup/state.json

  # Export specific instances by name or ID
  %[1]s state export --file ./backup/state.json --instances my-stack,inst-abc123

  # Export to S3
  %[1]s state export --file s3://my-bucket/state.json

  # Export to GCS
  %[1]s state export --file gcs://my-bucket/state.json

  # Export to Azure Blob Storage
  %[1]s state export --file azureblob://my-container/state.json

  # Use deploy engine config to determine storage backend (flag inherited from state command)
  %[1]s state --engine-config-file ~/.config/engine/config.json export --file ./state.json`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			flags := readStateExportFlags(confProvider)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateStateExportFlags(flags); err != nil {
				if flags.jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errStateExportFailed
				}
				return err
			}

			return runStateExportTUI(cmd, flags, cfg)
		},
	}

	prefix := cfg.EnvVarPrefix

	exportCmd.Flags().String(
		"file", "",
		"Path to output file. Can be local or remote (s3://, gcs://, azureblob://).",
	)
	confProvider.BindPFlag("stateExportFile", exportCmd.Flags().Lookup("file"))
	confProvider.BindEnvVar("stateExportFile", prefix+"_STATE_EXPORT_FILE")

	exportCmd.Flags().String(
		"instances", "",
		"Comma-separated list of instance names or IDs to export (exports all if not set).",
	)
	confProvider.BindPFlag("stateExportInstances", exportCmd.Flags().Lookup("instances"))
	confProvider.BindEnvVar("stateExportInstances", prefix+"_STATE_EXPORT_INSTANCES")

	exportCmd.Flags().Bool("json", false,
		"Output result as JSON (for headless/CI mode).",
	)
	confProvider.BindPFlag("stateExportJson", exportCmd.Flags().Lookup("json"))
	confProvider.BindEnvVar("stateExportJson", prefix+"_STATE_EXPORT_JSON")

	stateCmd.AddCommand(exportCmd)
}
