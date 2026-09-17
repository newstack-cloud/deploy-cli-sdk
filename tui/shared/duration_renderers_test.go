package shared

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type DurationRenderersTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDurationRenderersTestSuite(t *testing.T) {
	suite.Run(t, new(DurationRenderersTestSuite))
}

func (s *DurationRenderersTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func ms(v float64) *float64 { return &v }

func (s *DurationRenderersTestSuite) Test_resource_durations() {
	out := RenderResourceDurations(&state.ResourceCompletionDurations{
		ConfigCompleteDuration: ms(820),
		TotalDuration:          ms(4310),
	}, s.styles)

	s.Contains(out, "Config Complete: 820ms")
	s.Contains(out, "Total: 4.31s")
}

func (s *DurationRenderersTestSuite) Test_link_durations_break_down_by_component() {
	out := RenderLinkDurations(&state.LinkCompletionDurations{
		LinkedResourcesUpdate: &state.LinkComponentCompletionDurations{TotalDuration: ms(120)},
		IntermediaryResources: &state.LinkComponentCompletionDurations{TotalDuration: ms(1500)},
		TotalDuration:         ms(1750),
	}, s.styles)

	s.Contains(out, "Linked Resources Update: 120ms")
	s.Contains(out, "Intermediary Resources: 1.50s")
	s.Contains(out, "Total: 1.75s")
}

func (s *DurationRenderersTestSuite) Test_instance_durations() {
	out := RenderInstanceDurations(&state.InstanceCompletionDuration{
		PrepareDuration: ms(300),
		TotalDuration:   ms(65000),
	}, s.styles)

	s.Contains(out, "Prepare: 300ms")
	s.Contains(out, "Total: 1m 5s")
}

func (s *DurationRenderersTestSuite) Test_nothing_is_rendered_without_durations() {
	s.Empty(RenderResourceDurations(nil, s.styles))
	s.Empty(RenderLinkDurations(nil, s.styles))
	s.Empty(RenderInstanceDurations(nil, s.styles))
}

func (s *DurationRenderersTestSuite) Test_unrecorded_values_are_skipped() {
	// A resource that never reached config complete has no duration for it.
	out := RenderResourceDurations(&state.ResourceCompletionDurations{
		TotalDuration: ms(1200),
	}, s.styles)

	s.NotContains(out, "Config Complete")
	s.Contains(out, "Total: 1.20s")

	// Zero is "not recorded" rather than an instant operation.
	s.Empty(RenderResourceDurations(&state.ResourceCompletionDurations{
		TotalDuration: ms(0),
	}, s.styles))
}

func (s *DurationRenderersTestSuite) Test_attempts_are_listed_only_after_a_retry() {
	single := RenderResourceDurations(&state.ResourceCompletionDurations{
		TotalDuration:    ms(1200),
		AttemptDurations: []float64{1200},
	}, s.styles)
	s.NotContains(single, "Attempts", "a single attempt says nothing worth a line")

	retried := RenderResourceDurations(&state.ResourceCompletionDurations{
		TotalDuration:    ms(3000),
		AttemptDurations: []float64{900, 2100},
	}, s.styles)
	s.Contains(retried, "Attempts: 900ms, 2.10s")
}

func (s *DurationRenderersTestSuite) Test_the_section_is_omitted_when_there_is_nothing_to_show() {
	sb := &strings.Builder{}
	RenderTimingSection(sb, "", s.styles)
	s.Empty(sb.String(), "an empty timing heading is noise")

	sb = &strings.Builder{}
	RenderTimingSection(sb, "  Total: 1.20s\n", s.styles)
	s.Contains(sb.String(), "Timing:")
	s.Contains(sb.String(), "Total: 1.20s")
}
