package stateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3store "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/s3"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	azuriteAccountName    = "devstoreaccount1"
	azuriteAccountKey     = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
	azuriteServiceURL     = "http://127.0.0.1:10003/devstoreaccount1"
	azuriteStateContainer = "objectstore-state"
)

// Each test works under its own prefix in the shared emulator bucket, so tests and
// repeated runs cannot see each other's state.
func uniqueObjectStorePrefix() string {
	return fmt.Sprintf("stateio-tests/%d/", time.Now().UnixNano())
}

func s3ObjectStoreConfig(prefix string) *EngineConfig {
	return &EngineConfig{State: StateConfig{
		StorageEngine: StorageEngineObjectstore,
		ObjectStore: ObjectStoreConfig{
			Provider: ObjectStoreProviderS3,
			Prefix:   prefix,
			S3: ObjectStoreS3Config{
				Bucket:          "test-bucket",
				Region:          "us-east-1",
				Endpoint:        "http://localhost:4580",
				UsePathStyle:    true,
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		},
	}}
}

// A parent with a resource, a link and a nested child, which between them produce every
// kind of key the objectstore backend writes for an instance.
func objectStoreTestInstances() []state.InstanceState {
	child := state.InstanceState{
		InstanceID:   "os-child-1",
		InstanceName: "orders-networking",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"os-res-2": {ResourceID: "os-res-2", Name: "vpc", Type: "aws/ec2/vpc", InstanceID: "os-child-1"},
		},
		ResourceIDs: map[string]string{"vpc": "os-res-2"},
	}
	return []state.InstanceState{{
		InstanceID:   "os-parent-1",
		InstanceName: "orders",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"os-res-1": {
				ResourceID: "os-res-1", Name: "ordersTable", Type: "aws/dynamodb/table", InstanceID: "os-parent-1",
			},
		},
		ResourceIDs: map[string]string{"ordersTable": "os-res-1"},
		Links: map[string]*state.LinkState{
			"ordersFunction::ordersTable": {
				LinkID: "os-link-1", Name: "ordersFunction::ordersTable", InstanceID: "os-parent-1",
			},
		},
		ChildBlueprints: map[string]*state.InstanceState{"networking": &child},
	}}
}

type ObjectStoreS3IntegrationSuite struct {
	suite.Suite
	prefix string
	config *EngineConfig
	fs     afero.Fs
}

func TestObjectStoreS3IntegrationSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreS3IntegrationSuite))
}

func (s *ObjectStoreS3IntegrationSuite) SetupTest() {
	s.prefix = uniqueObjectStorePrefix()
	s.config = s3ObjectStoreConfig(s.prefix)
	s.fs = afero.NewMemMapFs()
}

func (s *ObjectStoreS3IntegrationSuite) importInstances(config *EngineConfig, instances []state.InstanceState) error {
	data, err := json.Marshal(instances)
	s.Require().NoError(err)
	_, err = Import(ImportParams{EngineConfig: config, FileSystem: s.fs, FileData: data})
	return err
}

func (s *ObjectStoreS3IntegrationSuite) exportInstances(config *EngineConfig, filters ...string) []state.InstanceState {
	_, err := Export(ExportParams{
		FilePath:        "/export.json",
		EngineConfig:    config,
		FileSystem:      s.fs,
		InstanceFilters: filters,
	})
	s.Require().NoError(err)

	data, err := afero.ReadFile(s.fs, "/export.json")
	s.Require().NoError(err)
	var instances []state.InstanceState
	s.Require().NoError(json.Unmarshal(data, &instances))
	return instances
}

func (s *ObjectStoreS3IntegrationSuite) bucketKeys() []string {
	s3Config := s.config.State.ObjectStore.S3
	client := s3store.NewClient(aws.Config{
		Region:      s3Config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, ""),
	}, s3store.ClientOptions{Endpoint: s3Config.Endpoint, UsePathStyle: s3Config.UsePathStyle})

	objects, err := s3store.NewService(client, s3Config.Bucket).List(context.Background(), s.prefix)
	s.Require().NoError(err)
	keys := make([]string, len(objects))
	for i, object := range objects {
		keys[i] = object.Key
	}
	return keys
}

func findInstance(instances []state.InstanceState, id string) *state.InstanceState {
	for i := range instances {
		if instances[i].InstanceID == id {
			return &instances[i]
		}
	}
	return nil
}

func (s *ObjectStoreS3IntegrationSuite) Test_writes_the_keys_lazy_lookups_depend_on() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))

	keys := statestore.NewKeyBuilder(s.prefix)
	s.ElementsMatch([]string{
		keys.Instance("os-parent-1"),
		keys.Instance("os-child-1"),
		keys.InstanceByName("orders"),
		keys.InstanceByName("orders-networking"),
		keys.ResourceRecord("os-res-1"),
		keys.ResourceRecord("os-res-2"),
		keys.LinkRecord("os-link-1"),
	}, s.bucketKeys())
}

func (s *ObjectStoreS3IntegrationSuite) Test_exports_every_instance_from_a_fresh_container_with_children_nested() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))

	exported := s.exportInstances(s.config)

	s.Len(exported, 1)
	parent := findInstance(exported, "os-parent-1")
	s.Require().NotNil(parent)
	s.Require().Contains(parent.ChildBlueprints, "networking")
	s.Equal("os-child-1", parent.ChildBlueprints["networking"].InstanceID)
	s.Contains(parent.ChildBlueprints["networking"].Resources, "os-res-2")
	s.Contains(parent.Resources, "os-res-1")
	s.Contains(parent.Links, "ordersFunction::ordersTable")
}

