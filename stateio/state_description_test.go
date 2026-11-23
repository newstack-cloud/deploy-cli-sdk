package stateio

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DescribeStateTestSuite struct {
	suite.Suite
}

func (s *DescribeStateTestSuite) Test_describes_each_storage_engine() {
	cases := map[string]StateConfig{
		"memfile state in /state": {StorageEngine: StorageEngineMemfile, MemFileStateDir: "/state"},
		"memfile state in /state, encrypted at rest": {
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: "/state",
			Encryption:      StateEncryptionConfig{Enabled: true},
		},
		`postgres state in database "bluelink" on db.internal:5432`: {
			StorageEngine:    StorageEnginePostgres,
			PostgresHost:     "db.internal",
			PostgresPort:     5432,
			PostgresDatabase: "bluelink",
			PostgresUser:     "admin",
			PostgresPassword: "hunter2",
		},
		"objectstore state at s3://state-bucket/bluelink-state/": {
			StorageEngine: StorageEngineObjectstore,
			ObjectStore: ObjectStoreConfig{
				Provider: ObjectStoreProviderS3,
				Prefix:   "bluelink-state/",
				S3:       ObjectStoreS3Config{Bucket: "state-bucket", SecretAccessKey: "hunter2"},
			},
		},
		"objectstore state at gcs://state-bucket/team-a/": {
			StorageEngine: StorageEngineObjectstore,
			ObjectStore: ObjectStoreConfig{
				Provider: ObjectStoreProviderGCS,
				Prefix:   "team-a/",
				GCS:      ObjectStoreGCSConfig{Bucket: "state-bucket"},
			},
		},
		"objectstore state at azureblob://state/bluelink-state/": {
			StorageEngine: StorageEngineObjectstore,
			ObjectStore: ObjectStoreConfig{
				Provider: ObjectStoreProviderAzureBlob,
				Prefix:   "bluelink-state/",
				AzureBlob: ObjectStoreAzureBlobConfig{
					Container:  "state",
					AccountKey: "hunter2",
				},
			},
		},
	}

	for expected, config := range cases {
		description := DescribeState(&config)
		s.Equal(expected, description)
		s.NotContains(description, "hunter2", "credentials must never be shown")
	}
}

func TestDescribeStateTestSuite(t *testing.T) {
	suite.Run(t, new(DescribeStateTestSuite))
}
