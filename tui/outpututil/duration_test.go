package outpututil

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type FormatDurationTestSuite struct {
	suite.Suite
}

func TestFormatDurationTestSuite(t *testing.T) {
	suite.Run(t, new(FormatDurationTestSuite))
}

func (s *FormatDurationTestSuite) Test_short_durations() {
	s.Equal("0ms", FormatDuration(0))
	s.Equal("999ms", FormatDuration(999))
	s.Equal("1.00s", FormatDuration(1000))
	s.Equal("45.00s", FormatDuration(45_000))
}

func (s *FormatDurationTestSuite) Test_minutes() {
	s.Equal("1m 30s", FormatDuration(90_000))
	s.Equal("59m 59s", FormatDuration(59*60*1000+59*1000))
}

func (s *FormatDurationTestSuite) Test_long_durations_do_not_pile_up_as_minutes() {
	// A deploy step can genuinely run for hours; "120m 0s" is hard to read.
	s.Equal("1h 0m 0s", FormatDuration(3_600_000))
	s.Equal("2h 1m 5s", FormatDuration(7_265_000))
	s.Equal("1d 1h 0m", FormatDuration(90_000_000))
}

// An unstarted attempt has a zero start time, and measuring from it overflows
// time.Duration, which saturates at math.MaxInt64 nanoseconds. That arrives
// here as ~9.2e12 milliseconds and used to render as "153722867m 16s", which
// reads like a real measurement of something that took 292 years.
func (s *FormatDurationTestSuite) Test_an_overflowed_duration_is_not_presented_as_a_measurement() {
	overflowed := float64(math.MaxInt64) / 1e6

	s.Equal("unavailable", FormatDuration(overflowed))
	s.False(IsPlausibleDuration(overflowed))
}

func (s *FormatDurationTestSuite) Test_the_overflow_matches_measuring_from_an_unset_time() {
	// Pin the arithmetic this guards against, so the guard cannot drift away
	// from the value it exists to catch.
	var unset time.Time
	fromUnsetTime := float64(time.Since(unset)) / 1e6

	s.False(IsPlausibleDuration(fromUnsetTime))
	s.Equal("unavailable", FormatDuration(fromUnsetTime))
}

func (s *FormatDurationTestSuite) Test_nonsense_values_are_rejected() {
	s.Equal("unavailable", FormatDuration(-5))
	s.Equal("unavailable", FormatDuration(math.NaN()))
	s.Equal("unavailable", FormatDuration(math.Inf(1)))
	s.Equal("unavailable", FormatDuration(math.Inf(-1)))
}

func (s *FormatDurationTestSuite) Test_the_bar_is_set_above_anything_a_deployment_could_take() {
	// 30 days is the cut-off and real work stays well under it, sentinels sit far
	// above, so a slow deployment is never quietly hidden.
	s.True(IsPlausibleDuration(29*24*60*60*1000), "29 days should still be reported")
	s.False(IsPlausibleDuration(31 * 24 * 60 * 60 * 1000))
}
