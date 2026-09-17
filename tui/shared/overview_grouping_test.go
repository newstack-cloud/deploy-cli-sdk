package shared

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type OverviewGroupingTestSuite struct {
	suite.Suite
}

func TestOverviewGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(OverviewGroupingTestSuite))
}

type overviewEntry struct {
	name  string
	group *ResourceGroup
}

func entryGroup(e overviewEntry) *ResourceGroup { return e.group }

func (s *OverviewGroupingTestSuite) Test_buckets_entries_by_group_with_ungrouped_last() {
	api := &ResourceGroup{GroupName: "myApi", GroupType: "celerity/api"}
	fn := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	entries := []overviewEntry{
		{name: "lambdaFunction", group: fn},
		{name: "standaloneBucket"},
		{name: "apiGateway", group: api},
		{name: "lambdaRole", group: fn},
	}

	groups := GroupOverviewEntries(entries, entryGroup)

	s.Require().Len(groups, 3)
	s.Equal(api, groups[0].Group)
	s.Equal([]string{"apiGateway"}, entryNames(groups[0].Entries))
	s.Equal(fn, groups[1].Group)
	// Order within a bucket is preserved.
	s.Equal([]string{"lambdaFunction", "lambdaRole"}, entryNames(groups[1].Entries))
	s.Nil(groups[2].Group)
	s.Equal([]string{"standaloneBucket"}, entryNames(groups[2].Entries))
}

func (s *OverviewGroupingTestSuite) Test_returns_single_ungrouped_bucket_when_nothing_grouped() {
	entries := []overviewEntry{{name: "bucketOne"}, {name: "bucketTwo"}}

	groups := GroupOverviewEntries(entries, entryGroup)

	s.Require().Len(groups, 1)
	s.Nil(groups[0].Group)
	s.Equal([]string{"bucketOne", "bucketTwo"}, entryNames(groups[0].Entries))
}

func (s *OverviewGroupingTestSuite) Test_group_label_matches_tree_header_format() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	s.Equal("[celerity/function] myFunc", ResourceGroupLabel(group))
	s.Equal("", ResourceGroupLabel(nil))
}

func (s *OverviewGroupingTestSuite) Test_entries_indent_only_under_a_group_header() {
	group := &ResourceGroup{GroupName: "myFunc", GroupType: "celerity/function"}
	s.Equal("    ", OverviewEntryIndent("  ", group))
	s.Equal("  ", OverviewEntryIndent("  ", nil))
}

func (s *OverviewGroupingTestSuite) Test_parent_element_path_round_trips_with_build() {
	topLevel := BuildElementPath("", "resources", "queue")
	s.Equal("", ParentElementPath(topLevel, "resources", "queue"))

	nested := BuildElementPath("children.notifications", "resources", "queue")
	s.Equal("children.notifications", ParentElementPath(nested, "resources", "queue"))

	// Link names contain the same separator used to join path segments.
	link := BuildElementPath("children.notifications", "links", "queue::handler")
	s.Equal("children.notifications", ParentElementPath(link, "links", "queue::handler"))
}

func entryNames(entries []overviewEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.name)
	}
	return names
}
