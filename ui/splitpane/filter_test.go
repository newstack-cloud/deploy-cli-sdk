package splitpane

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type FilterTestSuite struct {
	suite.Suite
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

// Feeds messages through the pane the way the runtime does, since Update
// returns the next model rather than changing this one.
func sendMsgs(model Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		model, _ = model.Update(msg)
	}
	return model
}

func typeTerm(model Model, term string) Model {
	for _, r := range term {
		model = sendMsgs(model, keyRune(r))
	}
	return model
}

// A deployment shaped like a transformed blueprint where two abstract resources
// expanded into concrete resources, plus their links and an ungrouped resource.
func (s *FilterTestSuite) treeItems() []Item {
	return []Item{
		groupHeader("group:celerity/api:ordersApi", "[celerity/api] ordersApi"),
		child("ordersApi_http_api"),
		child("ordersApi_http_stage"),
		child("ordersApi_http_api::ordersApi_http_stage"),
		groupHeader("group:celerity/handler:createOrder", "[celerity/handler] createOrder"),
		child("createOrder_lambda"),
		child("createOrder_role"),
		&mockItem{id: "auditTable", name: "auditTable", itemType: "resource"},
	}
}

func (s *FilterTestSuite) treeModel() Model {
	model := New(Config{Styles: testStyles()})
	model.SetItems(s.treeItems())
	// Wide enough that no name is truncated, which would hide it from the pane.
	return sendMsgs(model, tea.WindowSizeMsg{Width: 220, Height: 40})
}

// The names the pane shows, in the order it shows them. Longer names are
// matched first so a link is not read as the resource its name starts with.
func (s *FilterTestSuite) namesInPane(view string) []string {
	candidates := make([]string, 0)
	for _, item := range s.treeItems() {
		candidates = append(candidates, item.GetName())
	}
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i]) > len(candidates[j])
	})

	found := make([]string, 0)
	for line := range strings.SplitSeq(leftPaneOf(view), "\n") {
		for _, name := range candidates {
			if strings.Contains(line, name) {
				found = append(found, name)
				break
			}
		}
	}
	return found
}

func (s *FilterTestSuite) filterNames(term string) []string {
	model := s.treeModel()
	model.SetFilterTerm(term)
	return s.namesInPane(model.View())
}

func (s *FilterTestSuite) Test_a_matching_group_keeps_all_of_its_contents() {
	s.Equal(
		[]string{
			"[celerity/api] ordersApi",
			"ordersApi_http_api",
			"ordersApi_http_stage",
			"ordersApi_http_api::ordersApi_http_stage",
		},
		s.filterNames("ordersapi"),
	)
}

func (s *FilterTestSuite) Test_matching_the_abstract_type_keeps_the_whole_group() {
	s.Equal(
		[]string{
			"[celerity/handler] createOrder",
			"createOrder_lambda",
			"createOrder_role",
		},
		s.filterNames("celerity/handler"),
	)
}

func (s *FilterTestSuite) Test_a_matching_child_keeps_its_group_for_context() {
	// Only the matching child is kept, but its group header comes with it.
	s.Equal(
		[]string{"[celerity/handler] createOrder", "createOrder_role"},
		s.filterNames("_role"),
	)
}

func (s *FilterTestSuite) Test_matching_links_are_kept_with_their_group() {
	s.Equal(
		[]string{
			"[celerity/api] ordersApi",
			"ordersApi_http_api::ordersApi_http_stage",
		},
		s.filterNames("::"),
	)
}

func (s *FilterTestSuite) Test_ungrouped_items_match_on_their_own() {
	s.Equal([]string{"auditTable"}, s.filterNames("audit"))
}

func (s *FilterTestSuite) Test_a_matching_sibling_does_not_drag_in_the_next_group() {
	// "ordersApi" matches the first group only. Neither the following group nor
	// the ungrouped resource after it may be carried along.
	names := s.filterNames("ordersapi")
	s.NotContains(names, "[celerity/handler] createOrder")
	s.NotContains(names, "auditTable")
}

func (s *FilterTestSuite) Test_no_matches_yields_nothing() {
	s.Empty(s.filterNames("nothing-matches-this"))
}

func (s *FilterTestSuite) Test_matching_is_case_insensitive() {
	s.Equal(s.filterNames("ordersapi"), s.filterNames("OrDeRsApI"))
}

// --- pane behaviour ---

func (s *FilterTestSuite) newFilteredModel() Model {
	model := New(Config{
		Styles:         testStyles(),
		SectionGrouper: &groupingGrouper{groupID: "group:celerity/handler:orderHandler"},
	})
	model.SetItems([]Item{
		&mockItem{id: "lambda", name: "lambda", itemType: "resource"},
		&mockItem{id: "role", name: "role", itemType: "resource"},
	})
	return sendMsgs(model, tea.WindowSizeMsg{Width: 160, Height: 30})
}

func (s *FilterTestSuite) Test_slash_starts_filtering_and_typing_builds_the_term() {
	model := sendMsgs(s.newFilteredModel(), keyRune('/'))
	s.True(model.IsFiltering(), "typing should be captured after /")

	model = typeTerm(model, "role")
	s.Equal("role", model.FilterTerm())
	s.True(model.HasFilter())
}

func (s *FilterTestSuite) Test_enter_keeps_the_term_and_releases_the_keys() {
	model := sendMsgs(
		s.newFilteredModel(),
		keyRune('/'), keyRune('r'), tea.KeyMsg{Type: tea.KeyEnter},
	)

	s.False(model.IsFiltering(), "navigation keys should work again")
	s.Equal("r", model.FilterTerm(), "the term stays applied")
}

