package precommand

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

// mockStep is a test implementation of the Step interface.
type mockStep struct {
	err          error
	progressMsgs []ProgressMsg
	// delay is an optional pause after sending progress messages. A small
	// delay lets the tea runtime process ProgressUpdateMsgs before stepDoneMsg
	// is dispatched, avoiding a race between the two concurrent commands.
	delay time.Duration
}

func (m *mockStep) Run(
	_ context.Context,
	_ *config.Provider,
	_ string,
	progress chan<- ProgressMsg,
) error {
	for _, msg := range m.progressMsgs {
		progress <- msg
	}
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.err
}

type PrecommandModelSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func (s *PrecommandModelSuite) SetupSuite() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}


// Test_successful_step_completion verifies that when a step completes without
// error the final model has no error and is marked done.
func (s *PrecommandModelSuite) Test_successful_step_completion() {
	step := &mockStep{
		err: nil,
		progressMsgs: []ProgressMsg{
			{Phase: "Resolving context", Detail: "loading variables"},
		},
		// A small delay ensures ProgressUpdateMsg is dispatched to the tea
		// runtime before stepDoneMsg, avoiding a race between the two
		// concurrent commands started by Init.
		delay: 50 * time.Millisecond,
	}

	model := NewModel(Options{
		Step:         step,
		ConfProvider: config.NewProvider(),
		CommandName:  "deploy",
		Styles:       s.styles,
		Headless:     false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(Model)
	s.Nil(finalModel.Err)
}

// Test_step_failure verifies that when a step returns an error the model
// records the error on the public Err field and quits.
func (s *PrecommandModelSuite) Test_step_failure() {
	expectedErr := errors.New("context resolution failed")
	step := &mockStep{
		err:          expectedErr,
		progressMsgs: []ProgressMsg{},
	}

	model := NewModel(Options{
		Step:         step,
		ConfProvider: config.NewProvider(),
		CommandName:  "deploy",
		Styles:       s.styles,
		Headless:     false,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(Model)
	s.Require().NotNil(finalModel.Err)
	s.ErrorIs(finalModel.Err, expectedErr)
}

// Test_headless_mode verifies that in headless mode progress messages and the
// completion notice are written to the configured writer instead of being
// rendered in the TUI. Due to the model using value receivers, progressCh is
// not retained after Init, so only the first progress message is delivered.
func (s *PrecommandModelSuite) Test_headless_mode() {
	buf := testutils.NewSaveBuffer()
	step := &mockStep{
		err: nil,
		progressMsgs: []ProgressMsg{
			{Phase: "Resolving context", Detail: "loading variables"},
		},
		// A small delay ensures ProgressUpdateMsg is processed before
		// stepDoneMsg so the progress line is written to the buffer.
		delay: 50 * time.Millisecond,
	}

	model := NewModel(Options{
		Step:         step,
		ConfProvider: config.NewProvider(),
		CommandName:  "deploy",
		Styles:       s.styles,
		Headless:     true,
		Writer:       buf,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		buf,
		"Running pre-command step",
		"Resolving context",
		"loading variables",
		"Pre-command step complete",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(Model)
	s.Nil(finalModel.Err)
}

// Test_headless_mode_step_failure verifies that in headless mode a step error
// is written to the configured writer and the model records the error.
func (s *PrecommandModelSuite) Test_headless_mode_step_failure() {
	buf := testutils.NewSaveBuffer()
	expectedErr := errors.New("network timeout")
	step := &mockStep{
		err:          expectedErr,
		progressMsgs: []ProgressMsg{},
	}

	model := NewModel(Options{
		Step:         step,
		ConfProvider: config.NewProvider(),
		CommandName:  "deploy",
		Styles:       s.styles,
		Headless:     true,
		Writer:       buf,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContains(
		s.T(),
		buf,
		"Pre-command step failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(Model)
	s.Require().NotNil(finalModel.Err)
	s.ErrorIs(finalModel.Err, expectedErr)
}

func TestPrecommandModelSuite(t *testing.T) {
	suite.Run(t, new(PrecommandModelSuite))
}
