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
	"github.com/newstack-cloud/deploy-cli-sdk/tui/deployui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var errDeploymentFailed = errors.New("deployment failed")

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return ""
}

type deployFlags struct {
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
	autoApproveCodeOnly    bool
	skipPrompts            bool
	autoRollback           bool
	force                  bool
	jsonMode               bool
}

func readDeployFlags(confProvider *config.Provider, cfg *CLIConfig) deployFlags {
	changesetID, changesetIDIsDefault := confProvider.GetString("deployChangeSetID")
	instanceID, instanceIDIsDefault := confProvider.GetString("deployInstanceID")
	instanceName, instanceNameIsDefault := confProvider.GetString("deployInstanceName")
	blueprintFile, isDefault := confProvider.GetString("deployBlueprintFile")
	stageFirst, _ := confProvider.GetBool("deployStage")
	autoApprove, _ := confProvider.GetBool("deployAutoApprove")
	skipPrompts, _ := confProvider.GetBool("deploySkipPrompts")
	autoRollback, _ := confProvider.GetBool("deployAutoRollback")
	force, _ := confProvider.GetBool("deployForce")
	jsonMode, _ := confProvider.GetBool("deployJson")

	var autoApproveCodeOnly bool
	if cfg.EnableCodeOnlyApproval {
		autoApproveCodeOnly, _ = confProvider.GetBool("deployAutoApproveCodeOnly")
	}

	if jsonMode {
		autoApprove = true
	}

	return deployFlags{
		changesetID:            changesetID,
		changesetIDIsDefault:   changesetIDIsDefault,
		instanceID:             instanceID,
		instanceIDIsDefault:    instanceIDIsDefault,
		instanceName:           instanceName,
		instanceNameIsDefault:  instanceNameIsDefault,
		blueprintFile:          blueprintFile,
		isDefaultBlueprintFile: isDefault,
		stageFirst:             stageFirst,
		autoApprove:            autoApprove,
		autoApproveCodeOnly:    autoApproveCodeOnly,
		skipPrompts:            skipPrompts,
		autoRollback:           autoRollback,
		force:                  force,
		jsonMode:               jsonMode,
	}
}

func validateDeployFlags(flags deployFlags) error {
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
			"auto-approve or auto-approve-code-only",
			flags.autoApprove || flags.autoApproveCodeOnly,
		),
	)
}

