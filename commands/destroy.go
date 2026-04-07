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
	"github.com/newstack-cloud/deploy-cli-sdk/tui/destroyui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var errDestroyFailed = errors.New("destroy failed")

type destroyFlags struct {
	changesetID            string
	changesetIDIsDefault   bool
	instanceID             string
	instanceIDIsDefault    bool
	instanceName           string
	instanceNameIsDefault  bool
	blueprintFile          string
	isDefaultBlueprintFile bool
	stageFirst             bool
	autoApprove            bool
	skipPrompts            bool
	force                  bool
	jsonMode               bool
}

func readDestroyFlags(confProvider *config.Provider) destroyFlags {
	changesetID, changesetIDIsDefault := confProvider.GetString("destroyChangeSetID")
	instanceID, instanceIDIsDefault := confProvider.GetString("destroyInstanceID")
	instanceName, instanceNameIsDefault := confProvider.GetString("destroyInstanceName")
	blueprintFile, isDefaultBlueprintFile := confProvider.GetString("destroyBlueprintFile")
	stageFirst, _ := confProvider.GetBool("destroyStage")
	autoApprove, _ := confProvider.GetBool("destroyAutoApprove")
	skipPrompts, _ := confProvider.GetBool("destroySkipPrompts")
	force, _ := confProvider.GetBool("destroyForce")
	jsonMode, _ := confProvider.GetBool("destroyJson")

	if jsonMode {
		autoApprove = true
	}

	return destroyFlags{
		changesetID:            changesetID,
		changesetIDIsDefault:   changesetIDIsDefault,
		instanceID:             instanceID,
		instanceIDIsDefault:    instanceIDIsDefault,
		instanceName:           instanceName,
		instanceNameIsDefault:  instanceNameIsDefault,
		blueprintFile:          blueprintFile,
		isDefaultBlueprintFile: isDefaultBlueprintFile,
		stageFirst:             stageFirst,
		autoApprove:            autoApprove,
		skipPrompts:            skipPrompts,
		force:                  force,
		jsonMode:               jsonMode,
	}
}

func validateDestroyFlags(flags destroyFlags) error {
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
		headless.OneOf(
			headless.Flag{
				Name:      "stage",
				Value:     boolToString(flags.stageFirst),
				IsDefault: !flags.stageFirst,
			},
			headless.Flag{
				Name:      flagChangeSetID,
				Value:     flags.changesetID,
				IsDefault: flags.changesetIDIsDefault,
			},
		),
		headless.RequiredIfBool(
			headless.BoolFlagTrue("stage", flags.stageFirst),
			flagAutoApprove,
			flags.autoApprove,
		),
	)
}

