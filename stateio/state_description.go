package stateio

import (
	"fmt"
	"strings"
)

// DescribeState returns a one-line description of where the engine config says state
// lives and whether it is encrypted at rest, for showing to the person running an import
// or export. It never includes credentials.
func DescribeState(config *StateConfig) string {
	switch config.StorageEngine {
	case StorageEngineMemfile, "":
		description := fmt.Sprintf("memfile state in %s", config.MemFileStateDir)
		if config.Encryption.Enabled {
			return description + ", encrypted at rest"
		}
		return description
	case StorageEnginePostgres:
		return fmt.Sprintf(
			"postgres state in database %q on %s:%d",
			config.PostgresDatabase,
			config.PostgresHost,
			config.PostgresPort,
		)
	case StorageEngineObjectstore:
		return fmt.Sprintf("objectstore state at %s", objectStoreLocation(&config.ObjectStore))
	default:
		return fmt.Sprintf("%s state", config.StorageEngine)
	}
}

func objectStoreLocation(config *ObjectStoreConfig) string {
	prefix := strings.TrimPrefix(config.Prefix, "/")
	switch config.Provider {
	case ObjectStoreProviderS3:
		return fmt.Sprintf("s3://%s/%s", config.S3.Bucket, prefix)
	case ObjectStoreProviderGCS:
		return fmt.Sprintf("gcs://%s/%s", config.GCS.Bucket, prefix)
	case ObjectStoreProviderAzureBlob:
		return fmt.Sprintf("azureblob://%s/%s", config.AzureBlob.Container, prefix)
	default:
		return fmt.Sprintf("an unknown %q provider", config.Provider)
	}
}
