package inspectui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/deployui"
	"github.com/stretchr/testify/suite"
)

// Inspecting a failed deployment is how you find out why it failed, so the
// reasons state recorded have to reach the detail pane. Items here are built
// from instance state rather than from a live event stream.
type FailureReasonsTestSuite struct {
	suite.Suite
	styles   *stylespkg.Styles
	linkName string
	instance *state.InstanceState
}

func TestFailureReasonsTestSuite(t *testing.T) {
	suite.Run(t, new(FailureReasonsTestSuite))
}

func (s *FailureReasonsTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
	s.linkName = "ordersApi_http_api::createOrder_lambda"
	s.instance = &state.InstanceState{
		InstanceID:  "i-1",
		Status:      core.InstanceStatusDeployFailed,
		ResourceIDs: map[string]string{"createOrder_lambda": "r1"},
		Resources: map[string]*state.ResourceState{
			"r1": {
				ResourceID: "r1", Name: "createOrder_lambda", Type: "aws/lambda/function",
				Status:         core.ResourceStatusCreateFailed,
				FailureReasons: []string{"lambda create failed: handler not found"},
			},
		},
		Links: map[string]*state.LinkState{
			s.linkName: {
				LinkID: "l1", Name: s.linkName,
				Status:         core.LinkStatusCreateFailed,
				FailureReasons: []string{"link create failed: AccessDenied on iam:PutRolePolicy"},
			},
		},
	}
}

func (s *FailureReasonsTestSuite) items() []deployui.DeployItem {
	return buildItemsFromInstanceState(
		s.instance,
		map[string]*deployui.ResourceDeployItem{},
		map[string]*deployui.ChildDeployItem{},
		map[string]*deployui.LinkDeployItem{},
	)
}

func (s *FailureReasonsTestSuite) renderItem(id string) string {
	items := s.items()
	renderer := &InspectDetailsRenderer{MaxExpandDepth: 2, InstanceState: s.instance}
	for i := range items {
		if items[i].GetID() == id {
			return renderer.RenderDetails(&items[i], 90, s.styles)
		}
	}
	s.Failf("item missing", "no item with id %q", id)
	return ""
}

func (s *FailureReasonsTestSuite) Test_items_carry_the_reasons_recorded_in_state() {
	for _, item := range s.items() {
		switch item.Type {
		case deployui.ItemTypeResource:
			s.NotEmpty(item.Resource.FailureReasons, "resource lost its failure reasons")
		case deployui.ItemTypeLink:
			s.NotEmpty(item.Link.FailureReasons, "link lost its failure reasons")
		}
	}
}

func (s *FailureReasonsTestSuite) Test_a_failed_link_shows_why_it_failed() {
	out := s.renderItem(s.linkName)

	s.Contains(out, "Failure Reasons")
	s.Contains(out, "AccessDenied on iam:PutRolePolicy")
}

func (s *FailureReasonsTestSuite) Test_a_failed_resource_shows_why_it_failed() {
	out := s.renderItem("createOrder_lambda")

	s.Contains(out, "Failure Reasons")
	s.Contains(out, "handler not found")
}

func (s *FailureReasonsTestSuite) Test_nothing_is_shown_when_there_was_no_failure() {
	s.instance.Links[s.linkName].Status = core.LinkStatusCreated
	s.instance.Links[s.linkName].FailureReasons = nil

	s.NotContains(s.renderItem(s.linkName), "Failure Reasons")
}

func (s *FailureReasonsTestSuite) Test_reasons_from_a_live_stream_take_precedence() {
	// A streaming inspect fills reasons in from events, which are fresher than
	// whatever state held when the inspect started.
	res := &deployui.ResourceDeployItem{
		Name:           "createOrder_lambda",
		Status:         core.ResourceStatusCreateFailed,
		FailureReasons: []string{"reported by the event stream"},
	}
	item := &deployui.DeployItem{
		Type: deployui.ItemTypeResource, Resource: res, InstanceState: s.instance,
	}

	renderer := &InspectDetailsRenderer{MaxExpandDepth: 2, InstanceState: s.instance}
	out := renderer.RenderDetails(item, 90, s.styles)

	s.Contains(out, "reported by the event stream")
	s.NotContains(out, "handler not found")
}
