package stageui

import (
	"io"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ChangeDiagnosticsSuite struct {
	suite.Suite
}

func (s *ChangeDiagnosticsSuite) modelWithChanges(
	buf io.Writer,
	staged *changes.BlueprintChanges,
) *StageModel {
	model := NewStageModel(StageModelConfig{
		Logger:         zap.NewNop(),
		InstanceName:   "test-instance",
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		IsHeadless:     true,
		HeadlessWriter: buf,
	})
	model.completeChanges = staged
	return &model
}

func (s *ChangeDiagnosticsSuite) Test_a_transform_warning_reaches_the_summary() {
	buf := testutils.NewSaveBuffer()
	model := s.modelWithChanges(buf, &changes.BlueprintChanges{
		Diagnostics: []*core.Diagnostic{
			{
				Level: core.DiagnosticLevelWarning,
				Message: `celerity/consumer "taskQueueConsumer" could not be classified to a ` +
					`concrete event source; its trigger has been skipped`,
			},
		},
	})

	model.printHeadlessChangeDiagnostics()

	out := buf.String()
	s.Contains(out, "Diagnostics (1)")
	s.Contains(out, "taskQueueConsumer")
	// The printer wraps at 80 columns, so assert on a fragment rather than a phrase
	// that a wrap can split.
	s.Contains(out, "skipped")
	s.Contains(out, "WARNING")
}

// The level has to survive, since a warning an operator should act on reads differently
// from an informational note.
func (s *ChangeDiagnosticsSuite) Test_the_level_is_shown() {
	buf := testutils.NewSaveBuffer()
	model := s.modelWithChanges(buf, &changes.BlueprintChanges{
		Diagnostics: []*core.Diagnostic{
			{Level: core.DiagnosticLevelWarning, Message: "a warning"},
			{Level: core.DiagnosticLevelInfo, Message: "a note"},
		},
	})

	model.printHeadlessChangeDiagnostics()

	out := buf.String()
	s.Contains(out, "Diagnostics (2)")
	s.Contains(out, "a warning")
	s.Contains(out, "a note")
}

// A clean stage must not grow an empty section.
func (s *ChangeDiagnosticsSuite) Test_nothing_is_printed_when_there_are_none() {
	for name, staged := range map[string]*changes.BlueprintChanges{
		"no change set":  nil,
		"no diagnostics": {Diagnostics: nil},
	} {
		buf := testutils.NewSaveBuffer()
		model := s.modelWithChanges(buf, staged)
		model.printHeadlessChangeDiagnostics()
		s.Empty(buf.String(), name)
	}
}

func TestChangeDiagnosticsSuite(t *testing.T) {
	suite.Run(t, new(ChangeDiagnosticsSuite))
}
