package deployui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type OutcomeTallySuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestOutcomeTallySuite(t *testing.T) {
	suite.Run(t, new(OutcomeTallySuite))
}

func (s *OutcomeTallySuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// Runs a headless deployment of the given events and returns the one line of
// its summary that tallies what the run applied.
func (s *OutcomeTallySuite) summaryLineFor(
	events []*types.BlueprintInstanceEvent,
	finalStatus core.InstanceStatus,
) string {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(DeployModelConfig{
		DeployEngine: testutils.NewTestDeployEngineWithDeployment(
			append(events, finishEvent(finalStatus)),
			"test-instance-id",
			testInstanceState(finalStatus),
		),
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset-tally",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		IsHeadless:     true,
		HeadlessWriter: headlessOutput,
	})

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Headless lines carry a prefix, and "Complete:" is a substring of
	// "Incomplete:", so the longer label has to be looked for first.
	for line := range strings.SplitSeq(headlessOutput.String(), "\n") {
		for _, label := range []string{"Incomplete:", "Complete:"} {
			if at := strings.Index(line, label); at >= 0 {
				return strings.TrimSpace(line[at:])
			}
		}
	}

	s.FailNow("no summary line in the headless output", headlessOutput.String())
	return ""
}

func (s *OutcomeTallySuite) resourcesAt(statuses ...core.ResourceStatus) []*types.BlueprintInstanceEvent {
	events := []*types.BlueprintInstanceEvent{}
	for _, status := range statuses {
		events = append(events, resourceEvent(
			"resource-"+status.String(),
			status,
			core.PreciseResourceStatusUnknown,
		))
	}
	return events
}

func (s *OutcomeTallySuite) linksAt(statuses ...core.LinkStatus) []*types.BlueprintInstanceEvent {
	events := []*types.BlueprintInstanceEvent{}
	for _, status := range statuses {
		events = append(events, linkEvent(
			"a-"+status.String()+"::b-"+status.String(),
			status,
			core.PreciseLinkStatusUnknown,
		))
	}
	return events
}

// A failed run used to report the planned totals under the word "Complete",
// which read as success and matched the change set exactly.
func (s *OutcomeTallySuite) Test_a_failed_run_does_not_claim_completion() {
	line := s.summaryLineFor(
		s.resourcesAt(
			core.ResourceStatusCreated,
			core.ResourceStatusCreateFailed,
			core.ResourceStatusCreating,
		),
		core.InstanceStatusDeployFailed,
	)

	s.NotContains(line, "Complete:")
	s.Contains(line, "Incomplete: 1 of 3 items applied")
	s.Contains(line, "1 failed")
	s.Contains(line, "1 not applied")
}

func (s *OutcomeTallySuite) Test_a_successful_run_reports_completion() {
	line := s.summaryLineFor(
		s.resourcesAt(core.ResourceStatusCreated, core.ResourceStatusUpdated),
		core.InstanceStatusDeployed,
	)

	s.Equal("Complete: 2 items", line)
}

// Counts that are zero are left out rather than printed as "0 failed", so the
// line stays readable in the common case.
func (s *OutcomeTallySuite) Test_empty_categories_are_omitted() {
	line := s.summaryLineFor(
		s.resourcesAt(core.ResourceStatusCreated, core.ResourceStatusCreating),
		core.InstanceStatusDeployFailed,
	)

	s.Contains(line, "1 not applied")
	s.NotContains(line, "failed")
}

// Retained and rolled-back resources reached the state the deploy intended for
// them, so they are not failures. Each run holds one resource per status, so
// the count in the line pins every status in the group.
func (s *OutcomeTallySuite) Test_resource_statuses_that_count_as_applied() {
	line := s.summaryLineFor(
		s.resourcesAt(
			core.ResourceStatusCreated,
			core.ResourceStatusUpdated,
			core.ResourceStatusDestroyed,
			core.ResourceStatusRollbackComplete,
			core.ResourceStatusRetained,
		),
		core.InstanceStatusDeployed,
	)

	s.Equal("Complete: 5 items", line)
}

func (s *OutcomeTallySuite) Test_resource_statuses_that_count_as_failures() {
	line := s.summaryLineFor(
		s.resourcesAt(
			core.ResourceStatusCreateFailed,
			core.ResourceStatusUpdateFailed,
			core.ResourceStatusDestroyFailed,
			core.ResourceStatusRollbackFailed,
		),
		core.InstanceStatusDeployFailed,
	)

	s.Equal("Incomplete: 0 of 4 items applied, 4 failed", line)
}

// Interrupted by a failure elsewhere, and still in flight: neither is a failure
// of the resource itself, and neither is an applied change.
func (s *OutcomeTallySuite) Test_resource_statuses_that_count_as_unfinished() {
	line := s.summaryLineFor(
		s.resourcesAt(
			core.ResourceStatusCreateInterrupted,
			core.ResourceStatusUpdateInterrupted,
			core.ResourceStatusDestroyInterrupted,
			core.ResourceStatusUnknown,
		),
		core.InstanceStatusDeployFailed,
	)

	s.Equal("Incomplete: 0 of 4 items applied, 4 not applied", line)
}

func (s *OutcomeTallySuite) Test_link_statuses_are_classified_the_same_way() {
	applied := s.summaryLineFor(
		s.linksAt(
			core.LinkStatusCreated,
			core.LinkStatusUpdated,
			core.LinkStatusDestroyed,
			core.LinkStatusCreateRollbackComplete,
			core.LinkStatusDestroyRollbackComplete,
		),
		core.InstanceStatusDeployed,
	)
	s.Equal("Complete: 5 items", applied)

	failed := s.summaryLineFor(
		s.linksAt(
			core.LinkStatusCreateFailed,
			core.LinkStatusUpdateFailed,
			core.LinkStatusDestroyFailed,
			core.LinkStatusCreateRollbackFailed,
			core.LinkStatusDestroyRollbackFailed,
		),
		core.InstanceStatusDeployFailed,
	)
	s.Equal("Incomplete: 0 of 5 items applied, 5 failed", failed)

	unfinished := s.summaryLineFor(
		s.linksAt(core.LinkStatusCreating),
		core.InstanceStatusDeployFailed,
	)
	s.Equal("Incomplete: 0 of 1 item applied, 1 not applied", unfinished)
}
