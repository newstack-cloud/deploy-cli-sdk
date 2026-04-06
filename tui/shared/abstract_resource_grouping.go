// Package shared provides common types and utilities for deployment TUI components.
//
// This file defines the abstract resource grouping mechanism used to group
// concrete cloud resources under their abstract source types when a transformer
// plugin has expanded abstract resources.
package shared

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
)

// ResourceGroup holds the abstract resource grouping information for a
// concrete resource that was produced by a transformer plugin.
type ResourceGroup struct {
	// GroupName is the original abstract resource name (e.g. "myFunction").
	GroupName string
	// GroupType is the original abstract resource type (e.g. "celerity/function").
	GroupType string
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
