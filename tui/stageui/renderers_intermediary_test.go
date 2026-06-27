package stageui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func leafPath(id, leaf string) string {
	return "[\"intermediaries\"][\"" + id + "\"][\"" + leaf + "\"]"
}

func TestSplitIntermediaryChanges_create(t *testing.T) {
	id := "ruleA__fnB__eventbridge-invoke-permission"
	linkChanges := &provider.LinkChanges{
		NewFields: []*provider.FieldChange{
			{FieldPath: "[\"resourceA\"].someField", NewValue: core.MappingNodeFromString("v")},
			{FieldPath: leafPath(id, "resourceType"), NewValue: core.MappingNodeFromString("aws/lambda/permission")},
			{FieldPath: leafPath(id, "sourceArn"), NewValue: core.MappingNodeFromString("arn:rule")},
		},
	}

	regular, groups := SplitIntermediaryChanges(linkChanges)

	// The non-intermediary field stays in regular changes.
	require.Len(t, regular.NewFields, 1)
	assert.Equal(t, "[\"resourceA\"].someField", regular.NewFields[0].FieldPath)

	require.Len(t, groups, 1)
	assert.Equal(t, id, groups[0].id)
	assert.Equal(t, "aws/lambda/permission", groups[0].resourceType)
	assert.True(t, groups[0].created)
	assert.False(t, groups[0].destroyed)
	require.Len(t, groups[0].leaves, 1)
	assert.Equal(t, "sourceArn", groups[0].leaves[0].name)
	assert.Equal(t, leafNew, groups[0].leaves[0].kind)
	assert.Equal(t, "\"arn:rule\"", groups[0].leaves[0].newValue)
}

func TestSplitIntermediaryChanges_updateAndKnownOnDeploy(t *testing.T) {
	id := "ruleA__queueB__eventbridge-send-policy"
	linkChanges := &provider.LinkChanges{
		ModifiedFields: []*provider.FieldChange{
			{
				FieldPath: leafPath(id, "sourceArn"),
				PrevValue: core.MappingNodeFromString("arn:old"),
				NewValue:  core.MappingNodeFromString("arn:new"),
			},
		},
		FieldChangesKnownOnDeploy: []string{leafPath(id, "queueArn")},
	}

	_, groups := SplitIntermediaryChanges(linkChanges)

	require.Len(t, groups, 1)
	g := groups[0]
	assert.False(t, g.created)
	assert.False(t, g.destroyed)
	// resourceType unchanged -> not captured, leaves sorted by name (queueArn, sourceArn).
	require.Len(t, g.leaves, 2)
	assert.Equal(t, "queueArn", g.leaves[0].name)
	assert.Equal(t, leafKnownOnDeploy, g.leaves[0].kind)
	assert.Equal(t, "sourceArn", g.leaves[1].name)
	assert.Equal(t, leafModified, g.leaves[1].kind)
	assert.Equal(t, "\"arn:old\"", g.leaves[1].prevValue)
	assert.Equal(t, "\"arn:new\"", g.leaves[1].newValue)
}

func TestSplitIntermediaryChanges_destroy(t *testing.T) {
	id := "ruleA__fnB__eventbridge-invoke-permission"
	linkChanges := &provider.LinkChanges{
		RemovedFields: []string{
			leafPath(id, "resourceType"),
			leafPath(id, "sourceArn"),
		},
	}

	_, groups := SplitIntermediaryChanges(linkChanges)

	require.Len(t, groups, 1)
	assert.True(t, groups[0].destroyed)
	require.Len(t, groups[0].leaves, 1)
	assert.Equal(t, "sourceArn", groups[0].leaves[0].name)
	assert.Equal(t, leafRemoved, groups[0].leaves[0].kind)
}

func TestSplitIntermediaryChanges_noIntermediaries(t *testing.T) {
	linkChanges := &provider.LinkChanges{
		NewFields: []*provider.FieldChange{
			{FieldPath: "[\"resourceA\"].field", NewValue: core.MappingNodeFromString("v")},
		},
	}

	regular, groups := SplitIntermediaryChanges(linkChanges)
	assert.Empty(t, groups)
	assert.Len(t, regular.NewFields, 1)
}
