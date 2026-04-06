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

			cleanupValidations, _ := confProvider.GetBool("cleanupValidations")
			cleanupChangesets, _ := confProvider.GetBool("cleanupChangesets")
			cleanupReconciliationResults, _ := confProvider.GetBool("cleanupReconciliationResults")
			cleanupEvents, _ := confProvider.GetBool("cleanupEvents")

			noFlagsProvided := !cleanupValidations && !cleanupChangesets &&
				!cleanupReconciliationResults && !cleanupEvents

			inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
			headlessMode := !inTerminal

			if noFlagsProvided && headlessMode {
				cleanupValidations = true
				cleanupChangesets = true
				cleanupReconciliationResults = true
				cleanupEvents = true
			}

			if _, err := tea.LogToFile(fmt.Sprintf("%s-output.log", cfg.CLIName), "simple"); err != nil {
				log.Fatal(err)
			}

			cmd.SilenceUsage = true

			styles := stylespkg.NewStyles(
				lipgloss.NewRenderer(os.Stdout),
				cfg.Palette,
			)

			showOptionsForm := noFlagsProvided && !headlessMode

			app, err := cleanupui.NewCleanupApp(
				deployEngine,
				logger,
				cleanupValidations,
				cleanupChangesets,
				cleanupReconciliationResults,
				cleanupEvents,
				showOptionsForm,
				styles,
				headlessMode,
				os.Stdout,
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
