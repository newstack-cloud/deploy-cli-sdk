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
	"golang.org/x/term"
)

var errDestroyFailed = errors.New("destroy failed")

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
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				autoApprove = true
			}

			err = headless.Validate(
				headless.OneOf(
					headless.Flag{
						Name:      "instance-name",
						Value:     instanceName,
						IsDefault: instanceNameIsDefault,
					},
					headless.Flag{
						Name:      "instance-id",
						Value:     instanceID,
						IsDefault: instanceIDIsDefault,
					},
				),
				headless.OneOf(
					headless.Flag{
						Name:      "stage",
						Value:     boolToString(stageFirst),
						IsDefault: !stageFirst,
					},
					headless.Flag{
						Name:      "change-set-id",
						Value:     changesetID,
						IsDefault: changesetIDIsDefault,
					},
				),
				headless.RequiredIfBool(
					headless.BoolFlagTrue("stage", stageFirst),
					"auto-approve",
					autoApprove,
				),
			)
			if err != nil {
				if jsonMode {
					jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
					return errDestroyFailed
				}
				return err
			}

			if _, err := tea.LogToFile(fmt.Sprintf("%s-output.log", cfg.CLIName), "simple"); err != nil {
				log.Fatal(err)
			}

			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				cfg.Palette,
			)
			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headlessMode := !inTerminal || jsonMode

			if cfg.PreCommandStep != nil {
				if err := RunPreCommandStep(cfg.PreCommandStep, confProvider, "destroy", styles, headlessMode, os.Stdout); err != nil {
					return err
				}
			}

			var preflightModel tea.Model
			if cfg.PreflightFactory != nil {
				skipCheck, _ := confProvider.GetBool("skipPluginCheck")
				if !skipCheck {
					preflightModel = cfg.PreflightFactory.CreatePreflight(
						confProvider, "destroy", styles, headlessMode, os.Stdout, jsonMode,
					)
				}
			}

			app, err := destroyui.NewDestroyApp(
				destroyEngine,
				logger,
				changesetID,
				instanceID,
				instanceName,
				blueprintFile,
				isDefaultBlueprintFile,
				force,
				stageFirst,
				autoApprove,
				skipPrompts,
				styles,
				headlessMode,
				os.Stdout,
				jsonMode,
				preflightModel,
			)
			if err != nil {
				return err
			}

			options := []tea.ProgramOption{}
			if !headlessMode {
				options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
			} else {
				options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
			}

			finalModel, err := tea.NewProgram(app, options...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(destroyui.MainModel)

			if finalApp.Error != nil {
				cmd.SilenceErrors = true
				return errDestroyFailed
			}

			return nil
		},
	}

	prefix := cfg.EnvVarPrefix

	destroyCmd.PersistentFlags().String(
		"change-set-id", "",
		"The ID of the change set to use for destruction. "+
			"If not provided, the latest destroy change set for the instance will be used.",
	)
	confProvider.BindPFlag("destroyChangeSetID", destroyCmd.PersistentFlags().Lookup("change-set-id"))
	confProvider.BindEnvVar("destroyChangeSetID", prefix+"_DESTROY_CHANGE_SET_ID")

	destroyCmd.PersistentFlags().String(
		"instance-id", "",
		"The system-generated ID of the blueprint instance to destroy. "+
			"Leave empty if using --instance-name.",
	)
	confProvider.BindPFlag("destroyInstanceID", destroyCmd.PersistentFlags().Lookup("instance-id"))
	confProvider.BindEnvVar("destroyInstanceID", prefix+"_DESTROY_INSTANCE_ID")

	destroyCmd.PersistentFlags().String(
		"instance-name", "",
		"The user-defined unique identifier for the blueprint instance to destroy. "+
			"Leave empty if using --instance-id.",
	)
	confProvider.BindPFlag("destroyInstanceName", destroyCmd.PersistentFlags().Lookup("instance-name"))
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

	destroyCmd.PersistentFlags().Bool("auto-approve", false,
		"Automatically approve staged changes without prompting for confirmation. "+
			"This is intended for CI/CD pipelines where manual approval is not possible. "+
			"Only applicable when --stage is set.",
	)
	confProvider.BindPFlag("destroyAutoApprove", destroyCmd.PersistentFlags().Lookup("auto-approve"))
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
