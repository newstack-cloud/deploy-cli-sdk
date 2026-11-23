package stateio

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	azurestore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/azureblob"
	gcsstore "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/gcs"
	s3store "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/s3"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// The object storage services the objectstore storage engine can keep state in, as set
// in ObjectStoreConfig.Provider.
const (
	ObjectStoreProviderS3        = "s3"
	ObjectStoreProviderGCS       = "gcs"
	ObjectStoreProviderAzureBlob = "azureblob"
)

// The service set up here mirrors the deploy engine's, so the CLI reads and writes the
// same bucket, under the same prefix, with the same credentials resolution.
func openObjectStoreBackend(
	ctx context.Context,
	config *StateConfig,
	logger core.Logger,
) (*stateBackend, error) {
	svc, closeService, err := buildObjectStoreService(ctx, &config.ObjectStore)
	if err != nil {
		return nil, &StateAccessError{
			Code:    ErrCodeStateConfigInvalid,
			Message: "the objectstore storage could not be set up",
			Hint:    stateConfigHint("objectstore"),
			Err:     err,
		}
	}

	// The container is only opened for a single state import or export, which loads
	// instances fresh and exits long before a cached instance would be old enough to
	// revalidate, so revalidation would only add HEAD requests.
	container, err := objectstore.LoadStateContainer(
		ctx, svc, config.ObjectStore.Prefix, logger, objectstore.WithoutCacheRevalidation(),
	)
	if err != nil {
		closeService()
		return nil, &StateAccessError{
			Code:    ErrCodeStateUnavailable,
			Message: "the objectstore state could not be loaded",
			Err:     err,
		}
	}

	return &stateBackend{
		container:    container,
		close:        closeService,
		discardSetup: func() {},
	}, nil
}

func buildObjectStoreService(
	ctx context.Context,
	config *ObjectStoreConfig,
) (objectstore.Service, func(), error) {
	switch config.Provider {
	case ObjectStoreProviderS3:
		svc, err := buildS3ObjectStoreService(ctx, &config.S3)
		return svc, func() {}, err
	case ObjectStoreProviderGCS:
		return buildGCSObjectStoreService(ctx, &config.GCS)
	case ObjectStoreProviderAzureBlob:
		svc, err := buildAzureBlobObjectStoreService(&config.AzureBlob)
		return svc, func() {}, err
	default:
		return nil, nil, fmt.Errorf(
			"unsupported objectstore provider %q, supported providers are: \"s3\", \"gcs\", \"azureblob\"",
			config.Provider,
		)
	}
}

func buildS3ObjectStoreService(ctx context.Context, config *ObjectStoreS3Config) (objectstore.Service, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("state.objectstore.s3.bucket is required")
	}

	conf, err := loadAWSConfig(ctx, config.Region, config.AccessKeyID, config.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3store.NewClient(conf, s3store.ClientOptions{
		Endpoint:     config.Endpoint,
		UsePathStyle: config.UsePathStyle,
	})
	return s3store.NewService(client, config.Bucket), nil
}

// Unlike the other stores, the GCS client holds connections that are released on close.
func buildGCSObjectStoreService(
	ctx context.Context,
	config *ObjectStoreGCSConfig,
) (objectstore.Service, func(), error) {
	if config.Bucket == "" {
		return nil, nil, fmt.Errorf("state.objectstore.gcs.bucket is required")
	}

	client, err := gcsstore.NewClient(ctx, gcsstore.ClientOptions{
		Endpoint:              config.Endpoint,
		WithoutAuthentication: config.WithoutAuthentication,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gcs client: %w", err)
	}

	return gcsstore.NewService(client, config.Bucket), func() { _ = client.Close() }, nil
}

func buildAzureBlobObjectStoreService(config *ObjectStoreAzureBlobConfig) (objectstore.Service, error) {
	if config.Container == "" {
		return nil, fmt.Errorf("state.objectstore.azureblob.container is required")
	}
	if config.ServiceURL == "" || config.AccountName == "" || config.AccountKey == "" {
		return nil, fmt.Errorf(
			"state.objectstore.azureblob.service_url, account_name and account_key are required",
		)
	}

	client, err := azurestore.NewClient(azurestore.ClientOptions{
		ServiceURL:  config.ServiceURL,
		AccountName: config.AccountName,
		AccountKey:  config.AccountKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create azure blob client: %w", err)
	}

	return azurestore.NewService(client, config.Container), nil
}
