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
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/cleanupui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errCleanupFailed = errors.New("cleanup failed")

type cleanupFlags struct {
	validations           bool
	changesets            bool
	reconciliationResults bool
	events                bool
}

func readCleanupFlags(confProvider *config.Provider) cleanupFlags {
	validations, _ := confProvider.GetBool("cleanupValidations")
	changesets, _ := confProvider.GetBool("cleanupChangesets")
	reconciliationResults, _ := confProvider.GetBool("cleanupReconciliationResults")
	events, _ := confProvider.GetBool("cleanupEvents")

	return cleanupFlags{
		validations:           validations,
		changesets:            changesets,
		reconciliationResults: reconciliationResults,
		events:                events,
	}
}

func (f cleanupFlags) noFlagsProvided() bool {
	return !f.validations && !f.changesets && !f.reconciliationResults && !f.events
}

func (f cleanupFlags) resolveForHeadless(headlessMode bool) cleanupFlags {
	if f.noFlagsProvided() && headlessMode {
		f.validations = true
		f.changesets = true
		f.reconciliationResults = true
		f.events = true
	}
	return f
}

// SetupCleanupCommand registers a cleanup command on the root command,
// parameterized by CLIConfig for branding and defaults.
func SetupCleanupCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup temporary resources that have exceeded retention periods",
		Long: fmt.Sprintf(`Triggers cleanup of temporary resources in the deploy engine that have
exceeded their configured retention periods.

The deploy engine stores temporary data such as validation results, change sets,
reconciliation results, and streaming events for a configurable period.

In non-interactive mode, all resource types are cleaned up by default.
In interactive mode, you can select which resource types to clean up.

Use flags to clean specific resource types in either mode.

Examples:
  # Cleanup all resource types (non-interactive) or select types (interactive)
  %[1]s cleanup

  # Cleanup specific resource types
  %[1]s cleanup --validations --changesets`, cfg.CLIName),
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

			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headlessMode := !inTerminal

			flags := readCleanupFlags(confProvider).resolveForHeadless(headlessMode)

			if _, err := tea.LogToFile(fmt.Sprintf("%s-output.log", cfg.CLIName), "simple"); err != nil {
				log.Fatal(err)
			}

			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				cfg.Palette,
			)

			app, err := cleanupui.NewCleanupApp(cleanupui.CleanupAppConfig{
				Engine:                       deployEngine,
				Logger:                       logger,
				CleanupValidations:           flags.validations,
				CleanupChangesets:            flags.changesets,
				CleanupReconciliationResults: flags.reconciliationResults,
				CleanupEvents:                flags.events,
				ShowOptionsForm:              flags.noFlagsProvided() && !headlessMode,
				Styles:                       styles,
				Headless:                     headlessMode,
				HeadlessWriter:               os.Stdout,
			})
			if err != nil {
				return err
			}

			finalModel, err := tea.NewProgram(app, newTUIProgramOptions(headlessMode)...).Run()
			if err != nil {
				return err
			}
			finalApp := finalModel.(cleanupui.MainModel)

			if finalApp.Error != nil {
				cmd.SilenceErrors = true
				return errCleanupFailed
			}

			return nil
		},
	}

	prefix := cfg.EnvVarPrefix

	cleanupCmd.PersistentFlags().Bool("validations", false,
		"Cleanup blueprint validation results that have exceeded their retention period.",
	)
	confProvider.BindPFlag("cleanupValidations", cleanupCmd.PersistentFlags().Lookup("validations"))
	confProvider.BindEnvVar("cleanupValidations", prefix+"_CLEANUP_VALIDATIONS")

	cleanupCmd.PersistentFlags().Bool("changesets", false,
		"Cleanup change sets that have exceeded their retention period.",
	)
	confProvider.BindPFlag("cleanupChangesets", cleanupCmd.PersistentFlags().Lookup("changesets"))
	confProvider.BindEnvVar("cleanupChangesets", prefix+"_CLEANUP_CHANGESETS")

	cleanupCmd.PersistentFlags().Bool("reconciliation-results", false,
		"Cleanup reconciliation check results that have exceeded their retention period.",
	)
	confProvider.BindPFlag(
		"cleanupReconciliationResults",
		cleanupCmd.PersistentFlags().Lookup("reconciliation-results"),
	)
	confProvider.BindEnvVar("cleanupReconciliationResults", prefix+"_CLEANUP_RECONCILIATION_RESULTS")

	cleanupCmd.PersistentFlags().Bool("events", false,
		"Cleanup streaming events that have exceeded their retention period.",
	)
	confProvider.BindPFlag("cleanupEvents", cleanupCmd.PersistentFlags().Lookup("events"))
	confProvider.BindEnvVar("cleanupEvents", prefix+"_CLEANUP_EVENTS")

	rootCmd.AddCommand(cleanupCmd)
}
