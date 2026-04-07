package shared

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/stretchr/testify/suite"
)

type BlueprintCommandsSuite struct {
	suite.Suite
}

func TestBlueprintCommandsSuite(t *testing.T) {
	suite.Run(t, new(BlueprintCommandsSuite))
}

func (s *BlueprintCommandsSuite) SetupTest() {}

func (s *BlueprintCommandsSuite) Test_BlueprintSourceFromPath_https() {
	result := BlueprintSourceFromPath("https://example.com/blueprint.yaml")
	s.Equal(consts.BlueprintSourceHTTPS, result)
}

func (s *BlueprintCommandsSuite) Test_BlueprintSourceFromPath_s3() {
	result := BlueprintSourceFromPath("s3://mybucket/path/blueprint.yaml")
	s.Equal(consts.BlueprintSourceS3, result)
}

func (s *BlueprintCommandsSuite) Test_BlueprintSourceFromPath_gcs() {
	result := BlueprintSourceFromPath("gcs://mybucket/path/blueprint.yaml")
	s.Equal(consts.BlueprintSourceGCS, result)
}

func (s *BlueprintCommandsSuite) Test_BlueprintSourceFromPath_azureblob() {
	result := BlueprintSourceFromPath("azureblob://mycontainer/blueprint.yaml")
	s.Equal(consts.BlueprintSourceAzureBlob, result)
}

func (s *BlueprintCommandsSuite) Test_BlueprintSourceFromPath_local_path() {
	result := BlueprintSourceFromPath("/home/user/project/blueprint.yaml")
	s.Equal(consts.BlueprintSourceFile, result)
}

func (s *BlueprintCommandsSuite) Test_StripObjectStorageScheme_strips_s3_prefix() {
	result := StripObjectStorageScheme("s3://mybucket/path/file.yaml", "s3")
	s.Equal("mybucket/path/file.yaml", result)
}

func (s *BlueprintCommandsSuite) Test_StripObjectStorageScheme_returns_unchanged_when_no_match() {
	result := StripObjectStorageScheme("mybucket/path/file.yaml", "s3")
	s.Equal("mybucket/path/file.yaml", result)
}

func (s *BlueprintCommandsSuite) Test_BuildObjectStorageDocumentInfo_s3_path() {
	result := BuildObjectStorageDocumentInfo("s3://bucket/path/file.yaml", "s3")
	s.Equal(types.BlueprintDocumentInfo{
		FileSourceScheme: "s3",
		Directory:        "bucket/path",
		BlueprintFile:    "file.yaml",
	}, result)
}

func (s *BlueprintCommandsSuite) Test_BuildResourceActions_converts_slice_correctly() {
	externalState := &bpcore.MappingNode{
		Fields: map[string]*bpcore.MappingNode{
			"id": {Scalar: &bpcore.ScalarValue{StringValue: strPtr("res-123")}},
		},
	}
	resources := []container.ResourceReconcileResult{
		{
			ResourceID:        "res-abc",
			ChildPath:         "childA",
			RecommendedAction: container.ReconciliationActionAcceptExternal,
			ExternalState:     externalState,
			NewStatus:         1,
		},
	}

	actions := BuildResourceActions(resources)

	s.Require().Len(actions, 1)
	s.Equal("res-abc", actions[0].ResourceID)
	s.Equal("childA", actions[0].ChildPath)
	s.Equal(string(container.ReconciliationActionAcceptExternal), actions[0].Action)
	s.Equal(externalState, actions[0].ExternalState)
}

func (s *BlueprintCommandsSuite) Test_BuildResourceActions_empty_input_returns_empty_slice() {
	actions := BuildResourceActions([]container.ResourceReconcileResult{})
	s.Empty(actions)
}

func (s *BlueprintCommandsSuite) Test_BuildLinkActions_converts_correctly_with_intermediary_actions() {
	intResult := &container.IntermediaryReconcileResult{
		Name:          "int-resource",
		ExternalState: nil,
	}
	links := []container.LinkReconcileResult{
		{
			LinkID:              "link-1",
			ChildPath:           "childB",
			RecommendedAction:   container.ReconciliationActionAcceptExternal,
			NewStatus:           2,
			IntermediaryChanges: map[string]*container.IntermediaryReconcileResult{"int-1": intResult},
		},
	}

	actions := BuildLinkActions(links)

	s.Require().Len(actions, 1)
	s.Equal("link-1", actions[0].LinkID)
	s.Equal("childB", actions[0].ChildPath)
	s.Equal(string(container.ReconciliationActionAcceptExternal), actions[0].Action)
	s.Require().NotNil(actions[0].IntermediaryActions)
	s.Contains(actions[0].IntermediaryActions, "int-1")
	s.Equal(string(container.ReconciliationActionAcceptExternal), actions[0].IntermediaryActions["int-1"].Action)
}

