package shared

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type mockSkippableResource struct {
	action  ActionType
	status  core.ResourceStatus
	skipped bool
}

func (m *mockSkippableResource) GetAction() ActionType          { return m.action }
func (m *mockSkippableResource) GetResourceStatus() core.ResourceStatus { return m.status }
func (m *mockSkippableResource) SetSkipped(s bool)              { m.skipped = s }

type mockSkippableChild struct {
	action  ActionType
	status  core.InstanceStatus
	skipped bool
}

func (m *mockSkippableChild) GetAction() ActionType         { return m.action }
func (m *mockSkippableChild) GetChildStatus() core.InstanceStatus { return m.status }
func (m *mockSkippableChild) SetSkipped(s bool)             { m.skipped = s }

type mockSkippableLink struct {
	action  ActionType
	status  core.LinkStatus
	skipped bool
}

func (m *mockSkippableLink) GetAction() ActionType      { return m.action }
func (m *mockSkippableLink) GetLinkStatus() core.LinkStatus { return m.status }
func (m *mockSkippableLink) SetSkipped(s bool)          { m.skipped = s }

type ItemIconsStatusSuite struct {
	suite.Suite
	testStyles *stylespkg.Styles
}

func TestItemIconsStatusSuite(t *testing.T) {
	suite.Run(t, new(ItemIconsStatusSuite))
}