func (s *ObjectStoreS3IntegrationSuite) Test_exports_filtered_instances_by_id_and_name() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))

	exported := s.exportInstances(s.config, "orders", "os-child-1")

	// The requested child is exported only under its requested parent.
	s.Require().Len(exported, 1)
	parent := findInstance(exported, "os-parent-1")
	s.Require().NotNil(parent)
	s.Require().Contains(parent.ChildBlueprints, "networking")
	s.Equal("os-child-1", parent.ChildBlueprints["networking"].InstanceID)
}

func (s *ObjectStoreS3IntegrationSuite) Test_exports_a_child_requested_without_its_parent_at_the_top_level() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))

	exported := s.exportInstances(s.config, "os-child-1")

	s.Require().Len(exported, 1)
	s.NotNil(findInstance(exported, "os-child-1"))
}

func (s *ObjectStoreS3IntegrationSuite) Test_round_trips_memfile_to_objectstore_to_memfile() {
	memfileSource := memfileEngineConfig(s.T(), s.fs, "/source/state")
	s.Require().NoError(s.importInstances(memfileSource, objectStoreTestInstances()))

	s.Require().NoError(s.importInstances(s.config, s.exportInstances(memfileSource)))
	memfileTarget := memfileEngineConfig(s.T(), s.fs, "/target/state")
	s.Require().NoError(s.importInstances(memfileTarget, s.exportInstances(s.config)))

	roundTripped := s.exportInstances(memfileTarget)
	s.Len(roundTripped, 1)
	parent := findInstance(roundTripped, "os-parent-1")
	s.Require().NotNil(parent)
	s.Equal("orders", parent.InstanceName)
	s.Contains(parent.ChildBlueprints, "networking")
}

func (s *ObjectStoreS3IntegrationSuite) Test_re_importing_the_same_state_succeeds() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))

	s.Require().NoError(s.importInstances(s.config, s.exportInstances(s.config)))

	parent := findInstance(s.exportInstances(s.config), "os-parent-1")
	s.Require().NotNil(parent)
	s.Equal(int64(1), parent.Version)
}

func (s *ObjectStoreS3IntegrationSuite) Test_rejects_a_rename_and_a_name_conflict_before_writing() {
	s.Require().NoError(s.importInstances(s.config, objectStoreTestInstances()))
	keysBefore := s.bucketKeys()

	err := s.importInstances(s.config, []state.InstanceState{
		{InstanceID: "os-parent-1", InstanceName: "orders-v2", Status: core.InstanceStatusDeployed},
		{InstanceID: "os-other-1", InstanceName: "orders-networking", Status: core.InstanceStatusDeployed},
	})

	s.Require().Error(err)
	var importErr *ImportError
	s.Require().True(errors.As(err, &importErr), "expected an ImportError, got %v", err)
	s.Equal(ErrCodeInstanceNameConflict, importErr.Code)
	s.Equal(
		"nothing was imported\ninstance names already in use:\n"+
			`  - "orders-networking" is already used by instance os-child-1, not os-other-1`+"\n"+
			"instances cannot be renamed:\n"+
			`  - os-parent-1 is named "orders", not "orders-v2"`,
		importErr.Message,
	)
	s.ElementsMatch(keysBefore, s.bucketKeys())
}

func objectStoreRoundTrip(s *suite.Suite, config *EngineConfig) {
	fs := afero.NewMemMapFs()
	data, err := json.Marshal(objectStoreTestInstances())
	s.Require().NoError(err)
	_, err = Import(ImportParams{EngineConfig: config, FileSystem: fs, FileData: data})
	s.Require().NoError(err)

	result, err := Export(ExportParams{FilePath: "/export.json", EngineConfig: config, FileSystem: fs})
	s.Require().NoError(err)
	s.Equal(2, result.InstancesCount)
}

type ObjectStoreGCSIntegrationSuite struct {
	suite.Suite
}

func TestObjectStoreGCSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreGCSIntegrationSuite))
}

func (s *ObjectStoreGCSIntegrationSuite) Test_round_trips_state_through_gcs() {
	objectStoreRoundTrip(&s.Suite, &EngineConfig{State: StateConfig{
		StorageEngine: StorageEngineObjectstore,
		ObjectStore: ObjectStoreConfig{
			Provider: ObjectStoreProviderGCS,
			Prefix:   uniqueObjectStorePrefix(),
			GCS: ObjectStoreGCSConfig{
				Bucket:                "test-bucket",
				Endpoint:              "http://localhost:8185/storage/v1/",
				WithoutAuthentication: true,
			},
		},
	}})
}

type ObjectStoreAzureBlobIntegrationSuite struct {
	suite.Suite
}

func TestObjectStoreAzureBlobIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreAzureBlobIntegrationSuite))
}

func (s *ObjectStoreAzureBlobIntegrationSuite) SetupSuite() {
	cred, err := azblob.NewSharedKeyCredential(azuriteAccountName, azuriteAccountKey)
	s.Require().NoError(err)
	client, err := azblob.NewClientWithSharedKeyCredential(azuriteServiceURL, cred, nil)
	s.Require().NoError(err)
	// The container may be left over from an earlier run.
	_, _ = client.CreateContainer(context.Background(), azuriteStateContainer, nil)
}

func (s *ObjectStoreAzureBlobIntegrationSuite) Test_round_trips_state_through_azure_blob() {
	objectStoreRoundTrip(&s.Suite, &EngineConfig{State: StateConfig{
		StorageEngine: StorageEngineObjectstore,
		ObjectStore: ObjectStoreConfig{
			Provider: ObjectStoreProviderAzureBlob,
			Prefix:   uniqueObjectStorePrefix(),
			AzureBlob: ObjectStoreAzureBlobConfig{
				ServiceURL:  azuriteServiceURL,
				AccountName: azuriteAccountName,
				AccountKey:  azuriteAccountKey,
				Container:   azuriteStateContainer,
			},
		},
	}})
}
