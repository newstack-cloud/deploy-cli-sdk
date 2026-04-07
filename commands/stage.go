package commands

import (
	"errors"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/newstack-cloud/deploy-cli-sdk/jsonout"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/stageui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var errStagingFailed = errors.New("staging failed")

type stageFlags struct {
	blueprintFile          string
	isDefaultBlueprintFile bool
	instanceID             string
	instanceIDIsDefault    bool
	instanceName           string
	instanceNameIsDefault  bool
	destroy                bool
	skipDriftCheck         bool
	jsonMode               bool
}

func readStageFlags(confProvider *config.Provider) stageFlags {
	blueprintFile, isDefault := confProvider.GetString("stageBlueprintFile")
	instanceID, instanceIDIsDefault := confProvider.GetString("stageInstanceID")
	instanceName, instanceNameIsDefault := confProvider.GetString("stageInstanceName")
	destroy, _ := confProvider.GetBool("stageDestroy")
	skipDriftCheck, _ := confProvider.GetBool("stageSkipDriftCheck")
	jsonMode, _ := confProvider.GetBool("stageJson")

	return stageFlags{
		blueprintFile:          blueprintFile,
		isDefaultBlueprintFile: isDefault,
		instanceID:             instanceID,
		instanceIDIsDefault:    instanceIDIsDefault,
		instanceName:           instanceName,
		instanceNameIsDefault:  instanceNameIsDefault,
		destroy:                destroy,
		skipDriftCheck:         skipDriftCheck,
		jsonMode:               jsonMode,
	}
}

func validateStageFlags(flags stageFlags) error {
	if !flags.destroy {
		return nil
	}
	return headless.Validate(
		headless.OneOf(
			headless.Flag{
				Name:      flagInstanceName,
				Value:     flags.instanceName,
				IsDefault: flags.instanceNameIsDefault,
			},
			headless.Flag{
				Name:      flagInstanceID,
				Value:     flags.instanceID,
				IsDefault: flags.instanceIDIsDefault,
			},
		),
	)
}

func runStageTUI(
	cmd *cobra.Command,
	flags stageFlags,
	cfg *CLIConfig,
	confProvider *config.Provider,
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
) error {
	if _, err := tea.LogToFile(fmt.Sprintf("%s-output.log", cfg.CLIName), "simple"); err != nil {
		log.Fatal(err)
	}

	cmd.SilenceUsage = true

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		cfg.Palette,
	)
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || flags.jsonMode

	if cfg.PreCommandStep != nil {
		if err := RunPreCommandStep(cfg.PreCommandStep, confProvider, "stage", styles, headlessMode, os.Stdout); err != nil {
			return err
		}
	}

	preflightModel := createPreflight(cfg, confProvider, "stage", styles, headlessMode, flags.jsonMode)

	app, err := stageui.NewStageApp(stageui.StageAppConfig{
		DeployEngine:           deployEngine,
		Logger:                 logger,
		BlueprintFile:          flags.blueprintFile,
		IsDefaultBlueprintFile: flags.isDefaultBlueprintFile,
		InstanceID:             flags.instanceID,
		InstanceName:           flags.instanceName,
		Destroy:                flags.destroy,
		SkipDriftCheck:         flags.skipDriftCheck,
		Styles:                 styles,
		Headless:               headlessMode,
		HeadlessWriter:         os.Stdout,
		JSONMode:               flags.jsonMode,
		Preflight:              preflightModel,
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(app, newTUIProgramOptions(headlessMode)...).Run()
	if err != nil {
		return err
	}
	finalApp := finalModel.(stageui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errStagingFailed
	}

	return nil
}

// SetupStageCommand registers a stage command on the root command,
// parameterized by CLIConfig for branding and defaults.
func SetupStageCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	stageCmd := &cobra.Command{
		Use:   "stage",
		Short: "Stage changes for a blueprint deployment",
		Long: fmt.Sprintf(`Creates a changeset by computing the differences between a blueprint
and the current state of an existing instance (or empty state for new deployments).

The changeset can then be applied using the deploy command.

Examples:
  # Stage changes for a new deployment
  %[1]s stage

  # Stage changes for an existing instance by name
  %[1]s stage --instance-name my-app

  # Stage changes with JSON output
  %[1]s stage --instance-name my-app --json

  # Stage changes for an existing instance by ID
  %[1]s stage --instance-id abc123

  # Stage changes for destroying an instance
  %[1]s stage --instance-name my-app --destroy`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, handle, err := SetupLogger(cfg.CLIName)
			if err != nil {
				return err
			}
			defer handle.Close()

			deployEngine, err := engine.Create(confProvider, logger)
			if err != nil {
				return err
			}

			flags := readStageFlags(confProvider)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateStageFlags(flags); err != nil {
				if flags.jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errStagingFailed
				}
				return err
			}

			return runStageTUI(cmd, flags, cfg, confProvider, deployEngine, logger)
		},
	}

	prefix := cfg.EnvVarPrefix

	stageCmd.PersistentFlags().String(
		"blueprint-file", cfg.DefaultBlueprintFile,
		"The blueprint file to stage. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("stageBlueprintFile", stageCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("stageBlueprintFile", prefix+"_STAGE_BLUEPRINT_FILE")

	stageCmd.PersistentFlags().String(
		flagInstanceID, "",
		"The ID of an existing blueprint instance to stage changes for. "+
			"If not provided and --instance-name is not provided, changes will be staged for a new deployment.",
	)
	confProvider.BindPFlag("stageInstanceID", stageCmd.PersistentFlags().Lookup(flagInstanceID))
	confProvider.BindEnvVar("stageInstanceID", prefix+"_STAGE_INSTANCE_ID")

	stageCmd.PersistentFlags().String(
		flagInstanceName, "",
		"The user-defined name of an existing blueprint instance to stage changes for. "+
			"If not provided and --instance-id is not provided, changes will be staged for a new deployment.",
	)
	confProvider.BindPFlag("stageInstanceName", stageCmd.PersistentFlags().Lookup(flagInstanceName))
	confProvider.BindEnvVar("stageInstanceName", prefix+"_STAGE_INSTANCE_NAME")

	stageCmd.PersistentFlags().Bool("destroy", false,
		"Stage changes for destroying an existing instance. "+
			"Requires --instance-id or --instance-name to be provided.",
	)
	confProvider.BindPFlag("stageDestroy", stageCmd.PersistentFlags().Lookup("destroy"))
	confProvider.BindEnvVar("stageDestroy", prefix+"_STAGE_DESTROY")

	stageCmd.PersistentFlags().Bool("skip-drift-check", false,
		"Skip detection of external resource changes during staging.",
	)
	confProvider.BindPFlag("stageSkipDriftCheck", stageCmd.PersistentFlags().Lookup("skip-drift-check"))
	confProvider.BindEnvVar("stageSkipDriftCheck", prefix+"_STAGE_SKIP_DRIFT_CHECK")

	stageCmd.PersistentFlags().Bool("json", false,
		"Output result as a single JSON object when the operation completes. "+
			"Implies non-interactive mode (no TUI, no streaming text output).",
	)
	confProvider.BindPFlag("stageJson", stageCmd.PersistentFlags().Lookup("json"))
	confProvider.BindEnvVar("stageJson", prefix+"_STAGE_JSON")

	rootCmd.AddCommand(stageCmd)
}
