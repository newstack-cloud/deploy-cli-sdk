package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
	"github.com/spf13/pflag"
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
	force             bool
	fileKey           stateio.FileEncryptionKeyOptions
}

// --force and the key flags are read from the command line alone (see addStateFileKeyFlags).
func readStateImportFlags(
	confProvider *config.Provider,
	cmdFlags *pflag.FlagSet,
	envVarPrefix string,
) stateImportFlags {
	filePath, filePathIsDefault := confProvider.GetString("stateImportFile")
	engineConfigFile, _ := confProvider.GetString("stateEngineConfigFile")
	jsonMode, _ := confProvider.GetBool("stateImportJson")
	force, _ := cmdFlags.GetBool("force")
	fileKey := readStateFileKeyOptions(cmdFlags, envVarPrefix)

	return stateImportFlags{
		filePath:          filePath,
		filePathIsDefault: filePathIsDefault,
		engineConfigFile:  engineConfigFile,
		jsonMode:          jsonMode,
		force:             force,
		fileKey:           fileKey,
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
		found, err := stateio.FindEngineConfigFile()
		if err != nil {
			return nil, fmt.Errorf("failed to load engine config: %w", err)
		}
		if found == "" {
			return stateio.LoadEngineConfigFromEnv()
		}
		path = found
	}
	cfg, err := stateio.LoadEngineConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load engine config: %w", err)
	}
	return cfg, nil
}

func runStateImportTUI(
	cmd *cobra.Command,
	flags stateImportFlags,
	fileKey *stateio.StateEncryptionKeyConfig,
	cfg *CLIConfig,
) error {
	engineConfig, err := loadEngineConfig(flags.engineConfigFile)
	if err != nil {
		return handleStateFlagError(err, flags.jsonMode, errStateImportFailed)
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
		FileEncryption: fileKey,
		// A key given on the command line is for this file, while the passphrase in the
		// environment may be set for every import in a pipeline, plaintext ones included.
		RequireEncryptedFile: flags.fileKey.Key != "" || flags.fileKey.PassphraseFile != "",
		Force:                flags.force,
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(app, newTUIProgramOptions(cmd.Context(), headlessMode)...).Run()
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

// Formatted with the CLI's name, then its environment variable prefix.
const stateImportHelpTemplate = `Import deploy engine state from a local file or remote object storage.

The input file must be a JSON array of blueprint instances, or a file encrypted
by "state export --encrypt". This format is backend-agnostic and works with any
storage backend (memfile, PostgreSQL, object storage).

Nothing is imported if any instance in the file would rename an existing instance,
take a name another instance already has, or overwrite an instance with an
operation in progress. Use --force to import over an instance whose deployment
was lost or abandoned without its status being updated.

Encrypted files are detected automatically. Provide the key they were encrypted
with using --key, --passphrase-file or the %[2]s_STATE_FILE_PASSPHRASE
environment variable.

Examples:
  # Import state from a local file
  %[1]s state import --file ./backup/state.json

  # Import from S3
  %[1]s state import --file s3://my-bucket/state.json

  # Import from GCS
  %[1]s state import --file gcs://my-bucket/state.json

  # Import from Azure Blob Storage
  %[1]s state import --file azureblob://my-container/state.json

  # Import a file encrypted with an AWS KMS key
  %[1]s state import --file s3://my-bucket/state.json --key awskms:alias/state-export

  # Import a file encrypted with a passphrase
  %[1]s state import --file ./state.json --passphrase-file ./passphrase.txt

  # Use deploy engine config to determine storage backend (flag inherited from state command)
  %[1]s state --engine-config-file ~/.config/engine/config.json import --file ./state.json`

func stateImportLongHelp(cfg *CLIConfig) string {
	return fmt.Sprintf(stateImportHelpTemplate, cfg.CLIName, cfg.EnvVarPrefix)
}

func setupStateImportCommand(stateCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import state from a file",
		Long:  stateImportLongHelp(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			flags := readStateImportFlags(confProvider, cmd.Flags(), cfg.EnvVarPrefix)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateStateImportFlags(flags); err != nil {
				return handleStateFlagError(err, flags.jsonMode, errStateImportFailed)
			}

			fileKey, err := stateio.ResolveFileEncryptionKey(flags.fileKey)
			if err != nil {
				return handleStateFlagError(err, flags.jsonMode, errStateImportFailed)
			}

			return runStateImportTUI(cmd, flags, fileKey, cfg)
		},
	}

	addStateImportFlags(importCmd, confProvider, cfg.EnvVarPrefix)

	stateCmd.AddCommand(importCmd)
}

func addStateImportFlags(importCmd *cobra.Command, confProvider *config.Provider, prefix string) {
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

	addStateFileKeyFlags(importCmd)

	// Read force from the command line alone, not an environment variable or the config file:
	// overriding the check should be a deliberate choice for one import, not a setting
	// that applies to every run of a pipeline or project.
	importCmd.Flags().Bool("force", false,
		"Import over instances that have an operation in progress. Only for a deployment that "+
			"was lost or abandoned without its status being updated.",
	)
}

type stateExportFlags struct {
	filePath          string
	filePathIsDefault bool
	engineConfigFile  string
	instanceFilters   []string
	jsonMode          bool
	encrypt           bool
	// Set when the value given for --encrypt, or its environment variable or config file
	// setting, is not a boolean. It is reported rather than read as false, which would
	// write a plaintext file when encryption was asked for.
	encryptErr error
	fileKey    stateio.FileEncryptionKeyOptions
}

func readStateExportFlags(
	confProvider *config.Provider,
	cmdFlags *pflag.FlagSet,
	envVarPrefix string,
) stateExportFlags {
	filePath, filePathIsDefault := confProvider.GetString("stateExportFile")
	engineConfigFile, _ := confProvider.GetString("stateEngineConfigFile")
	instancesFlag, _ := confProvider.GetString("stateExportInstances")
	jsonMode, _ := confProvider.GetBool("stateExportJson")
	encrypt, encryptErr := readEncryptSetting(confProvider)

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
		encrypt:           encrypt,
		encryptErr:        encryptErr,
		fileKey:           readStateFileKeyOptions(cmdFlags, envVarPrefix),
	}
}