func runDestroyTUI(
	cmd *cobra.Command,
	flags destroyFlags,
	cfg *CLIConfig,
	confProvider *config.Provider,
	destroyEngine engine.DeployEngine,
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
		if err := RunPreCommandStep(cfg.PreCommandStep, confProvider, "destroy", styles, headlessMode, os.Stdout); err != nil {
			return err
		}
	}

	preflightModel := createPreflight(cfg, confProvider, "destroy", styles, headlessMode, flags.jsonMode)

	app, err := destroyui.NewDestroyApp(destroyui.DestroyAppConfig{
		DestroyEngine:          destroyEngine,
		Logger:                 logger,
		ChangesetID:            flags.changesetID,
		InstanceID:             flags.instanceID,
		InstanceName:           flags.instanceName,
		BlueprintFile:          flags.blueprintFile,
		IsDefaultBlueprintFile: flags.isDefaultBlueprintFile,
		Force:                  flags.force,
		StageFirst:             flags.stageFirst,
		AutoApprove:            flags.autoApprove,
		SkipPrompts:            flags.skipPrompts,
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
	finalApp := finalModel.(destroyui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errDestroyFailed
	}

	return nil
}

// SetupDestroyCommand registers a destroy command on the root command,
// parameterized by CLIConfig for branding and defaults.
func SetupDestroyCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a blueprint instance",
		Long: fmt.Sprintf(`Destroys a deployed blueprint instance, removing all associated resources,
child blueprints, and links.

The destruction streams events in real-time, allowing you to monitor progress
as resources are being destroyed.

Examples:
  # Interactive mode - select instance to destroy
  %[1]s destroy

  # Destroy with pre-selected instance using latest destroy change set
  %[1]s destroy --instance-name my-app

  # Destroy using a specific change set
  %[1]s destroy --instance-name my-app --change-set-id abc123

  # Stage destroy changes first, then execute with auto-approve
  %[1]s destroy --instance-name my-app --stage --auto-approve

  # Force destroy, overriding state conflicts
  %[1]s destroy --instance-name my-app --force`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, handle, err := SetupLogger(cfg.CLIName)
			if err != nil {
				return err
			}
			defer handle.Close()

			destroyEngine, err := engine.Create(confProvider, logger)
			if err != nil {
				return err
			}

			flags := readDestroyFlags(confProvider)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateDestroyFlags(flags); err != nil {
				if flags.jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errDestroyFailed
				}
				return err
			}

			return runDestroyTUI(cmd, flags, cfg, confProvider, destroyEngine, logger)
		},
	}

	prefix := cfg.EnvVarPrefix

	destroyCmd.PersistentFlags().String(
		flagChangeSetID, "",
		"The ID of the change set to use for destruction. "+
			"If not provided, the latest destroy change set for the instance will be used.",
	)
	confProvider.BindPFlag("destroyChangeSetID", destroyCmd.PersistentFlags().Lookup(flagChangeSetID))
	confProvider.BindEnvVar("destroyChangeSetID", prefix+"_DESTROY_CHANGE_SET_ID")

	destroyCmd.PersistentFlags().String(
		flagInstanceID, "",
		"The system-generated ID of the blueprint instance to destroy. "+
			"Leave empty if using --instance-name.",
	)
	confProvider.BindPFlag("destroyInstanceID", destroyCmd.PersistentFlags().Lookup(flagInstanceID))
	confProvider.BindEnvVar("destroyInstanceID", prefix+"_DESTROY_INSTANCE_ID")

	destroyCmd.PersistentFlags().String(
		flagInstanceName, "",
		"The user-defined unique identifier for the blueprint instance to destroy. "+
			"Leave empty if using --instance-id.",
	)
	confProvider.BindPFlag("destroyInstanceName", destroyCmd.PersistentFlags().Lookup(flagInstanceName))
	confProvider.BindEnvVar("destroyInstanceName", prefix+"_DESTROY_INSTANCE_NAME")

	destroyCmd.PersistentFlags().String(
		"blueprint-file", cfg.DefaultBlueprintFile,
		"The blueprint file for staging destroy changes. "+
			"Only used when --stage is set. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("destroyBlueprintFile", destroyCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("destroyBlueprintFile", prefix+"_DESTROY_BLUEPRINT_FILE")

	destroyCmd.PersistentFlags().Bool("force", false,
		"Override state conflicts and force destruction.",
	)
	confProvider.BindPFlag("destroyForce", destroyCmd.PersistentFlags().Lookup("force"))
	confProvider.BindEnvVar("destroyForce", prefix+"_DESTROY_FORCE")

	destroyCmd.PersistentFlags().Bool("stage", false,
		"Stage destroy changes and review them before execution. "+
			"When set, the CLI will first run the change staging process to show "+
			"what changes will be applied, allowing you to review and confirm before destroying.",
	)
	confProvider.BindPFlag("destroyStage", destroyCmd.PersistentFlags().Lookup("stage"))
	confProvider.BindEnvVar("destroyStage", prefix+"_DESTROY_STAGE")

	destroyCmd.PersistentFlags().Bool(flagAutoApprove, false,
		"Automatically approve staged changes without prompting for confirmation. "+
			"This is intended for CI/CD pipelines where manual approval is not possible. "+
			"Only applicable when --stage is set.",
	)
	confProvider.BindPFlag("destroyAutoApprove", destroyCmd.PersistentFlags().Lookup(flagAutoApprove))
	confProvider.BindEnvVar("destroyAutoApprove", prefix+"_DESTROY_AUTO_APPROVE")

	destroyCmd.PersistentFlags().Bool("skip-prompts", false,
		"Skip interactive prompts and use flag values directly. "+
			"Requires all necessary flags to be provided (--instance-name or --instance-id, "+
			"and either --stage or --change-set-id).",
	)
	confProvider.BindPFlag("destroySkipPrompts", destroyCmd.PersistentFlags().Lookup("skip-prompts"))
	confProvider.BindEnvVar("destroySkipPrompts", prefix+"_DESTROY_SKIP_PROMPTS")

	destroyCmd.PersistentFlags().Bool("json", false,
		"Output result as a single JSON object when the operation completes. "+
			"Implies non-interactive mode (no TUI, no streaming text output).",
	)
	confProvider.BindPFlag("destroyJson", destroyCmd.PersistentFlags().Lookup("json"))

	rootCmd.AddCommand(destroyCmd)
}
