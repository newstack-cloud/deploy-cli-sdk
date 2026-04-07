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
	"github.com/newstack-cloud/deploy-cli-sdk/tui/inspectui"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/listui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errInspectFailed = errors.New("inspect failed")
var errListFailed = errors.New("list instances failed")

// SetupInstancesCommand registers an instances command with inspect and list
// subcommands on the root command, parameterized by CLIConfig for branding.
func SetupInstancesCommand(rootCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	instancesCmd := &cobra.Command{
		Use:   "instances",
		Short: "Manage and view blueprint instances",
		Long: `Commands for managing and viewing blueprint instances deployed via the
deploy engine. Use subcommands to list, inspect, or manage instances.`,
	}

	setupInstancesInspectCommand(instancesCmd, confProvider, cfg)
	setupInstancesListCommand(instancesCmd, confProvider, cfg)

	rootCmd.AddCommand(instancesCmd)
}

func setupInstancesInspectCommand(instancesCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	inspectCmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect a blueprint instance",
		Long: fmt.Sprintf(`Displays the current state of a blueprint instance including resources,
links, child blueprints, and deployment status.

If a deployment or destroy operation is currently in progress, the command
streams real-time updates until the operation completes.

Examples:
  # Interactive mode - enter instance name when prompted
  %[1]s instances inspect

  # Inspect by instance name
  %[1]s instances inspect --instance-name my-app

  # Inspect by instance ID
  %[1]s instances inspect --instance-id abc123

  # Output as JSON (useful for CI/CD or scripting)
  %[1]s instances inspect --instance-name my-app --json`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(cmd, confProvider, cfg)
		},
	}

	prefix := cfg.EnvVarPrefix

	inspectCmd.PersistentFlags().String(
		flagInstanceID, "",
		"The system-generated ID of the blueprint instance to inspect. "+
			"Leave empty if using --instance-name.",
	)
	confProvider.BindPFlag("instancesInspectInstanceID", inspectCmd.PersistentFlags().Lookup(flagInstanceID))
	confProvider.BindEnvVar("instancesInspectInstanceID", prefix+"_INSTANCES_INSPECT_INSTANCE_ID")

	inspectCmd.PersistentFlags().String(
		flagInstanceName, "",
		"The user-defined unique name of the blueprint instance to inspect. "+
			"Leave empty if using --instance-id.",
	)
	confProvider.BindPFlag("instancesInspectInstanceName", inspectCmd.PersistentFlags().Lookup(flagInstanceName))
	confProvider.BindEnvVar("instancesInspectInstanceName", prefix+"_INSTANCES_INSPECT_INSTANCE_NAME")

	inspectCmd.PersistentFlags().Bool("json", false,
		"Output the instance state as JSON. "+
			"Implies non-interactive mode (no TUI).",
	)
	confProvider.BindPFlag("instancesInspectJson", inspectCmd.PersistentFlags().Lookup("json"))
	confProvider.BindEnvVar("instancesInspectJson", prefix+"_INSTANCES_INSPECT_JSON")

	instancesCmd.AddCommand(inspectCmd)
}

type inspectFlags struct {
	instanceID            string
	instanceIDIsDefault   bool
	instanceName          string
	instanceNameIsDefault bool
	jsonMode              bool
}

func readInspectFlags(confProvider *config.Provider) inspectFlags {
	instanceID, instanceIDIsDefault := confProvider.GetString("instancesInspectInstanceID")
	instanceName, instanceNameIsDefault := confProvider.GetString("instancesInspectInstanceName")
	jsonMode, _ := confProvider.GetBool("instancesInspectJson")

	return inspectFlags{
		instanceID:            instanceID,
		instanceIDIsDefault:   instanceIDIsDefault,
		instanceName:          instanceName,
		instanceNameIsDefault: instanceNameIsDefault,
		jsonMode:              jsonMode,
	}
}

func validateInspectFlags(flags inspectFlags) error {
	return headless.Validate(headless.OneOf(
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
	))
}

func runInspect(cmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) error {
	logger, handle, err := SetupLogger(cfg.CLIName)
	if err != nil {
		return err
	}
	defer handle.Close()

	deployEngine, err := engine.Create(confProvider, logger)
	if err != nil {
		return err
	}

	flags := readInspectFlags(confProvider)

	if flags.jsonMode {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	if err := validateInspectFlags(flags); err != nil {
		if flags.jsonMode {
			jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
			return errInspectFailed
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
	headlessMode := !inTerminal || flags.jsonMode

	app, err := inspectui.NewInspectApp(inspectui.InspectAppConfig{
		DeployEngine:   deployEngine,
		Logger:         logger,
		InstanceID:     flags.instanceID,
		InstanceName:   flags.instanceName,
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
	finalApp := finalModel.(inspectui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errInspectFailed
	}

	return nil
}

func setupInstancesListCommand(instancesCmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List blueprint instances",
		Long: fmt.Sprintf(`Lists all blueprint instances managed by the deploy engine.

In interactive mode, the list is paginated and you can filter instances using search.
Selecting an instance navigates to the inspect view.

Examples:
  # Interactive mode - browse and filter instances
  %[1]s instances list

  # Filter instances by name
  %[1]s instances list --search "production"

  # Output as JSON (useful for CI/CD or scripting)
  %[1]s instances list --json`, cfg.CLIName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListInstances(cmd, confProvider, cfg)
		},
	}

	prefix := cfg.EnvVarPrefix

	listCmd.PersistentFlags().String(
		"search", "",
		"Filter instances by name (case-insensitive substring match).",
	)
	confProvider.BindPFlag("instancesListSearch", listCmd.PersistentFlags().Lookup("search"))
	confProvider.BindEnvVar("instancesListSearch", prefix+"_INSTANCES_LIST_SEARCH")

	listCmd.PersistentFlags().Bool("json", false,
		"Output the instance list as JSON. Implies non-interactive mode.",
	)
	confProvider.BindPFlag("instancesListJson", listCmd.PersistentFlags().Lookup("json"))
	confProvider.BindEnvVar("instancesListJson", prefix+"_INSTANCES_LIST_JSON")

	instancesCmd.AddCommand(listCmd)
}

func runListInstances(cmd *cobra.Command, confProvider *config.Provider, cfg *CLIConfig) error {
	logger, handle, err := SetupLogger(cfg.CLIName)
	if err != nil {
		return err
	}
	defer handle.Close()

	deployEngine, err := engine.Create(confProvider, logger)
	if err != nil {
		return err
	}

	search, _ := confProvider.GetString("instancesListSearch")
	jsonMode, _ := confProvider.GetBool("instancesListJson")

	if jsonMode {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	cmd.SilenceUsage = true

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		cfg.Palette,
	)
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || jsonMode

	app, err := listui.NewListApp(
		deployEngine,
		logger,
		search,
		styles,
		headlessMode,
		os.Stdout,
		jsonMode,
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
	listApp := finalModel.(listui.MainModel)

	if listApp.Error != nil {
		cmd.SilenceErrors = true
		return errListFailed
	}

	return nil
}
