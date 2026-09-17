// Package shared provides common types and utilities for deployment TUI components.
//
// This file defines the abstract resource grouping mechanism used to group
// concrete cloud resources under their abstract source types when a transformer
// plugin has expanded abstract resources.
package shared

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	// AnnotationSourceAbstractName is the annotation key set by transformer plugins
	// to record the original abstract resource name that a concrete resource was
	// expanded from.
	AnnotationSourceAbstractName = "bluelink.transform.source.abstractName"

	// AnnotationSourceAbstractType is the annotation key set by transformer plugins
	// to record the original abstract resource type that a concrete resource was
	// expanded from.
	AnnotationSourceAbstractType = "bluelink.transform.source.abstractType"

	// AnnotationResourceCategory is the annotation key set by transformer plugins
	// to classify a concrete resource as either "code-hosting" or "infrastructure".
	// Used by the code-only auto-approval mechanism.
	AnnotationResourceCategory = "bluelink.transform.resourceCategory"

	// ResourceCategoryCodeHosting indicates a resource that hosts application code
	// (e.g. Lambda function, ECS task, API Gateway).
	ResourceCategoryCodeHosting = "code-hosting"

	// ResourceCategoryInfrastructure indicates an infrastructure dependency
	// (e.g. DynamoDB table, S3 bucket, IAM role, VPC).
	ResourceCategoryInfrastructure = "infrastructure"

	// AnnotationResourceRole is the annotation key set by transformer plugins to
	// describe the part an abstract resource plays in a deployment, as opposed
	// to what it is made of, which is what AnnotationResourceCategory records.
	// Absent means "component".
	AnnotationResourceRole = "bluelink.transform.resourceRole"

	// ResourceRoleComponent indicates an abstract resource that is part of the
	// application, such as an API, a handler or a datastore.
	ResourceRoleComponent = "component"

	// ResourceRoleAmbient indicates an abstract resource that most of a
	// deployment sits inside or depends on, such as a VPC. Links out of it
	// describe the environment rather than the application, so the tree shows
	// them against the resource at the other end.
	ResourceRoleAmbient = "ambient"
)

// ResourceGroup holds the abstract resource grouping information for a
// concrete resource that was produced by a transformer plugin.
type ResourceGroup struct {
	// GroupName is the original abstract resource name (e.g. "myFunction").
	GroupName string
	// GroupType is the original abstract resource type (e.g. "celerity/function").
	GroupType string
	// Role is the part the abstract resource plays in the deployment, as the
	// transformer declared it. Empty means ResourceRoleComponent.
	Role string
}

// IsAmbient reports whether nearly everything in the deployment links out of
// this abstract resource, which decides where those links are shown.
func (g *ResourceGroup) IsAmbient() bool {
	return g != nil && g.Role == ResourceRoleAmbient
}

// ExtractGrouping reads standard Bluelink transform annotations from resource
// metadata to determine if a concrete resource was expanded from an abstract type.
// Returns nil if the resource was not produced by a transformer or if the
// required annotations are missing.
func ExtractGrouping(meta *state.ResourceMetadataState) *ResourceGroup {
	if meta == nil || meta.Annotations == nil {
		return nil
	}

	nameNode, hasName := meta.Annotations[AnnotationSourceAbstractName]
	typeNode, hasType := meta.Annotations[AnnotationSourceAbstractType]
	if !hasName || !hasType {
		return nil
	}

	name := core.StringValue(nameNode)
	typ := core.StringValue(typeNode)
	if name == "" || typ == "" {
		return nil
	}

	return &ResourceGroup{
		GroupName: name,
		GroupType: typ,
		Role:      core.StringValue(meta.Annotations[AnnotationResourceRole]),
	}
}

// ExtractGroupingFromResolved reads the same transform annotations from the
// resolved resource in a change set rather than from persisted state.
// This is the only source available for resources that are being created,
// which have no current resource state to read annotations from.
// Returns nil if the annotations are absent.
func ExtractGroupingFromResolved(resolved *provider.ResolvedResource) *ResourceGroup {
	if resolved == nil || resolved.Metadata == nil || resolved.Metadata.Annotations == nil {
		return nil
	}

	fields := resolved.Metadata.Annotations.Fields
	if fields == nil {
		return nil
	}

	name := core.StringValue(fields[AnnotationSourceAbstractName])
	typ := core.StringValue(fields[AnnotationSourceAbstractType])
	if name == "" || typ == "" {
		return nil
	}

	return &ResourceGroup{
		GroupName: name,
		GroupType: typ,
		Role:      core.StringValue(fields[AnnotationResourceRole]),
	}
}

// GroupableItem is an optional interface that splitpane.Item implementations
// can implement to indicate they belong to an abstract resource group.
// The SectionGrouper uses this to nest concrete resources under their
// abstract type parent in the navigation tree.
type GroupableItem interface {
	GetResourceGroup() *ResourceGroup
}

// LinkClassifiable is an optional interface that splitpane.Item implementations
// for links can implement to enable link classification into internal
// (within one abstract group) and cross-group categories.
type LinkClassifiable interface {
	GetLinkResourceNames() (resourceA, resourceB string)
}

// ExtractResourceCategory reads the resource category annotation from resource
// metadata. Returns an empty string if not set.
func ExtractResourceCategory(meta *state.ResourceMetadataState) string {
	if meta == nil || meta.Annotations == nil {
		return ""
	}

	node, ok := meta.Annotations[AnnotationResourceCategory]
	if !ok {
		return ""
	}

	return core.StringValue(node)
}