func (s *BlueprintCommandsSuite) Test_BuildLinkActions_empty_input_returns_empty_slice() {
	actions := BuildLinkActions([]container.LinkReconcileResult{})
	s.Empty(actions)
}

func (s *BlueprintCommandsSuite) Test_BuildIntermediaryActions_nil_input_returns_nil() {
	result := BuildIntermediaryActions(nil)
	s.Nil(result)
}

func (s *BlueprintCommandsSuite) Test_BuildIntermediaryActions_empty_map_returns_nil() {
	result := BuildIntermediaryActions(map[string]*container.IntermediaryReconcileResult{})
	s.Nil(result)
}

func (s *BlueprintCommandsSuite) Test_BuildIntermediaryActions_converts_entries_with_accept_external_action() {
	externalState := &bpcore.MappingNode{
		Fields: map[string]*bpcore.MappingNode{
			"key": {Scalar: &bpcore.ScalarValue{StringValue: strPtr("value")}},
		},
	}
	changes := map[string]*container.IntermediaryReconcileResult{
		"int-resource": {
			Name:          "int-resource",
			ExternalState: externalState,
		},
	}

	result := BuildIntermediaryActions(changes)

	s.Require().NotNil(result)
	s.Require().Contains(result, "int-resource")
	s.Equal(string(container.ReconciliationActionAcceptExternal), result["int-resource"].Action)
	s.Equal(externalState, result["int-resource"].ExternalState)
}

func (s *BlueprintCommandsSuite) Test_BuildDocumentInfo_file_source_resolves_to_absolute_path() {
	info, err := BuildDocumentInfo(consts.BlueprintSourceFile, "blueprint.yaml")
	s.NoError(err)
	s.Equal("file", info.FileSourceScheme)
	s.NotEmpty(info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_BuildDocumentInfo_https_source() {
	info, err := BuildDocumentInfo(consts.BlueprintSourceHTTPS, "https://example.com/repo/blueprint.yaml")
	s.NoError(err)
	s.Equal("https", info.FileSourceScheme)
	s.Equal("/repo", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_BuildDocumentInfo_s3_source() {
	info, err := BuildDocumentInfo(consts.BlueprintSourceS3, "s3://mybucket/path/blueprint.yaml")
	s.NoError(err)
	s.Equal("s3", info.FileSourceScheme)
	s.Equal("mybucket/path", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_BuildDocumentInfo_gcs_source() {
	info, err := BuildDocumentInfo(consts.BlueprintSourceGCS, "gcs://mybucket/path/blueprint.yaml")
	s.NoError(err)
	s.Equal("gcs", info.FileSourceScheme)
	s.Equal("mybucket/path", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_BuildDocumentInfo_azureblob_source() {
	info, err := BuildDocumentInfo(consts.BlueprintSourceAzureBlob, "azureblob://mycontainer/path/blueprint.yaml")
	s.NoError(err)
	s.Equal("azureblob", info.FileSourceScheme)
	s.Equal("mycontainer/path", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_BuildHTTPSDocumentInfo_with_subdirectory_path() {
	info, err := BuildHTTPSDocumentInfo("https://example.com/repo/blueprint.yaml")
	s.NoError(err)
	s.Equal("https", info.FileSourceScheme)
	s.Equal("/repo", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
	s.Equal(map[string]any{"host": "example.com"}, info.BlueprintLocationMetadata)
}

func (s *BlueprintCommandsSuite) Test_BuildHTTPSDocumentInfo_root_path_yields_empty_directory() {
	info, err := BuildHTTPSDocumentInfo("https://example.com/blueprint.yaml")
	s.NoError(err)
	s.Equal("https", info.FileSourceScheme)
	s.Equal("", info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
	s.Equal(map[string]any{"host": "example.com"}, info.BlueprintLocationMetadata)
}

func (s *BlueprintCommandsSuite) Test_BuildLocalFileDocumentInfo_relative_path_resolves_to_absolute() {
	info, err := BuildLocalFileDocumentInfo("blueprint.yaml")
	s.NoError(err)
	s.Equal("file", info.FileSourceScheme)
	s.NotEmpty(info.Directory)
	s.Equal("blueprint.yaml", info.BlueprintFile)
}

func (s *BlueprintCommandsSuite) Test_GetEffectiveInstanceID_returns_instance_id_when_non_empty() {
	result := GetEffectiveInstanceID("id-123", "my-instance")
	s.Equal("id-123", result)
}

func (s *BlueprintCommandsSuite) Test_GetEffectiveInstanceID_returns_instance_name_as_fallback() {
	result := GetEffectiveInstanceID("", "my-instance")
	s.Equal("my-instance", result)
}

func strPtr(s string) *string {
	return &s
}
