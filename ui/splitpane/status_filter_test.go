package splitpane

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

func testStyles() *styles.Styles {
	return styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

// Filtering a large deployment by name answers "where is this resource"; the
// question during a deployment is usually "what failed" or "what is still
// running". A "status:" word answers that, and composes with a name.
type StatusFilterTestSuite struct {
	suite.Suite
}

func TestStatusFilterTestSuite(t *testing.T) {
	suite.Run(t, new(StatusFilterTestSuite))
}

// statusKeywordsForTest stands in for the real icon-to-keyword mapping.
func statusKeywordsForTest(item Item) []string {
	switch item.GetIcon(false) {
	case "✗":
		return []string{"failed", "failure"}
	case "◐":
		return []string{"in-progress", "running"}
	case "✓":
		return []string{"success", "succeeded"}
	}
	return nil
}

func (s *StatusFilterTestSuite) modelWithStatuses() Model {
	model := New(Config{StatusKeywords: statusKeywordsForTest})
	model.SetItems([]Item{
		&mockItem{id: "lambdaOk", name: "lambdaOk", icon: "✓", itemType: "resource"},
		&mockItem{id: "lambdaBad", name: "lambdaBad", icon: "✗", itemType: "resource"},
		&mockItem{id: "roleBusy", name: "roleBusy", icon: "◐", itemType: "resource"},
		&mockItem{id: "roleBad", name: "roleBad", icon: "✗", itemType: "resource"},
	})
	return model
}

func (s *StatusFilterTestSuite) visibleNames(model Model) []string {
	names := make([]string, 0)
	for _, item := range model.visibleItems() {
		names = append(names, item.GetName())
	}
	return names
}

func (s *StatusFilterTestSuite) Test_a_status_narrows_to_that_status() {
	model := s.modelWithStatuses()
	model.SetFilterTerm("status:failed")

	s.ElementsMatch([]string{"lambdaBad", "roleBad"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_alternative_spellings_are_accepted() {
	model := s.modelWithStatuses()
	model.SetFilterTerm("status:failure")

	s.ElementsMatch([]string{"lambdaBad", "roleBad"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_several_statuses_widen_the_match() {
	// "failed or in-progress", not "both at once", which nothing could be.
	model := s.modelWithStatuses()
	model.SetFilterTerm("status:failed status:in-progress")

	s.ElementsMatch([]string{"lambdaBad", "roleBad", "roleBusy"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_a_status_composes_with_a_name() {
	model := s.modelWithStatuses()
	model.SetFilterTerm("status:failed role")

	s.Equal([]string{"roleBad"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_the_name_alone_still_works() {
	model := s.modelWithStatuses()
	model.SetFilterTerm("lambda")

	s.ElementsMatch([]string{"lambdaOk", "lambdaBad"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_an_unknown_status_matches_nothing() {
	model := s.modelWithStatuses()
	model.SetFilterTerm("status:nonsense")

	s.Empty(s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_status_filtering_is_off_when_nothing_reports_a_status() {
	// Without the hook a status filter cannot be answered, so it matches
	// nothing rather than quietly behaving as though it were not typed.
	model := New(Config{})
	model.SetItems([]Item{&mockItem{id: "a", name: "a", icon: "✗", itemType: "resource"}})
	model.SetFilterTerm("status:failed")

	s.Empty(s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_parsing_separates_statuses_from_text() {
	parsed := parseFilterTerm("status:failed orders status:pending api")

	s.Equal([]string{"failed", "pending"}, parsed.statuses)
	s.Equal("orders api", parsed.text)
}

func (s *StatusFilterTestSuite) Test_a_bare_prefix_is_ignored() {
	parsed := parseFilterTerm("status: orders")

	s.Empty(parsed.statuses)
	s.Equal("orders", parsed.text)
}

// --- discoverability ---

func (s *StatusFilterTestSuite) Test_a_partial_status_narrows_while_it_is_typed() {
	// Typing "status:f" should already be showing failures, the way a name
	// filter narrows on every keystroke, rather than matching nothing until
	// the word is finished.
	model := s.modelWithStatuses()

	model.SetFilterTerm("status:f")
	s.ElementsMatch([]string{"lambdaBad", "roleBad"}, s.visibleNames(model))

	model.SetFilterTerm("status:fail")
	s.ElementsMatch([]string{"lambdaBad", "roleBad"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) Test_a_partial_status_can_span_several_statuses() {
	model := s.modelWithStatuses()
	// "s" is the start of both "success" and nothing else here; the point is
	// that a prefix is allowed to be ambiguous while being typed.
	model.SetFilterTerm("status:s")

	s.ElementsMatch([]string{"lambdaOk"}, s.visibleNames(model))
}

func (s *StatusFilterTestSuite) paneWithHints(width int) Model {
	model := New(Config{
		Styles:            testStyles(),
		StatusKeywords:    statusKeywordsForTest,
		StatusFilterHints: []string{"failed", "in-progress", "pending", "success", "skipped"},
	})
	model.SetItems([]Item{&mockItem{id: "a", name: "a", icon: "✗", itemType: "resource"}})
	model.handleWindowSize(tea.WindowSizeMsg{Width: width, Height: 24})
	return model
}

func (s *StatusFilterTestSuite) Test_the_statuses_are_offered_while_typing() {
	model := s.paneWithHints(160)
	model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	view := model.leftPane.View()
	s.Contains(view, "status:failed", "the filter should say what it accepts")
	s.Contains(view, "name or status", "the empty box should say what it takes")
}

func (s *StatusFilterTestSuite) Test_the_statuses_have_a_line_of_their_own() {
	model := s.paneWithHints(160)
	model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Sharing a line with the key hints is what made them easy to miss and got
	// them clipped by the pane width.
	for line := range strings.SplitSeq(model.leftPane.View(), "\n") {
		if strings.Contains(line, "status:failed") {
			s.NotContains(line, "esc to clear")
			return
		}
	}
	s.Fail("no line offering the statuses")
}

func (s *StatusFilterTestSuite) Test_the_offered_statuses_are_never_cut_mid_word() {
	// A narrow pane shows fewer of them rather than half of one.
	model := s.paneWithHints(80)
	model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	line := ""
	for candidate := range strings.SplitSeq(model.leftPane.View(), "\n") {
		if strings.Contains(candidate, StatusFilterPrefix) {
			line = candidate
		}
	}
	s.Require().NotEmpty(line)

	for word := range strings.FieldsSeq(line) {
		if word == "…" || word == "│" {
			continue
		}
		status, found := strings.CutPrefix(word, StatusFilterPrefix)
		s.Require().Truef(found, "unexpected word %q", word)
		s.Containsf(
			[]string{"failed", "in-progress", "pending", "success", "skipped"},
			status, "%q is a partial status", status,
		)
	}
}

func (s *StatusFilterTestSuite) Test_the_statuses_are_not_offered_once_the_term_is_committed() {
	model := s.paneWithHints(160)
	model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "status:failed" {
		model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	view := model.leftPane.View()
	s.Contains(view, "Filter: status:failed", "the term stays visible")
	s.Contains(view, "/ to edit", "and how to change it")
	s.NotContains(view, "status:in-progress", "but the vocabulary has served its purpose")
}

// --- fitting the pane ---

func (s *StatusFilterTestSuite) hintLineAt(terminalWidth int) string {
	return s.paneWithHints(terminalWidth).statusHintLine()
}

func (s *StatusFilterTestSuite) Test_a_wide_pane_offers_every_status() {
	line := s.hintLineAt(240)

	for _, status := range []string{"failed", "in-progress", "pending", "success", "skipped"} {
		s.Contains(line, StatusFilterPrefix+status)
	}
	s.NotContains(line, "…", "nothing was left out, so nothing should be marked as left out")
}

func (s *StatusFilterTestSuite) Test_a_narrower_pane_offers_fewer_and_says_so() {
	line := s.hintLineAt(120)

	s.Contains(line, StatusFilterPrefix+"failed", "the most useful status survives")
	s.True(strings.HasSuffix(line, "…"), "the rest should be marked as omitted")
}

// Losing the line entirely on a narrow pane hides the feature exactly where
// filtering matters most, so the qualifier alone is kept.
func (s *StatusFilterTestSuite) Test_a_pane_too_narrow_for_a_status_still_shows_the_qualifier() {
	line := s.hintLineAt(50)

	s.Equal(StatusFilterPrefix+"…", line)
}

// Whatever the width, the line has to fit as a hint that wraps or is clipped
// mid-word is worse than a shorter one.
func (s *StatusFilterTestSuite) Test_the_line_always_fits_the_pane() {
	for _, terminalWidth := range []int{240, 200, 160, 120, 100, 80, 60, 50, 40, 30, 20} {
		model := s.paneWithHints(terminalWidth)
		line := model.statusHintLine()

		s.LessOrEqualf(
			len([]rune(line)), model.leftPane.Width-4,
			"the hint overflows the pane at terminal width %d: %q", terminalWidth, line,
		)
	}
}

func (s *StatusFilterTestSuite) Test_the_whole_pane_never_overflows_while_filtering() {
	for _, terminalWidth := range []int{240, 160, 120, 80, 60, 40} {
		model := s.paneWithHints(terminalWidth)
		model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

		for line := range strings.SplitSeq(model.leftPane.View(), "\n") {
			s.LessOrEqualf(
				len([]rune(strings.TrimRight(line, " "))), model.leftPane.Width,
				"a line overflows at terminal width %d: %q", terminalWidth, line,
			)
		}
	}
}