func (s *FilterTestSuite) Test_escape_clears_the_term() {
	model := sendMsgs(
		s.newFilteredModel(),
		keyRune('/'), keyRune('r'), tea.KeyMsg{Type: tea.KeyEsc},
	)

	s.False(model.IsFiltering())
	s.False(model.HasFilter())
}

func (s *FilterTestSuite) Test_backspace_removes_the_last_character() {
	model := typeTerm(sendMsgs(s.newFilteredModel(), keyRune('/')), "role")
	model = sendMsgs(model, tea.KeyMsg{Type: tea.KeyBackspace})

	s.Equal("rol", model.FilterTerm())
}

func (s *FilterTestSuite) Test_letters_do_not_navigate_while_the_term_is_typed() {
	// "k" and "j" move the selection when not filtering.
	model := sendMsgs(s.newFilteredModel(), keyRune('/'), keyRune('j'), keyRune('k'))

	s.Equal("jk", model.FilterTerm())
	s.Equal(0, model.SelectedIndex(), "selection should not have moved")
}

func (s *FilterTestSuite) Test_a_filter_reveals_matches_inside_collapsed_groups() {
	model := s.newFilteredModel()
	// The group is collapsed, so its children are not shown.
	s.Require().NotContains(leftPaneOf(model.View()), "role")

	model.SetFilterTerm("role")

	s.Contains(
		leftPaneOf(model.View()),
		"role",
		"a match inside a collapsed group should surface",
	)
}

func (s *FilterTestSuite) Test_selection_moves_to_a_visible_item_when_filtered_out() {
	model := s.newFilteredModel()
	model.SetExpanded("group:celerity/handler:orderHandler", true)
	model.Select("lambda")
	s.Require().Equal("lambda", model.SelectedID())

	model.SetFilterTerm("role")

	s.NotEqual("lambda", model.SelectedID(), "selection should not sit on a hidden item")
	s.Require().NotNil(model.SelectedItem())
	s.Contains(
		leftPaneOf(model.View()),
		model.SelectedItem().GetName(),
		"selection is not among the items on show",
	)
}

func groupHeader(id, name string) Item {
	return &mockItem{id: id, name: name, itemType: "resource", expandable: true}
}

func child(name string) Item {
	return &mockItem{id: name, name: name, itemType: "resource", depth: 1}
}

// --- reaching items inside collapsed groups ---

func (s *FilterTestSuite) Test_selecting_an_item_inside_a_collapsed_group_expands_it() {
	model := s.newFilteredModel()
	// The group is collapsed, so only its header is shown.
	s.Require().NotContains(leftPaneOf(model.View()), "role")

	// A failed link or resource nested in a group has to be reachable, which
	// means expanding whatever encloses it rather than giving up and falling
	// back to the first visible item.
	model.Select("role")

	s.Equal("role", model.SelectedID(), "selection fell back instead of revealing the item")
	s.True(
		model.IsExpanded("group:celerity/handler:orderHandler"),
		"the enclosing group should be expanded",
	)
	s.Contains(leftPaneOf(model.View()), "role", "the item should now be on show")
}

// --- the prompt has to be on screen to be usable ---

func (s *FilterTestSuite) scrolledModel() Model {
	items := make([]Item, 0, 60)
	for i := range 60 {
		items = append(items, &mockItem{
			id:       fmt.Sprintf("res%02d", i),
			name:     fmt.Sprintf("resource%02d", i),
			itemType: "resource",
		})
	}

	model := New(Config{
		Styles: stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Title:  "Deployment",
	})
	model.SetItems(items)
	model = sendMsgs(model, tea.WindowSizeMsg{Width: 140, Height: 30})
	// Scroll to the bottom of a list far longer than the pane.
	return sendMsgs(model, tea.KeyMsg{Type: tea.KeyEnd})
}

// Scrolled away from the top, which is what makes the prompt's position matter.
func (s *FilterTestSuite) requireScrolledDown(model Model) {
	pane := leftPaneOf(model.View())
	s.Require().NotContains(pane, "resource00", "the pane should be scrolled off the top")
	s.Require().Contains(pane, "resource59")
}

func (s *FilterTestSuite) Test_the_prompt_is_shown_when_the_pane_is_scrolled_down() {
	model := s.scrolledModel()
	s.requireScrolledDown(model)

	model = sendMsgs(model, keyRune('/'))

	// The prompt is drawn at the top of the pane, so opening the filter has to
	// bring it into view or the key looks like it did nothing.
	s.Contains(leftPaneOf(model.View()), "Filter:", "the filter prompt opened off-screen")
}

func (s *FilterTestSuite) Test_the_prompt_stays_on_screen_while_typing() {
	model := typeTerm(sendMsgs(s.scrolledModel(), keyRune('/')), "resource5")

	pane := leftPaneOf(model.View())
	s.Contains(pane, "Filter:", "the prompt scrolled away as the list narrowed")
	s.Contains(pane, "resource5", "matches should be on screen with the prompt")
}

func (s *FilterTestSuite) Test_committing_the_term_hands_scrolling_back() {
	model := sendMsgs(s.scrolledModel(), keyRune('/'), tea.KeyMsg{Type: tea.KeyEnter})

	// With no term the full list is shown again, and moving to the end scrolls
	// as it normally would rather than being pinned to the top.
	model = sendMsgs(model, tea.KeyMsg{Type: tea.KeyEnd})
	s.requireScrolledDown(model)
}
