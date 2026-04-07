package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type DeployHelpersTestSuite struct {
	suite.Suite
}

func TestDeployHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(DeployHelpersTestSuite))
}

func (s *DeployHelpersTestSuite) Test_Items_returns_nil_when_no_items() {
	m := &DeployModel{}
	s.Nil(m.Items())
}

func (s *DeployHelpersTestSuite) Test_Items_returns_items_slice() {
	m := &DeployModel{
		items: []DeployItem{
			{Type: ItemTypeResource, Resource: &ResourceDeployItem{Name: "r1"}},
			{Type: ItemTypeChild, Child: &ChildDeployItem{Name: "c1"}},
		},
	}
	items := m.Items()
	s.Len(items, 2)
	s.Equal("r1", items[0].Resource.Name)
	s.Equal("c1", items[1].Child.Name)
}

func (s *DeployHelpersTestSuite) Test_ResourceDeployItem_SetSkipped_sets_field() {
	r := &ResourceDeployItem{Name: "r"}
	s.False(r.Skipped)
	r.SetSkipped(true)
	s.True(r.Skipped)
	r.SetSkipped(false)
	s.False(r.Skipped)
}

func (s *DeployHelpersTestSuite) Test_ChildDeployItem_SetSkipped_sets_field() {
	c := &ChildDeployItem{Name: "c"}
	s.False(c.Skipped)
	c.SetSkipped(true)
	s.True(c.Skipped)
	c.SetSkipped(false)
	s.False(c.Skipped)
}

func (s *DeployHelpersTestSuite) Test_ChildDeployItem_GetAction_returns_action() {
	c := &ChildDeployItem{Action: ActionUpdate}
	s.Equal(ActionUpdate, c.GetAction())
}

func (s *DeployHelpersTestSuite) Test_ChildDeployItem_GetChildStatus_returns_status() {
	c := &ChildDeployItem{Status: core.InstanceStatusDeployed}
	s.Equal(core.InstanceStatusDeployed, c.GetChildStatus())
}

func (s *DeployHelpersTestSuite) Test_LinkDeployItem_SetSkipped_sets_field() {
	l := &LinkDeployItem{LinkName: "a::b"}
	s.False(l.Skipped)
	l.SetSkipped(true)
	s.True(l.Skipped)
	l.SetSkipped(false)
	s.False(l.Skipped)
}

func (s *DeployHelpersTestSuite) Test_LinkDeployItem_GetAction_returns_action() {
	l := &LinkDeployItem{Action: ActionCreate}
	s.Equal(ActionCreate, l.GetAction())
}

func (s *DeployHelpersTestSuite) Test_LinkDeployItem_GetLinkStatus_returns_status() {
	l := &LinkDeployItem{Status: core.LinkStatusCreated}
	s.Equal(core.LinkStatusCreated, l.GetLinkStatus())
}

func (s *DeployHelpersTestSuite) Test_CollectChildResult_adds_failed_child() {
	c := &ResultCollector{}
	item := &ChildDeployItem{
		Name:           "child-a",
		Status:         core.InstanceStatusDeployFailed,
		FailureReasons: []string{"deployment failed"},
	}
	c.CollectChildResult(item, "children.child-a")

	s.Len(c.Failures, 1)
	s.Equal("child-a", c.Failures[0].ElementName)
	s.Equal("children.child-a", c.Failures[0].ElementPath)
	s.Equal("child", c.Failures[0].ElementType)
	s.Len(c.Failures[0].FailureReasons, 1)
	s.Empty(c.Successful)
	s.Empty(c.Interrupted)
}

func (s *DeployHelpersTestSuite) Test_CollectChildResult_adds_interrupted_child() {
	c := &ResultCollector{}
	item := &ChildDeployItem{
		Name:   "child-b",
		Status: core.InstanceStatusDeployInterrupted,
	}
	c.CollectChildResult(item, "children.child-b")

	s.Len(c.Interrupted, 1)
	s.Equal("child-b", c.Interrupted[0].ElementName)
	s.Equal("child", c.Interrupted[0].ElementType)
	s.Empty(c.Successful)
	s.Empty(c.Failures)
}

func (s *DeployHelpersTestSuite) Test_CollectChildResult_adds_successful_child() {
	c := &ResultCollector{}
	item := &ChildDeployItem{
		Name:   "child-c",
		Status: core.InstanceStatusDeployed,
	}
	c.CollectChildResult(item, "children.child-c")

	s.Len(c.Successful, 1)
	s.Equal("child-c", c.Successful[0].ElementName)
	s.Equal("deployed", c.Successful[0].Action)
	s.Equal("child", c.Successful[0].ElementType)
	s.Empty(c.Failures)
	s.Empty(c.Interrupted)
}

