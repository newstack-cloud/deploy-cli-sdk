package destroyui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

// Encapsulates the state needed for collecting destroy results.
type resultCollector struct {
	resourcesByName map[string]*ResourceDestroyItem
	childrenByName  map[string]*ChildDestroyItem
	linksByName     map[string]*LinkDestroyItem
	destroyed       []DestroyedElement
	failures        []ElementFailure
	interrupted     []InterruptedElement
	retained        []RetainedElement

	// Maps a resource's parent path and name to the abstract
	// resource it was expanded from, so links can inherit the group shared by
	// both of their endpoints.
	resourceGroups map[string]*shared.ResourceGroup
}

// Scans all items to collect destroyed elements,
// failures, and interrupted elements. This provides the data for the destroy overview.
// It traverses the hierarchy to build full element paths.
func (m *DestroyModel) collectDestroyResults() {
	collector := &resultCollector{
		resourcesByName: m.resourcesByName,
		childrenByName:  m.childrenByName,
		linksByName:     m.linksByName,
	}

	collector.collectFromItems(m.items, "")
	collector.assignLinkGroups()

	m.destroyedElements = collector.destroyed
	m.elementFailures = collector.failures
	m.interruptedElements = collector.interrupted
	m.retainedElements = collector.retained
}

// Recursively collects destroyed, failed, and interrupted elements,
// building full paths as it traverses the hierarchy.
func (c *resultCollector) collectFromItems(items []DestroyItem, parentPath string) {
	for _, item := range items {
		switch item.Type {
		case ItemTypeResource:
			if item.Resource != nil {
				path := shared.BuildElementPath(parentPath, "resources", item.Resource.Name)
				c.collectResourceResult(item.Resource, path)
			}
		case ItemTypeChild:
			if item.Child != nil {
				path := shared.BuildElementPath(parentPath, "children", item.Child.Name)
				c.collectChildResult(item.Child, path)

				if item.Changes != nil {
					c.collectFromChanges(item.Changes, path, item.Child.Name)
				}
			}
		case ItemTypeLink:
			if item.Link != nil {
				path := shared.BuildElementPath(parentPath, "links", item.Link.LinkName)
				c.collectLinkResult(item.Link, path)
			}
		}
	}
}

// collectFromChanges recursively collects results from nested blueprint changes.
func (c *resultCollector) collectFromChanges(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	if bpChanges == nil {
		return
	}

	c.collectNestedResources(bpChanges, parentPath, pathPrefix)
	c.collectNestedLinks(bpChanges, parentPath, pathPrefix)
	c.collectNestedChildren(bpChanges, parentPath, pathPrefix)
}