func runDeployTUI(
	cmd *cobra.Command,
	flags deployFlags,
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
		if err := RunPreCommandStep(cfg.PreCommandStep, confProvider, "deploy", styles, headlessMode, os.Stdout); err != nil {
			return err
		}
	}

	preflightModel := createPreflight(cfg, confProvider, "deploy", styles, headlessMode, flags.jsonMode)

	app, err := deployui.NewDeployApp(deployui.DeployAppConfig{
		DeployEngine:           deployEngine,
		Logger:                 logger,
		ChangesetID:            flags.changesetID,
		InstanceID:             flags.instanceID,
		InstanceName:           flags.instanceName,
		BlueprintFile:          flags.blueprintFile,
		IsDefaultBlueprintFile: flags.isDefaultBlueprintFile,
		AutoRollback:           flags.autoRollback,
		Force:                  flags.force,
		StageFirst:             flags.stageFirst,
		AutoApprove:            flags.autoApprove,
		AutoApproveCodeOnly:    flags.autoApproveCodeOnly,
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
	finalApp := finalModel.(deployui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errDeploymentFailed
	}

	return nil
}

// SetupDeployCommand registers a deploy command on the root command,
// parameterized by CLIConfig for branding and defaults.
func SetupDeployCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a blueprint instance",
		Long: fmt.Sprintf(`Executes a change set for a blueprint instance, supporting both new
deployments and updates to existing instances.

The deployment streams events in real-time, allowing you to monitor progress
of resources, child blueprints, and links as they are deployed.

Examples:
  # Interactive mode - select blueprint and instance
  %[1]s deploy

  # Deploy with pre-selected instance using latest change set
  %[1]s deploy --instance-name my-app

  # Deploy specific change set
  %[1]s deploy --instance-name my-app --change-set-id abc123

  # Deploy from a specific blueprint file
  %[1]s deploy --blueprint-file ./%[2]s --instance-name my-app

  # Deploy with auto-rollback enabled
  %[1]s deploy --instance-name my-app --auto-rollback`, cfg.CLIName, cfg.DefaultBlueprintFile),
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

			flags := readDeployFlags(confProvider, cfg)

			if flags.jsonMode {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			if err := validateDeployFlags(flags); err != nil {
				if flags.jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errDeploymentFailed
				}
				return err
			}

			return runDeployTUI(cmd, flags, cfg, confProvider, deployEngine, logger)
		},
	}

	prefix := cfg.EnvVarPrefix

	deployCmd.PersistentFlags().String(
		flagChangeSetID, "",
		"The ID of the change set to deploy. "+
			"If not provided, the latest change set for the instance will be used.",
	)
	confProvider.BindPFlag("deployChangeSetID", deployCmd.PersistentFlags().Lookup(flagChangeSetID))
	confProvider.BindEnvVar("deployChangeSetID", prefix+"_DEPLOY_CHANGE_SET_ID")

	deployCmd.PersistentFlags().String(
		flagInstanceID, "",
		"The system-generated ID of the blueprint instance to deploy to. "+
			"Leave empty if using --instance-name or for new deployments.",
	)
	confProvider.BindPFlag("deployInstanceID", deployCmd.PersistentFlags().Lookup(flagInstanceID))
	confProvider.BindEnvVar("deployInstanceID", prefix+"_DEPLOY_INSTANCE_ID")

	deployCmd.PersistentFlags().String(
		flagInstanceName, "",
		"The user-defined unique identifier for the target blueprint instance. "+
			"Leave empty if using --instance-id or for new deployments.",
	)
	confProvider.BindPFlag("deployInstanceName", deployCmd.PersistentFlags().Lookup(flagInstanceName))
	confProvider.BindEnvVar("deployInstanceName", prefix+"_DEPLOY_INSTANCE_NAME")

	deployCmd.PersistentFlags().String(
		"blueprint-file", cfg.DefaultBlueprintFile,
		"The blueprint file for runtime substitution resolution. "+
			"This can be a local file, a public URL or a path to a file in an object storage bucket. "+
			"Local files can be specified as a relative or absolute path to the file. "+
			"Public URLs must start with https:// and represent a valid URL to a blueprint file. "+
			"Object storage bucket files must be specified in the format of {scheme}://{bucket-name}/{object-path}, "+
			"where {scheme} is one of the following: s3, gcs, azureblob.",
	)
	confProvider.BindPFlag("deployBlueprintFile", deployCmd.PersistentFlags().Lookup("blueprint-file"))
	confProvider.BindEnvVar("deployBlueprintFile", prefix+"_DEPLOY_BLUEPRINT_FILE")

	deployCmd.PersistentFlags().Bool("auto-rollback", false,
		"Automatically rollback on deployment failure.",
	)
	confProvider.BindPFlag("deployAutoRollback", deployCmd.PersistentFlags().Lookup("auto-rollback"))
	confProvider.BindEnvVar("deployAutoRollback", prefix+"_DEPLOY_AUTO_ROLLBACK")

	deployCmd.PersistentFlags().Bool("force", false,
		"Override state conflicts and force deployment.",
	)
	confProvider.BindPFlag("deployForce", deployCmd.PersistentFlags().Lookup("force"))
	confProvider.BindEnvVar("deployForce", prefix+"_DEPLOY_FORCE")

	deployCmd.PersistentFlags().Bool("stage", false,
		"Stage changes and review them before deployment. "+
			"When set, the CLI will first run the change staging process to show "+
			"what changes will be applied, allowing you to review and confirm before deploying.",
	)
	confProvider.BindPFlag("deployStage", deployCmd.PersistentFlags().Lookup("stage"))
	confProvider.BindEnvVar("deployStage", prefix+"_DEPLOY_STAGE")

	deployCmd.PersistentFlags().Bool(flagAutoApprove, false,
		"Automatically approve staged changes without prompting for confirmation. "+
			"This is intended for CI/CD pipelines where manual approval is not possible. "+
			"Only applicable when --stage is set.",
	)
	confProvider.BindPFlag("deployAutoApprove", deployCmd.PersistentFlags().Lookup(flagAutoApprove))
	confProvider.BindEnvVar("deployAutoApprove", prefix+"_DEPLOY_AUTO_APPROVE")

	if cfg.EnableCodeOnlyApproval {
		deployCmd.PersistentFlags().Bool("auto-approve-code-only", false,
			"Automatically approve staged changes when only code-hosting resources are modified. "+
				"Requires --stage. Denied when any creates, deletes, or infrastructure changes are present. "+
				"In non-interactive mode, exits with an error if approval is denied.",
		)
		confProvider.BindPFlag("deployAutoApproveCodeOnly", deployCmd.PersistentFlags().Lookup("auto-approve-code-only"))
		confProvider.BindEnvVar("deployAutoApproveCodeOnly", prefix+"_DEPLOY_AUTO_APPROVE_CODE_ONLY")
	}

	deployCmd.PersistentFlags().Bool("skip-prompts", false,
		"Skip interactive prompts and use flag values directly. "+
			"Requires all necessary flags to be provided (--instance-name or --instance-id, "+
			"and either --stage or --change-set-id).",
	)
	confProvider.BindPFlag("deploySkipPrompts", deployCmd.PersistentFlags().Lookup("skip-prompts"))
	confProvider.BindEnvVar("deploySkipPrompts", prefix+"_DEPLOY_SKIP_PROMPTS")

	deployCmd.PersistentFlags().Bool("json", false,
		"Output result as a single JSON object when the operation completes. "+
			"Implies non-interactive mode (no TUI, no streaming text output).",
	)
	confProvider.BindPFlag("deployJson", deployCmd.PersistentFlags().Lookup("json"))

	rootCmd.AddCommand(deployCmd)
}