func readEncryptSetting(confProvider *config.Provider) (bool, error) {
	value, _ := confProvider.GetString("stateExportEncrypt")
	if value == "" {
		return false, nil
	}
	encrypt, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("--encrypt must be true or false, got %q", value)
	}
	return encrypt, nil
}

func validateStateExportFlags(flags stateExportFlags) error {
	if flags.encryptErr != nil {
		return flags.encryptErr
	}
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

func runStateExportTUI(
	cmd *cobra.Command,
	flags stateExportFlags,
	fileKey *stateio.StateEncryptionKeyConfig,
	cfg *CLIConfig,
) error {
	engineConfig, err := loadEngineConfig(flags.engineConfigFile)
	if err != nil {
		return handleStateFlagError(err, flags.jsonMode, errStateExportFailed)
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
		FileEncryption:  fileKey,
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(app, newTUIProgramOptions(cmd.Context(), headlessMode)...).Run()
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

// Formatted with the CLI's name, then its environment variable prefix.
const stateExportHelpTemplate = `Export deploy engine state to a local file or remote object storage.

The output file is a JSON array of blueprint instances. This format is
backend-agnostic and can be imported into any storage backend (memfile, PostgreSQL, object storage).

Exported state can hold resolved secrets. Use --encrypt to encrypt the file, with
the key given by --key, --passphrase-file or the %[2]s_STATE_FILE_PASSPHRASE
environment variable.

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

  # Export an encrypted file with an AWS KMS key
  %[1]s state export --file s3://my-bucket/state.json --encrypt --key awskms:alias/state-export

  # Export an encrypted file with a passphrase from the environment
  %[2]s_STATE_FILE_PASSPHRASE=... %[1]s state export --file ./state.json --encrypt

  # Use deploy engine config to determine storage backend (flag inherited from state command)
  %[1]s state --engine-config-file ~/.config/engine/config.json export --file ./state.json`

func stateExportLongHelp(cfg *CLIConfig) string {
	return fmt.Sprintf(stateExportHelpTemplate, cfg.CLIName, cfg.EnvVarPrefix)
}

func setupStateExportCommand(stateCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export state to a file",
		Long:  stateExportLongHelp(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			flags := readStateExportFlags(confProvider, cmd.Flags(), cfg.EnvVarPrefix)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateStateExportFlags(flags); err != nil {
				return handleStateFlagError(err, flags.jsonMode, errStateExportFailed)
			}

			fileKey, err := resolveExportFileKey(flags.encrypt, flags.fileKey, cfg.EnvVarPrefix)
			if err != nil {
				return handleStateFlagError(err, flags.jsonMode, errStateExportFailed)
			}

			return runStateExportTUI(cmd, flags, fileKey, cfg)
		},
	}

	addStateExportFlags(exportCmd, confProvider, cfg.EnvVarPrefix)

	stateCmd.AddCommand(exportCmd)
}

func addStateExportFlags(exportCmd *cobra.Command, confProvider *config.Provider, prefix string) {
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

	exportCmd.Flags().Bool("encrypt", false,
		"Encrypt the output file with the key given by --key, --passphrase-file "+
			"or the passphrase environment variable.",
	)
	confProvider.BindPFlag("stateExportEncrypt", exportCmd.Flags().Lookup("encrypt"))
	confProvider.BindEnvVar("stateExportEncrypt", prefix+"_STATE_EXPORT_ENCRYPT")

	addStateFileKeyFlags(exportCmd)
}

func handleStateFlagError(err error, jsonMode bool, failure error) error {
	if jsonMode {
		jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
		return failure
	}
	return err
}
