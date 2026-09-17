package shared

import (
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type StatusKeywordsTestSuite struct {
	suite.Suite
}

func TestStatusKeywordsTestSuite(t *testing.T) {
	suite.Run(t, new(StatusKeywordsTestSuite))
}

func (s *StatusKeywordsTestSuite) Test_each_status_icon_has_keywords() {
	// Anything the tree can render should be filterable, or the filter is
	// silently blind to whole categories of item.
	for _, icon := range []string{
		IconPending, IconInProgress, IconSuccess, IconFailed,
		IconRollingBack, IconRollbackFailed, IconRollbackComplete,
		IconInterrupted, IconSkipped, IconNoChange, IconRetained,
	} {
		s.NotEmptyf(StatusKeywordsForIcon(icon), "icon %q has no filter keywords", icon)
	}
}

func (s *StatusKeywordsTestSuite) Test_rollback_failures_are_found_by_failed() {
	// Someone looking for what went wrong should not have to know the
	// difference between a failure and a failed rollback.
	s.Contains(StatusKeywordsForIcon(IconRollbackFailed), "failed")
}

func (s *StatusKeywordsTestSuite) Test_an_items_keywords_come_from_its_icon() {
	item := iconItem("lambda", IconFailed)
	s.Equal(StatusKeywordsForIcon(IconFailed), StatusKeywordsForItem(item))
}

// A group header shows the most severe status among its contents, not one of
// its own. Letting it match would pull in every resource beside the failure.
func (s *StatusKeywordsTestSuite) Test_a_group_header_reports_no_status_of_its_own() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{iconItem("lambda", IconFailed)},
	}
	s.Require().Equal(IconFailed, group.GetIcon(false), "the header still shows the failure")
	s.Empty(StatusKeywordsForItem(group), "but it should not match a status filter itself")
}

func (s *StatusKeywordsTestSuite) Test_a_wrapped_group_header_is_still_a_group_header() {
	group := &ResourceGroupItem{
		Children: []splitpane.Item{iconItem("lambda", IconFailed)},
	}
	wrapped := &DepthAdjustedItem{Item: group, AdjustedDepth: 1}

	s.Empty(StatusKeywordsForItem(wrapped))
}

func (s *StatusKeywordsTestSuite) Test_the_suggested_keywords_are_all_real() {
	for _, keyword := range KnownStatusKeywords() {
		found := false
		for _, keywords := range statusKeywords {
			for _, candidate := range keywords {
				if candidate == keyword {
					found = true
				}
			}
		}
		s.Truef(found, "suggested keyword %q matches no status", keyword)
	}
}
