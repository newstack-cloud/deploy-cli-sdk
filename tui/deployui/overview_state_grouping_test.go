package deployui

import (
	"bytes"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// Covers the sequence a first-time deployment actually follows where results are
// collected the moment the deployment finishes, when nothing created by this
// deployment has any state yet, and the instance state carrying the transform
// annotations only arrives afterwards.
type OverviewStateGroupingTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestOverviewStateGroupingTestSuite(t *testing.T) {
	suite.Run(t, new(OverviewStateGroupingTestSuite))
}

func (s *OverviewStateGroupingTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *OverviewStateGroupingTestSuite) Test_groups_resolve_once_post_deploy_state_arrives() {
	model := s.finishedModelForFreshCreate()

	// Nothing has state at collection time, so nothing can be grouped yet.
	s.Require().Len(model.successfulElements, 3)
	for _, elem := range model.successfulElements {
		s.Nilf(elem.AbstractGroup, "%s grouped before state arrived", elem.ElementPath)
	}

	updated, _ := model.Update(PostDeployInstanceStateFetchedMsg{
		InstanceState: postDeployStateWithGroups(),
	})
	deployModel, ok := updated.(DeployModel)
	s.Require().True(ok)

	groups := map[string]string{}
	for _, elem := range deployModel.successfulElements {
		if elem.AbstractGroup != nil {
			groups[elem.ElementName] = elem.AbstractGroup.GroupName
		}
	}

	s.Equal("orderHandler", groups["orderHandler_lambda"])
	s.Equal("orderHandler", groups["orderHandler_role"])
	// The link between two resources of the same group joins that group.
	s.Equal("orderHandler", groups["orderHandler_lambda::orderHandler_role"])
}

func (s *OverviewStateGroupingTestSuite) finishedModelForFreshCreate() DeployModel {
	model := NewDeployModel(DeployModelConfig{
		Logger:         zap.NewNop(),
		ChangesetID:    "test-changeset",
		InstanceName:   "test-instance",
		BlueprintFile:  "test.blueprint.yaml",
		Styles:         s.styles,
		HeadlessWriter: &bytes.Buffer{},
	})

	// A fresh create with change set entries only, no current resource state.
	model.items = []DeployItem{
		createdResourceItem("orderHandler_lambda"),
		createdResourceItem("orderHandler_role"),
		createdLinkItem("orderHandler_lambda::orderHandler_role"),
	}
	model.finished = true
	model.collectDeploymentResults()

	return model
}

func createdResourceItem(name string) DeployItem {
	return DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:    name,
			Action:  ActionCreate,
			Status:  core.ResourceStatusCreated,
			Changes: &provider.Changes{},
		},
	}
}

func createdLinkItem(linkName string) DeployItem {
	return DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkName:      linkName,
			ResourceAName: ExtractResourceAFromLinkName(linkName),
			ResourceBName: ExtractResourceBFromLinkName(linkName),
			Action:        ActionCreate,
			Status:        core.LinkStatusCreated,
		},
	}
}

func postDeployStateWithGroups() *state.InstanceState {
	resources := map[string]*state.ResourceState{}
	resourceIDs := map[string]string{}
	for _, name := range []string{"orderHandler_lambda", "orderHandler_role"} {
		id := name + "-id"
		resourceIDs[name] = id
		resources[id] = &state.ResourceState{
			ResourceID: id,
			Name:       name,
			Metadata: &state.ResourceMetadataState{
				Annotations: map[string]*core.MappingNode{
					shared.AnnotationSourceAbstractName: core.MappingNodeFromString("orderHandler"),
					shared.AnnotationSourceAbstractType: core.MappingNodeFromString("celerity/handler"),
				},
			},
		}
	}

	return &state.InstanceState{
		InstanceID:  "test-instance-id",
		Status:      core.InstanceStatusDeployed,
		ResourceIDs: resourceIDs,
		Resources:   resources,
	}
}
