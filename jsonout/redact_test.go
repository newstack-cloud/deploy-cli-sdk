package jsonout

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/stretchr/testify/suite"
)

type RedactSuite struct {
	suite.Suite
}

const theSecret = "super-secret-token"

func changeSetWithASecret() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"configStore": {
				NewFields: []provider.FieldChange{
					{
						FieldPath: "spec.secureValues.apiToken",
						NewValue:  core.MappingNodeFromString(theSecret),
						Sensitive: true,
					},
					{
						FieldPath: "spec.values.logLevel",
						NewValue:  core.MappingNodeFromString("info"),
					},
				},
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceName: "configStore",
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Spec: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"secureValues": {
									Fields: map[string]*core.MappingNode{
										"apiToken": core.MappingNodeFromString(theSecret),
									},
								},
								"values": {
									Fields: map[string]*core.MappingNode{
										"logLevel": core.MappingNodeFromString("info"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *RedactSuite) Test_a_sensitive_field_change_value_is_replaced() {
	redacted := RedactChanges(changeSetWithASecret())

	fields := redacted.NewResources["configStore"].NewFields
	s.Require().Len(fields, 2)
	s.Equal(headless.RedactedValue, core.StringValue(fields[0].NewValue))
	s.Equal("info", core.StringValue(fields[1].NewValue), "a field that is not sensitive is untouched")
}

// The resolved spec the deploy phase applies holds the same values the field
// changes describe, so redacting only the changes leaves the secret readable a
// few lines further down the same document.
func (s *RedactSuite) Test_the_secret_is_gone_from_the_whole_document() {
	redacted := RedactChanges(changeSetWithASecret())

	encoded, err := json.Marshal(redacted)
	s.Require().NoError(err)
	s.NotContains(string(encoded), theSecret)
	s.Contains(string(encoded), "info", "non-sensitive values still have to be readable")
}

// The terminal renderer works from the same change set, so redaction must not
// reach back into it.
func (s *RedactSuite) Test_the_original_change_set_is_left_intact() {
	original := changeSetWithASecret()

	RedactChanges(original)

	s.Equal(
		theSecret,
		core.StringValue(original.NewResources["configStore"].NewFields[0].NewValue),
	)
	resolved := original.NewResources["configStore"].AppliedResourceInfo.ResourceWithResolvedSubs
	s.Equal(theSecret, core.StringValue(resolved.Spec.Fields["secureValues"].Fields["apiToken"]))
}

// A dotted field path cannot address a key that itself contains a dot, so the
// nearest addressable ancestor goes instead of leaving the value exposed.
func (s *RedactSuite) Test_an_unaddressable_key_redacts_its_ancestor() {
	changeSet := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"configStore": {
				NewFields: []provider.FieldChange{
					{
						FieldPath: "spec.secureValues.my.dotted.key",
						NewValue:  core.MappingNodeFromString(theSecret),
						Sensitive: true,
					},
				},
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Spec: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"secureValues": {
									Fields: map[string]*core.MappingNode{
										"my.dotted.key": core.MappingNodeFromString(theSecret),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	encoded, err := json.Marshal(RedactChanges(changeSet))
	s.Require().NoError(err)
	s.False(strings.Contains(string(encoded), theSecret), "the secret must not survive in any form")
}

func TestRedactSuite(t *testing.T) {
	suite.Run(t, new(RedactSuite))
}
