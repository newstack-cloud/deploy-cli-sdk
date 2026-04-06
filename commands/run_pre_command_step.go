package commands

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/precommand"
)

// RunPreCommandStep runs the pre-command step with visual progress feedback.
// In interactive mode, it runs a mini bubbletea program with a spinner.
// In headless mode, it writes structured progress to the writer.
// This runs before the main TUI starts, so the pre-command step can
// modify the deploy config that the TUI/engine will read.
func RunPreCommandStep(
	step precommand.Step,
	confProvider *config.Provider,
	commandName string,
	styles *stylespkg.Styles,
	headless bool,
	writer io.Writer,
) error {
	model := precommand.NewModel(precommand.Options{
		Step:         step,
		ConfProvider: confProvider,
		CommandName:  commandName,
		Styles:       styles,
		Headless:     headless,
		Writer:       writer,
	})

	opts := []tea.ProgramOption{}
	if headless {
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
	}

	finalModel, err := tea.NewProgram(model, opts...).Run()
	if err != nil {
		return fmt.Errorf("pre-command step program error: %w", err)
	}

	if final, ok := finalModel.(precommand.Model); ok && final.Err != nil {
		return final.Err
	}

	return nil
}