func (s *ItemIconsStatusSuite) SetupTest() {
	s.testStyles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_creating_returns_inprogress() {
	s.Equal(IconInProgress, ResourceStatusIcon(core.ResourceStatusCreating))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_updating_returns_inprogress() {
	s.Equal(IconInProgress, ResourceStatusIcon(core.ResourceStatusUpdating))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_destroying_returns_inprogress() {
	s.Equal(IconInProgress, ResourceStatusIcon(core.ResourceStatusDestroying))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_created_returns_success() {
	s.Equal(IconSuccess, ResourceStatusIcon(core.ResourceStatusCreated))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_updated_returns_success() {
	s.Equal(IconSuccess, ResourceStatusIcon(core.ResourceStatusUpdated))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_destroyed_returns_success() {
	s.Equal(IconSuccess, ResourceStatusIcon(core.ResourceStatusDestroyed))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_createFailed_returns_failed() {
	s.Equal(IconFailed, ResourceStatusIcon(core.ResourceStatusCreateFailed))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_updateFailed_returns_failed() {
	s.Equal(IconFailed, ResourceStatusIcon(core.ResourceStatusUpdateFailed))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_destroyFailed_returns_failed() {
	s.Equal(IconFailed, ResourceStatusIcon(core.ResourceStatusDestroyFailed))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_rollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, ResourceStatusIcon(core.ResourceStatusRollingBack))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_rollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, ResourceStatusIcon(core.ResourceStatusRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_rollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, ResourceStatusIcon(core.ResourceStatusRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_createInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, ResourceStatusIcon(core.ResourceStatusCreateInterrupted))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_updateInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, ResourceStatusIcon(core.ResourceStatusUpdateInterrupted))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_destroyInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, ResourceStatusIcon(core.ResourceStatusDestroyInterrupted))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_unknown_returns_pending() {
	s.Equal(IconPending, ResourceStatusIcon(core.ResourceStatusUnknown))
}

func (s *ItemIconsStatusSuite) Test_ResourceStatusIcon_default_returns_pending() {
	s.Equal(IconPending, ResourceStatusIcon(core.ResourceStatus(9999)))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_preparing_returns_pending() {
	s.Equal(IconPending, InstanceStatusIcon(core.InstanceStatusPreparing))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deploying_returns_inprogress() {
	s.Equal(IconInProgress, InstanceStatusIcon(core.InstanceStatusDeploying))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updating_returns_inprogress() {
	s.Equal(IconInProgress, InstanceStatusIcon(core.InstanceStatusUpdating))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroying_returns_inprogress() {
	s.Equal(IconInProgress, InstanceStatusIcon(core.InstanceStatusDestroying))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployed_returns_success() {
	s.Equal(IconSuccess, InstanceStatusIcon(core.InstanceStatusDeployed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updated_returns_success() {
	s.Equal(IconSuccess, InstanceStatusIcon(core.InstanceStatusUpdated))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyed_returns_success() {
	s.Equal(IconSuccess, InstanceStatusIcon(core.InstanceStatusDestroyed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployFailed_returns_failed() {
	s.Equal(IconFailed, InstanceStatusIcon(core.InstanceStatusDeployFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updateFailed_returns_failed() {
	s.Equal(IconFailed, InstanceStatusIcon(core.InstanceStatusUpdateFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyFailed_returns_failed() {
	s.Equal(IconFailed, InstanceStatusIcon(core.InstanceStatusDestroyFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, InstanceStatusIcon(core.InstanceStatusDeployRollingBack))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updateRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, InstanceStatusIcon(core.InstanceStatusUpdateRollingBack))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, InstanceStatusIcon(core.InstanceStatusDestroyRollingBack))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, InstanceStatusIcon(core.InstanceStatusDeployRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updateRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, InstanceStatusIcon(core.InstanceStatusUpdateRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, InstanceStatusIcon(core.InstanceStatusDestroyRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, InstanceStatusIcon(core.InstanceStatusDeployRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updateRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, InstanceStatusIcon(core.InstanceStatusUpdateRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, InstanceStatusIcon(core.InstanceStatusDestroyRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_deployInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, InstanceStatusIcon(core.InstanceStatusDeployInterrupted))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_updateInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, InstanceStatusIcon(core.InstanceStatusUpdateInterrupted))
}

func (s *ItemIconsStatusSuite) Test_InstanceStatusIcon_destroyInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, InstanceStatusIcon(core.InstanceStatusDestroyInterrupted))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_creating_returns_inprogress() {
	s.Equal(IconInProgress, LinkStatusIcon(core.LinkStatusCreating))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updating_returns_inprogress() {
	s.Equal(IconInProgress, LinkStatusIcon(core.LinkStatusUpdating))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroying_returns_inprogress() {
	s.Equal(IconInProgress, LinkStatusIcon(core.LinkStatusDestroying))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_created_returns_success() {
	s.Equal(IconSuccess, LinkStatusIcon(core.LinkStatusCreated))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updated_returns_success() {
	s.Equal(IconSuccess, LinkStatusIcon(core.LinkStatusUpdated))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyed_returns_success() {
	s.Equal(IconSuccess, LinkStatusIcon(core.LinkStatusDestroyed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_createFailed_returns_failed() {
	s.Equal(IconFailed, LinkStatusIcon(core.LinkStatusCreateFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updateFailed_returns_failed() {
	s.Equal(IconFailed, LinkStatusIcon(core.LinkStatusUpdateFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyFailed_returns_failed() {
	s.Equal(IconFailed, LinkStatusIcon(core.LinkStatusDestroyFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_createRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, LinkStatusIcon(core.LinkStatusCreateRollingBack))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updateRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, LinkStatusIcon(core.LinkStatusUpdateRollingBack))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyRollingBack_returns_rollingBack() {
	s.Equal(IconRollingBack, LinkStatusIcon(core.LinkStatusDestroyRollingBack))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_createRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, LinkStatusIcon(core.LinkStatusCreateRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updateRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, LinkStatusIcon(core.LinkStatusUpdateRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyRollbackFailed_returns_rollbackFailed() {
	s.Equal(IconRollbackFailed, LinkStatusIcon(core.LinkStatusDestroyRollbackFailed))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_createRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, LinkStatusIcon(core.LinkStatusCreateRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updateRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, LinkStatusIcon(core.LinkStatusUpdateRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyRollbackComplete_returns_rollbackComplete() {
	s.Equal(IconRollbackComplete, LinkStatusIcon(core.LinkStatusDestroyRollbackComplete))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_createInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, LinkStatusIcon(core.LinkStatusCreateInterrupted))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_updateInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, LinkStatusIcon(core.LinkStatusUpdateInterrupted))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_destroyInterrupted_returns_interrupted() {
	s.Equal(IconInterrupted, LinkStatusIcon(core.LinkStatusDestroyInterrupted))
}

func (s *ItemIconsStatusSuite) Test_LinkStatusIcon_unknown_returns_pending() {
	s.Equal(IconPending, LinkStatusIcon(core.LinkStatusUnknown))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_inprogress_returns_nonempty_containing_icon() {
	icon := IconInProgress
	result := StyleResourceIcon(icon, core.ResourceStatusCreating, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_success_returns_nonempty_containing_icon() {
	icon := IconSuccess
	result := StyleResourceIcon(icon, core.ResourceStatusCreated, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_failed_returns_nonempty_containing_icon() {
	icon := IconFailed
	result := StyleResourceIcon(icon, core.ResourceStatusCreateFailed, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_rollingBack_returns_nonempty_containing_icon() {
	icon := IconRollingBack
	result := StyleResourceIcon(icon, core.ResourceStatusRollingBack, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_rollbackComplete_returns_nonempty_containing_icon() {
	icon := IconRollbackComplete
	result := StyleResourceIcon(icon, core.ResourceStatusRollbackComplete, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleResourceIcon_pending_default_returns_nonempty_containing_icon() {
	icon := IconPending
	result := StyleResourceIcon(icon, core.ResourceStatusUnknown, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_inprogress_returns_nonempty_containing_icon() {
	icon := IconInProgress
	result := StyleInstanceIcon(icon, core.InstanceStatusDeploying, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_success_returns_nonempty_containing_icon() {
	icon := IconSuccess
	result := StyleInstanceIcon(icon, core.InstanceStatusDeployed, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_failed_returns_nonempty_containing_icon() {
	icon := IconFailed
	result := StyleInstanceIcon(icon, core.InstanceStatusDeployFailed, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_rollingBack_returns_nonempty_containing_icon() {
	icon := IconRollingBack
	result := StyleInstanceIcon(icon, core.InstanceStatusDeployRollingBack, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_rollbackComplete_returns_nonempty_containing_icon() {
	icon := IconRollbackComplete
	result := StyleInstanceIcon(icon, core.InstanceStatusDeployRollbackComplete, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleInstanceIcon_pending_default_returns_nonempty_containing_icon() {
	icon := IconPending
	result := StyleInstanceIcon(icon, core.InstanceStatusPreparing, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_inprogress_returns_nonempty_containing_icon() {
	icon := IconInProgress
	result := StyleLinkIcon(icon, core.LinkStatusCreating, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_success_returns_nonempty_containing_icon() {
	icon := IconSuccess
	result := StyleLinkIcon(icon, core.LinkStatusCreated, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_failed_returns_nonempty_containing_icon() {
	icon := IconFailed
	result := StyleLinkIcon(icon, core.LinkStatusCreateFailed, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_rollingBack_returns_nonempty_containing_icon() {
	icon := IconRollingBack
	result := StyleLinkIcon(icon, core.LinkStatusCreateRollingBack, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_rollbackComplete_returns_nonempty_containing_icon() {
	icon := IconRollbackComplete
	result := StyleLinkIcon(icon, core.LinkStatusCreateRollbackComplete, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_StyleLinkIcon_pending_default_returns_nonempty_containing_icon() {
	icon := IconPending
	result := StyleLinkIcon(icon, core.LinkStatusUnknown, s.testStyles)
	s.NotEmpty(result)
	s.True(strings.Contains(result, icon))
}

func (s *ItemIconsStatusSuite) Test_IsPendingResourceStatus_unknown_returns_true() {
	s.True(IsPendingResourceStatus(core.ResourceStatusUnknown))
}

func (s *ItemIconsStatusSuite) Test_IsPendingResourceStatus_creating_returns_false() {
	s.False(IsPendingResourceStatus(core.ResourceStatusCreating))
}

func (s *ItemIconsStatusSuite) Test_IsPendingResourceStatus_created_returns_false() {
	s.False(IsPendingResourceStatus(core.ResourceStatusCreated))
}

func (s *ItemIconsStatusSuite) Test_IsPendingChildStatus_preparing_returns_true() {
	s.True(IsPendingChildStatus(core.InstanceStatusPreparing))
}

func (s *ItemIconsStatusSuite) Test_IsPendingChildStatus_notDeployed_returns_true() {
	s.True(IsPendingChildStatus(core.InstanceStatusNotDeployed))
}

func (s *ItemIconsStatusSuite) Test_IsPendingChildStatus_deploying_returns_false() {
	s.False(IsPendingChildStatus(core.InstanceStatusDeploying))
}

func (s *ItemIconsStatusSuite) Test_IsPendingLinkStatus_unknown_returns_true() {
	s.True(IsPendingLinkStatus(core.LinkStatusUnknown))
}

func (s *ItemIconsStatusSuite) Test_IsPendingLinkStatus_creating_returns_false() {
	s.False(IsPendingLinkStatus(core.LinkStatusCreating))
}

func (s *ItemIconsStatusSuite) Test_MarkPendingResourcesAsSkipped_pending_nonNoChange_gets_skipped() {
	item := &mockSkippableResource{
		action: ActionCreate,
		status: core.ResourceStatusUnknown,
	}
	resources := map[string]*mockSkippableResource{"res1": item}
	MarkPendingResourcesAsSkipped(resources)
	s.True(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingResourcesAsSkipped_noChange_action_not_skipped() {
	item := &mockSkippableResource{
		action: ActionNoChange,
		status: core.ResourceStatusUnknown,
	}
	resources := map[string]*mockSkippableResource{"res1": item}
	MarkPendingResourcesAsSkipped(resources)
	s.False(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingResourcesAsSkipped_nonPending_not_skipped() {
	item := &mockSkippableResource{
		action: ActionCreate,
		status: core.ResourceStatusCreating,
	}
	resources := map[string]*mockSkippableResource{"res1": item}
	MarkPendingResourcesAsSkipped(resources)
	s.False(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingChildrenAsSkipped_pending_nonNoChange_gets_skipped() {
	item := &mockSkippableChild{
		action: ActionCreate,
		status: core.InstanceStatusPreparing,
	}
	children := map[string]*mockSkippableChild{"child1": item}
	MarkPendingChildrenAsSkipped(children)
	s.True(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingChildrenAsSkipped_notDeployed_gets_skipped() {
	item := &mockSkippableChild{
		action: ActionCreate,
		status: core.InstanceStatusNotDeployed,
	}
	children := map[string]*mockSkippableChild{"child1": item}
	MarkPendingChildrenAsSkipped(children)
	s.True(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingChildrenAsSkipped_noChange_action_not_skipped() {
	item := &mockSkippableChild{
		action: ActionNoChange,
		status: core.InstanceStatusPreparing,
	}
	children := map[string]*mockSkippableChild{"child1": item}
	MarkPendingChildrenAsSkipped(children)
	s.False(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingChildrenAsSkipped_nonPending_not_skipped() {
	item := &mockSkippableChild{
		action: ActionCreate,
		status: core.InstanceStatusDeploying,
	}
	children := map[string]*mockSkippableChild{"child1": item}
	MarkPendingChildrenAsSkipped(children)
	s.False(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingLinksAsSkipped_pending_nonNoChange_gets_skipped() {
	item := &mockSkippableLink{
		action: ActionCreate,
		status: core.LinkStatusUnknown,
	}
	links := map[string]*mockSkippableLink{"link1": item}
	MarkPendingLinksAsSkipped(links)
	s.True(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingLinksAsSkipped_noChange_action_not_skipped() {
	item := &mockSkippableLink{
		action: ActionNoChange,
		status: core.LinkStatusUnknown,
	}
	links := map[string]*mockSkippableLink{"link1": item}
	MarkPendingLinksAsSkipped(links)
	s.False(item.skipped)
}

func (s *ItemIconsStatusSuite) Test_MarkPendingLinksAsSkipped_nonPending_not_skipped() {
	item := &mockSkippableLink{
		action: ActionCreate,
		status: core.LinkStatusCreating,
	}
	links := map[string]*mockSkippableLink{"link1": item}
	MarkPendingLinksAsSkipped(links)
	s.False(item.skipped)
}
