package deployui

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

// Result collection methods for DeployModel.
// These methods scan deployment items to collect successful operations,
// failures, and interrupted elements for the deployment overview.

// ResultCollector encapsulates the state needed for collecting deployment results.
// This pattern reduces parameter counts by grouping related data together.
// Exported for testing purposes.
type ResultCollector struct {
	ResourcesByName map[string]*ResourceDeployItem
	ChildrenByName  map[string]*ChildDeployItem
	LinksByName     map[string]*LinkDeployItem
	Successful      []SuccessfulElement
	Failures        []ElementFailure
	Interrupted     []InterruptedElement

	// Maps a resource's parent path and name to the abstract
	// resource it was expanded from, so links can inherit the group shared by
	// both of their endpoints.
	resourceGroups map[string]*shared.ResourceGroup
}

// resultCollector is an alias for internal use within this package.
type resultCollector = ResultCollector

// collectDeploymentResults scans all items to collect successful operations,
// failures, and interrupted elements. This provides the data for the deployment overview.
// It traverses the hierarchy to build full element paths.
func (m *DeployModel) collectDeploymentResults() {
	collector := &resultCollector{
		ResourcesByName: m.resourcesByName,
		ChildrenByName:  m.childrenByName,
		LinksByName:     m.linksByName,
	}

	collector.CollectFromItems(m.items, "")
	collector.AssignLinkGroups()

	m.successfulElements = collector.Successful
	m.elementFailures = collector.Failures
	m.interruptedElements = collector.Interrupted
}

// Re-runs result collection against items that have
// been hydrated with newer state, and republishes the results to the renderers
// that hold their own copies of them.
func (m *DeployModel) recollectDeploymentResults() {
	if !m.finished {
		return
	}

	m.collectDeploymentResults()
	m.footerRenderer.SuccessfulElements = m.successfulElements
	m.footerRenderer.ElementFailures = m.elementFailures
	m.footerRenderer.InterruptedElements = m.interruptedElements
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
}

// collectFromItems recursively collects successful operations, failures, and interruptions from items,
// building full paths as it traverses the hierarchy.
func (c *ResultCollector) CollectFromItems(items []DeployItem, parentPath string) {
	for _, item := range items {
		switch item.Type {
		case ItemTypeResource:
			if item.Resource != nil {
				path := shared.BuildElementPath(parentPath, "resources", item.Resource.Name)
				c.CollectResourceResult(item.Resource, path)
			}
		case ItemTypeChild:
			if item.Child != nil {
				path := shared.BuildElementPath(parentPath, "children", item.Child.Name)
				c.CollectChildResult(item.Child, path)

				if item.Changes != nil {
					c.CollectFromChanges(item.Changes, path, item.Child.Name)
				}
			}
		case ItemTypeLink:
			if item.Link != nil {
				path := shared.BuildElementPath(parentPath, "links", item.Link.LinkName)
				c.CollectLinkResult(item.Link, path)
			}
		}
	}
}

// collectFromChanges recursively collects results from nested blueprint changes.
// The pathPrefix is used for map key lookups (e.g., "parentChild/childName"),
// while parentPath is used for display (e.g., "children.parentChild::children.childName").
func (c *ResultCollector) CollectFromChanges(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	if blueprintChanges == nil {
		return
	}

	c.collectNestedResources(blueprintChanges, parentPath, pathPrefix)
	c.collectNestedChildren(blueprintChanges, parentPath, pathPrefix)
}

func (c *ResultCollector) collectNestedResources(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for resourceName := range blueprintChanges.NewResources {
		resourceKey := shared.BuildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.ResourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := shared.BuildElementPath(parentPath, "resources", resourceName)
			c.CollectResourceResult(resource, path)
		}
	}
	for resourceName := range blueprintChanges.ResourceChanges {
		resourceKey := shared.BuildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.ResourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := shared.BuildElementPath(parentPath, "resources", resourceName)
			c.CollectResourceResult(resource, path)
		}
	}
}

func (c *ResultCollector) collectNestedChildren(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for childName, nc := range blueprintChanges.NewChildren {
		childKey := shared.BuildMapKey(pathPrefix, childName)
		child := lookupChild(c.ChildrenByName, childKey, childName)
		if child != nil {
			path := shared.BuildElementPath(parentPath, "children", childName)
			c.CollectChildResult(child, path)

			childChanges := &changes.BlueprintChanges{
				NewResources: nc.NewResources,
				NewChildren:  nc.NewChildren,
			}
			c.CollectFromChanges(childChanges, path, childKey)
		}
	}
	for childName, cc := range blueprintChanges.ChildChanges {
		childKey := shared.BuildMapKey(pathPrefix, childName)
		child := lookupChild(c.ChildrenByName, childKey, childName)
		if child != nil {
			path := shared.BuildElementPath(parentPath, "children", childName)
			c.CollectChildResult(child, path)

			ccCopy := cc
			c.CollectFromChanges(&ccCopy, path, childKey)
		}
	}
}