func (s *DeployHelpersTestSuite) Test_CollectChildResult_no_action_for_failed_child_without_reasons() {
	c := &ResultCollector{}
	item := &ChildDeployItem{
		Name:           "child-d",
		Status:         core.InstanceStatusDeployFailed,
		FailureReasons: nil,
	}
	c.CollectChildResult(item, "children.child-d")

	s.Empty(c.Failures)
	s.Empty(c.Successful)
	s.Empty(c.Interrupted)
}

func (s *DeployHelpersTestSuite) Test_CollectLinkResult_adds_failed_link() {
	c := &ResultCollector{}
	item := &LinkDeployItem{
		LinkName:       "resA::resB",
		Status:         core.LinkStatusCreateFailed,
		FailureReasons: []string{"link error"},
	}
	c.CollectLinkResult(item, "links.resA::resB")

	s.Len(c.Failures, 1)
	s.Equal("resA::resB", c.Failures[0].ElementName)
	s.Equal("link", c.Failures[0].ElementType)
	s.Empty(c.Successful)
	s.Empty(c.Interrupted)
}

func (s *DeployHelpersTestSuite) Test_CollectLinkResult_adds_interrupted_link() {
	c := &ResultCollector{}
	item := &LinkDeployItem{
		LinkName: "resA::resB",
		Status:   core.LinkStatusCreateInterrupted,
	}
	c.CollectLinkResult(item, "links.resA::resB")

	s.Len(c.Interrupted, 1)
	s.Equal("resA::resB", c.Interrupted[0].ElementName)
	s.Equal("link", c.Interrupted[0].ElementType)
	s.Empty(c.Successful)
	s.Empty(c.Failures)
}

func (s *DeployHelpersTestSuite) Test_CollectLinkResult_adds_successful_link() {
	c := &ResultCollector{}
	item := &LinkDeployItem{
		LinkName: "resA::resB",
		Status:   core.LinkStatusCreated,
	}
	c.CollectLinkResult(item, "links.resA::resB")

	s.Len(c.Successful, 1)
	s.Equal("resA::resB", c.Successful[0].ElementName)
	s.Equal("created", c.Successful[0].Action)
	s.Equal("link", c.Successful[0].ElementType)
	s.Empty(c.Failures)
	s.Empty(c.Interrupted)
}

func (s *DeployHelpersTestSuite) Test_CollectFromChanges_nil_changes_is_noop() {
	c := &ResultCollector{}
	c.CollectFromChanges(nil, "children.parent", "parent")
	s.Empty(c.Successful)
}

func (s *DeployHelpersTestSuite) Test_CollectFromChanges_collects_new_resources_in_nested_child() {
	resourceItem := &ResourceDeployItem{Name: "nested-res", Status: core.ResourceStatusCreated}
	c := &ResultCollector{
		ResourcesByName: map[string]*ResourceDeployItem{
			"parent/nested-res": resourceItem,
		},
		ChildrenByName: map[string]*ChildDeployItem{},
		LinksByName:    map[string]*LinkDeployItem{},
	}

	bpChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"nested-res": {},
		},
	}

	c.CollectFromChanges(bpChanges, "children.parent", "parent")

	s.Len(c.Successful, 1)
	s.Equal("nested-res", c.Successful[0].ElementName)
}

func (s *DeployHelpersTestSuite) Test_CollectFromChanges_collects_nested_child_with_NewChildren() {
	childItem := &ChildDeployItem{Name: "inner-child", Status: core.InstanceStatusDeployed}
	c := &ResultCollector{
		ResourcesByName: map[string]*ResourceDeployItem{},
		ChildrenByName: map[string]*ChildDeployItem{
			"parent/inner-child": childItem,
		},
		LinksByName: map[string]*LinkDeployItem{},
	}

	bpChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"inner-child": {},
		},
	}

	c.CollectFromChanges(bpChanges, "children.parent", "parent")

	s.Len(c.Successful, 1)
	s.Equal("inner-child", c.Successful[0].ElementName)
	s.Equal("child", c.Successful[0].ElementType)
}

func (s *DeployHelpersTestSuite) Test_CollectFromChanges_collects_nested_child_with_ChildChanges() {
	childItem := &ChildDeployItem{Name: "inner-child", Status: core.InstanceStatusUpdated}
	c := &ResultCollector{
		ResourcesByName: map[string]*ResourceDeployItem{},
		ChildrenByName: map[string]*ChildDeployItem{
			"parent/inner-child": childItem,
		},
		LinksByName: map[string]*LinkDeployItem{},
	}

	bpChanges := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"inner-child": {},
		},
	}

	c.CollectFromChanges(bpChanges, "children.parent", "parent")

	s.Len(c.Successful, 1)
	s.Equal("inner-child", c.Successful[0].ElementName)
	s.Equal("updated", c.Successful[0].Action)
}