func (c *resultCollector) collectNestedResources(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, resourceName := range bpChanges.RemovedResources {
		resourceKey := shared.BuildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := shared.BuildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
	for resourceName := range bpChanges.ResourceChanges {
		resourceKey := shared.BuildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := shared.BuildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
}

func (c *resultCollector) collectNestedLinks(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, linkName := range bpChanges.RemovedLinks {
		linkKey := shared.BuildMapKey(pathPrefix, linkName)
		link := lookupLink(c.linksByName, linkKey, linkName)
		if link != nil {
			path := shared.BuildElementPath(parentPath, "links", linkName)
			c.collectLinkResult(link, path)
		}
	}
}

func (c *resultCollector) collectNestedChildren(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, childName := range bpChanges.RemovedChildren {
		childKey := shared.BuildMapKey(pathPrefix, childName)
		child := lookupChild(c.childrenByName, childKey, childName)
		if child != nil {
			path := shared.BuildElementPath(parentPath, "children", childName)
			c.collectChildResult(child, path)

			if childChanges, ok := bpChanges.ChildChanges[childName]; ok {
				c.collectFromChanges(&childChanges, path, childKey)
			}
		}
	}
	for childName, cc := range bpChanges.ChildChanges {
		childKey := shared.BuildMapKey(pathPrefix, childName)
		child := lookupChild(c.childrenByName, childKey, childName)
		if child != nil {
			path := shared.BuildElementPath(parentPath, "children", childName)
			c.collectChildResult(child, path)

			ccCopy := cc
			c.collectFromChanges(&ccCopy, path, childKey)
		}
	}
}

func lookupResource(m map[string]*ResourceDestroyItem, pathKey, name string) *ResourceDestroyItem {
	return shared.LookupByKey(m, pathKey, name)
}

func lookupChild(m map[string]*ChildDestroyItem, pathKey, name string) *ChildDestroyItem {
	return shared.LookupByKey(m, pathKey, name)
}

func lookupLink(m map[string]*LinkDestroyItem, pathKey, name string) *LinkDestroyItem {
	return shared.LookupByKey(m, pathKey, name)
}

func (c *resultCollector) collectResourceResult(item *ResourceDestroyItem, path string) {
	group := item.AbstractGroup()
	c.recordResourceGroup(item.Name, path, group)

	if IsFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    item.ResourceType,
			FailureReasons: item.FailureReasons,
			AbstractGroup:  group,
		})
		return
	}
	if IsInterruptedResourceStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName:   item.Name,
			ElementPath:   path,
			ElementType:   item.ResourceType,
			AbstractGroup: group,
		})
		return
	}
	if IsRetainedResourceStatus(item.Status) {
		c.retained = append(c.retained, RetainedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: item.ResourceType,
		})
		return
	}
	if IsSuccessResourceStatus(item.Status) {
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName:   item.Name,
			ElementPath:   path,
			ElementType:   item.ResourceType,
			AbstractGroup: group,
		})
	}
}

func (c *resultCollector) recordResourceGroup(name, path string, group *shared.ResourceGroup) {
	if group == nil {
		return
	}
	if c.resourceGroups == nil {
		c.resourceGroups = map[string]*shared.ResourceGroup{}
	}
	parentPath := shared.ParentElementPath(path, "resources", name)
	c.resourceGroups[shared.BuildMapKey(parentPath, name)] = group
}

// Assigns each collected link the abstract resource group
// shared by both of its endpoints. Links that cross groups, or whose endpoints
// are not grouped, are left ungrouped so they stay at the top level of the
// overview rather than being filed under one side of the link.
func (c *resultCollector) assignLinkGroups() {
	if len(c.resourceGroups) == 0 {
		return
	}

	for i := range c.destroyed {
		elem := &c.destroyed[i]
		if group := c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath); group != nil {
			elem.AbstractGroup = group
		}
	}

	for i := range c.failures {
		elem := &c.failures[i]
		if group := c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath); group != nil {
			elem.AbstractGroup = group
		}
	}

	for i := range c.interrupted {
		elem := &c.interrupted[i]
		if group := c.linkGroup(elem.ElementType, elem.ElementName, elem.ElementPath); group != nil {
			elem.AbstractGroup = group
		}
	}
}

// Returns the group both endpoints of a link belong to.
// Destroy elements carry the resource type in ElementType, so only the
// literal "link" marker identifies a link.
func (c *resultCollector) linkGroup(elementType, name, path string) *shared.ResourceGroup {
	if elementType != "link" {
		return nil
	}

	parentPath := shared.ParentElementPath(path, "links", name)
	groupAMapKey := shared.BuildMapKey(parentPath, extractResourceAFromLinkName(name))
	groupA := c.resourceGroups[groupAMapKey]

	groupBMapKey := shared.BuildMapKey(parentPath, extractResourceBFromLinkName(name))
	groupB := c.resourceGroups[groupBMapKey]

	if groupA == nil || groupB == nil || *groupA != *groupB {
		return nil
	}

	return groupA
}

func (c *resultCollector) collectChildResult(item *ChildDestroyItem, path string) {
	if IsFailedInstanceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "child",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedInstanceStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
		return
	}
	if IsSuccessInstanceStatus(item.Status) {
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
	}
}

func (c *resultCollector) collectLinkResult(item *LinkDestroyItem, path string) {
	if IsFailedLinkStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.LinkName,
			ElementPath:    path,
			ElementType:    "link",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedLinkStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
		return
	}
	if IsSuccessLinkStatus(item.Status) {
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
	}
}
