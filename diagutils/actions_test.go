package diagutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/stretchr/testify/suite"
)

type ActionsTestSuite struct {
	suite.Suite
}

func TestActionsTestSuite(t *testing.T) {
	suite.Run(t, new(ActionsTestSuite))
}

func (s *ActionsTestSuite) Test_returns_nil_for_unsupported_action() {
	action := errors.SuggestedAction{Type: "unknown_action"}
	result := GetConcreteAction(action, nil)
	s.Nil(result)
}

func (s *ActionsTestSuite) Test_install_provider_without_namespace_returns_explore_links() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeInstallProvider)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Empty(result.Commands)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_install_provider_with_known_namespace_returns_install_command() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeInstallProvider)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins install")
	s.Contains(result.Commands[0], "aws")
}

func (s *ActionsTestSuite) Test_install_provider_with_unknown_namespace_returns_placeholder_org() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeInstallProvider)}
	metadata := map[string]any{"providerNamespace": "custom-provider"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins install <organisation>/custom-provider")
}

func (s *ActionsTestSuite) Test_update_provider_without_namespace_returns_links() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeUpdateProvider)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Empty(result.Commands)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Check for new versions")
}

func (s *ActionsTestSuite) Test_update_provider_with_known_namespace_returns_update_command() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeUpdateProvider)}
	metadata := map[string]any{"providerNamespace": "azure"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "bluelink plugins update")
	s.Contains(result.Commands[0], "azure")
}

func (s *ActionsTestSuite) Test_update_provider_with_unknown_namespace_returns_placeholder_org() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeUpdateProvider)}
	metadata := map[string]any{"providerNamespace": "my-custom-provider"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Commands, 1)
	s.Contains(result.Commands[0], "<organisation>/my-custom-provider")
}

func (s *ActionsTestSuite) Test_check_function_name_returns_provider_registry_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckFunctionName)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore provider functions in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/providers", result.Links[0].URL)
}

func (s *ActionsTestSuite) Test_check_resource_type_without_namespace_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceType)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_check_resource_type_with_namespace_returns_provider_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceType)}
	metadata := map[string]any{"providerNamespace": "gcloud"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "resource types")
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/providers")
	s.Contains(result.Links[0].URL, "gcloud")
}

func (s *ActionsTestSuite) Test_check_resource_type_schema_without_type_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceTypeSchema)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_check_resource_type_schema_returns_resource_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceTypeSchema)}
	metadata := map[string]any{"resourceType": "aws/s3/bucket"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "aws/s3/bucket")
	s.Contains(result.Links[0].URL, "resources/aws--s3--bucket")
}

func (s *ActionsTestSuite) Test_check_resource_type_schema_encodes_colons_in_url() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckResourceTypeSchema)}
	metadata := map[string]any{"resourceType": "kubernetes::core/v1/pod"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Contains(result.Links[0].URL, "kubernetes--core--v1--pod")
}

func (s *ActionsTestSuite) Test_check_abstract_resource_type_without_namespace_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckAbstractResourceType)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore transformers")
	s.Contains(result.Links[0].URL, "registry.bluelink.dev/transformers")
}

func (s *ActionsTestSuite) Test_check_abstract_resource_type_with_namespace_returns_transformer_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckAbstractResourceType)}
	metadata := map[string]any{"transformerNamespace": "celerity"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "abstract resource types")
	s.Contains(result.Links[0].URL, "transformers")
	s.Contains(result.Links[0].URL, "celerity")
}

func (s *ActionsTestSuite) Test_check_data_source_type_without_namespace_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckDataSourceType)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_check_data_source_type_with_namespace_returns_provider_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckDataSourceType)}
	metadata := map[string]any{"providerNamespace": "kubernetes"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "data source types")
	s.Contains(result.Links[0].URL, "providers")
	s.Contains(result.Links[0].URL, "kubernetes")
}

func (s *ActionsTestSuite) Test_check_variable_type_without_namespace_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckVariableType)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_check_variable_type_with_namespace_returns_provider_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckVariableType)}
	metadata := map[string]any{"providerNamespace": "aws"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "custom variable types")
}

func (s *ActionsTestSuite) Test_check_custom_variable_options_without_namespace_returns_explore_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckCustomVariableOptions)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "Explore providers")
}

func (s *ActionsTestSuite) Test_check_custom_variable_options_with_namespace_returns_provider_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckCustomVariableOptions)}
	metadata := map[string]any{"providerNamespace": "azure"}
	result := GetConcreteAction(action, metadata)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Contains(result.Links[0].Title, "custom variable options")
	s.Contains(result.Links[0].URL, "providers")
	s.Contains(result.Links[0].URL, "azure")
}

func (s *ActionsTestSuite) Test_check_transformers_returns_registry_link() {
	action := errors.SuggestedAction{Type: string(errors.ActionTypeCheckTransformers)}
	result := GetConcreteAction(action, nil)
	s.NotNil(result)
	s.Len(result.Links, 1)
	s.Equal("Explore transformers in the official registry", result.Links[0].Title)
	s.Equal("https://registry.bluelink.dev/transformers", result.Links[0].URL)
}