// lookupResource looks up a resource by path-based key, falling back to simple name.
func lookupResource(m map[string]*ResourceDeployItem, pathKey, name string) *ResourceDeployItem {
	return shared.LookupByKey(m, pathKey, name)
}

// lookupChild looks up a child by path-based key, falling back to simple name.
func lookupChild(m map[string]*ChildDeployItem, pathKey, name string) *ChildDeployItem {
	return shared.LookupByKey(m, pathKey, name)
}

func (c *ResultCollector) CollectResourceResult(item *ResourceDeployItem, path string) {
	group := item.AbstractGroup()
	c.recordResourceGroup(item.Name, path, group)

	if IsFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "resource",
			FailureReasons: item.FailureReasons,
			AbstractGroup:  group,
		})
		return
	}
	if IsInterruptedResourceStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName:   item.Name,
			ElementPath:   path,
			ElementType:   "resource",
			AbstractGroup: group,
		})
		return
	}
	if IsSuccessResourceStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName:   item.Name,
			ElementPath:   path,
			ElementType:   "resource",
			Action:        ResourceStatusToAction(item.Status),
			AbstractGroup: group,
		})
	}
}

func (c *ResultCollector) recordResourceGroup(name, path string, group *shared.ResourceGroup) {
	if group == nil {
		return
	}
	if c.resourceGroups == nil {
		c.resourceGroups = map[string]*shared.ResourceGroup{}
	}
	parentPath := shared.ParentElementPath(path, "resources", name)
	c.resourceGroups[shared.BuildMapKey(parentPath, name)] = group
}

// AssignLinkGroups assigns each collected link the abstract resource group
// shared by both of its endpoints. Links that cross groups, or whose endpoints
// are not grouped, are left ungrouped so they stay at the top level of the
// overview rather than being filed under one side of the link.
func (c *ResultCollector) AssignLinkGroups() {
	for i := range c.Successful {
		elem := &c.Successful[i]
		elem.AbstractGroup = c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath)
	}

	for i := range c.Failures {
		elem := &c.Failures[i]
		elem.AbstractGroup = c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath)
	}

	for i := range c.Interrupted {
		elem := &c.Interrupted[i]
		elem.AbstractGroup = c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath)
	}
}

// Returns the group both endpoints of a link belong to, or the
// element's existing group when it is not a link.
func (c *ResultCollector) linkGroup(elementType, name, path string) *shared.ResourceGroup {
	if elementType != "link" || len(c.resourceGroups) == 0 {
		return c.existingGroup(elementType, name, path)
	}

	parentPath := shared.ParentElementPath(path, "links", name)
	groupAMapKey := shared.BuildMapKey(
		parentPath,
		ExtractResourceAFromLinkName(name),
	)
	groupA := c.resourceGroups[groupAMapKey]

	groupBMapKey := shared.BuildMapKey(
		parentPath,
		ExtractResourceBFromLinkName(name),
	)
	groupB := c.resourceGroups[groupBMapKey]

	if groupA == nil || groupB == nil || *groupA != *groupB {
		return nil
	}

	return groupA
}

func (c *ResultCollector) existingGroup(elementType, name, path string) *shared.ResourceGroup {
	if elementType != "resource" {
		return nil
	}

	parentPath := shared.ParentElementPath(path, "resources", name)
	return c.resourceGroups[shared.BuildMapKey(parentPath, name)]
}

func (c *ResultCollector) CollectChildResult(item *ChildDeployItem, path string) {
	if IsFailedInstanceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "child",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedInstanceStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
		return
	}
	if IsSuccessInstanceStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
			Action:      InstanceStatusToAction(item.Status),
		})
	}
}

func (c *ResultCollector) CollectLinkResult(item *LinkDeployItem, path string) {
	if IsFailedLinkStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.LinkName,
			ElementPath:    path,
			ElementType:    "link",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedLinkStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
		return
	}
	if IsSuccessLinkStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
			Action:      LinkStatusToAction(item.Status),
		})
	}
}

// Helper functions for link name parsing.

// ExtractResourceAFromLinkName extracts the first resource name from a link name (format: "resourceA::resourceB").
func ExtractResourceAFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// ExtractResourceBFromLinkName extracts the second resource name from a link name (format: "resourceA::resourceB").
func ExtractResourceBFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
