package splitpane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"
)

// The picker is how a status filter is found without knowing the "status:"
// syntax, so what matters is that it appears, says what each status would
// leave, and puts the chosen one into the term.
type StatusPickerTestSuite struct {
	suite.Suite
}

func TestStatusPickerTestSuite(t *testing.T) {
	suite.Run(t, new(StatusPickerTestSuite))
}

func (s *StatusPickerTestSuite) pickerModel() Model {
	model := New(Config{
		Styles:            testStyles(),
		StatusKeywords:    statusKeywordsForTest,
		StatusFilterHints: []string{"failed", "in-progress", "success", "pending"},
	})
	model.SetItems([]Item{
		&mockItem{id: "lambdaOk", name: "lambdaOk", icon: "✓", itemType: "resource"},
		&mockItem{id: "lambdaBad", name: "lambdaBad", icon: "✗", itemType: "resource"},
		&mockItem{id: "roleBad", name: "roleBad", icon: "✗", itemType: "resource"},
	})

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	return updated
}

func (s *StatusPickerTestSuite) send(model Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		model, _ = model.Update(msg)
	}
	return model
}

// The item list is the left pane, and the details pane can still be showing
// the selected item, so claims about the list are made against that half.
func leftPaneOf(view string) string {
	lines := []string{}
	for line := range strings.SplitSeq(view, "\n") {
		if at := strings.Index(line, "││"); at >= 0 {
			line = line[:at]
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// Opens a filter and then the picker, which is the only way in.
func (s *StatusPickerTestSuite) openPicker() Model {
	model := s.send(s.pickerModel(), keyRune('/'), tea.KeyMsg{Type: tea.KeyTab})
	s.Require().True(model.IsPickingStatus(), "tab should open the picker")
	return model
}

func (s *StatusPickerTestSuite) Test_the_picker_is_drawn_once_it_is_open() {
	view := s.openPicker().View()

	s.Contains(view, "Filter by status")
	s.Contains(view, "failed")
	s.Contains(view, "choose", "the picker should say what its keys do")
}

// Without this the mode was invisible: keys were being routed to the picker
// while the pane still showed the item list.
func (s *StatusPickerTestSuite) Test_the_picker_stands_in_for_the_item_list() {
	items := leftPaneOf(s.send(s.pickerModel(), keyRune('/')).View())
	s.Require().Contains(items, "lambdaOk")

	s.NotContains(leftPaneOf(s.openPicker().View()), "lambdaOk")
}

func (s *StatusPickerTestSuite) Test_each_status_says_how_much_it_would_leave() {
	view := s.openPicker().View()

	s.Contains(view, "failed  (2)")
	s.Contains(view, "success  (1)")
	s.Contains(view, "pending  (0)", "a status nothing is in is still offered")
}

func (s *StatusPickerTestSuite) Test_enter_applies_the_highlighted_status() {
	model := s.send(s.openPicker(), tea.KeyMsg{Type: tea.KeyEnter})

	s.False(model.IsPickingStatus(), "applying should close the picker")
	s.Equal(StatusFilterPrefix+"failed", model.FilterTerm())

	list := leftPaneOf(model.View())
	s.Contains(list, "lambdaBad", "the term should have been applied to the list")
	s.NotContains(list, "lambdaOk")
}

func (s *StatusPickerTestSuite) Test_the_selection_moves_before_being_applied() {
	model := s.send(
		s.openPicker(),
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyEnter},
	)

	s.Equal(StatusFilterPrefix+"success", model.FilterTerm())
}

// A name already typed is worth keeping, since the status is meant to compose
// with it rather than replace it.
func (s *StatusPickerTestSuite) Test_a_typed_name_survives_the_pick() {
	model := s.send(
		s.pickerModel(),
		keyRune('/'), keyRune('l'), keyRune('a'), keyRune('m'),
		tea.KeyMsg{Type: tea.KeyTab},
		tea.KeyMsg{Type: tea.KeyEnter},
	)

	s.Equal(StatusFilterPrefix+"failed lam", model.FilterTerm())
	s.Contains(leftPaneOf(model.View()), "lambdaBad")
}

func (s *StatusPickerTestSuite) Test_esc_leaves_the_term_alone() {
	model := s.send(s.openPicker(), tea.KeyMsg{Type: tea.KeyEsc})

	s.False(model.IsPickingStatus())
	s.Empty(model.FilterTerm())
	s.Contains(leftPaneOf(model.View()), "lambdaOk", "the list should be back")
}

// The picker lists the same statuses with counts, so the inline vocabulary
// would be saying it twice, in a form whose keys no longer apply.
func (s *StatusPickerTestSuite) Test_the_inline_vocabulary_gives_way_to_the_picker() {
	typing := leftPaneOf(s.send(s.pickerModel(), keyRune('/')).View())
	s.Require().Contains(typing, StatusFilterPrefix+"failed")
	s.Contains(typing, "tab to pick a status", "the picker has to be discoverable")

	picking := leftPaneOf(s.openPicker().View())
	for line := range strings.SplitSeq(picking, "\n") {
		s.NotContains(line, StatusFilterPrefix+"failed")
		s.NotContains(line, "enter to keep")
	}
}
