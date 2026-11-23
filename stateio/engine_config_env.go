package stateio

import (
	"fmt"
	"strconv"
	"strings"
)

const engineStateEnvPrefix = "BLUELINK_DEPLOY_ENGINE_STATE_"

type stateEnvBinding struct {
	// The key under "state" as the deploy engine binds it, e.g. "encryption.awskms.key_id".
	key   string
	apply func(config *StateConfig, value string) error
}

// Mirrors the "state.*" environment variables the deploy engine binds, minus the
// settings that only affect a running engine (event thresholds, file size guides, object
// store cache revalidation).
// Previous encryption keys have no environment binding in the engine either.
var stateEnvBindings = []stateEnvBinding{
	stringEnv("storage_engine", func(c *StateConfig) *string { return &c.StorageEngine }),
	stringEnv("memfile_state_dir", func(c *StateConfig) *string { return &c.MemFileStateDir }),
	stringEnv("postgres_user", func(c *StateConfig) *string { return &c.PostgresUser }),
	stringEnv("postgres_password", func(c *StateConfig) *string { return &c.PostgresPassword }),
	stringEnv("postgres_host", func(c *StateConfig) *string { return &c.PostgresHost }),
	intEnv("postgres_port", func(c *StateConfig) *int { return &c.PostgresPort }),
	stringEnv("postgres_database", func(c *StateConfig) *string { return &c.PostgresDatabase }),
	stringEnv("postgres_ssl_mode", func(c *StateConfig) *string { return &c.PostgresSSLMode }),
	intEnv("postgres_pool_max_conns", func(c *StateConfig) *int { return &c.PostgresPoolMaxConns }),
	stringEnv("postgres_pool_max_conn_lifetime", func(c *StateConfig) *string {
		return &c.PostgresPoolMaxConnLifetime
	}),

	boolEnv("encryption.enabled", func(c *StateConfig) *bool { return &c.Encryption.Enabled }),
	stringEnv("encryption.provider", func(c *StateConfig) *string { return &c.Encryption.Provider }),
	stringEnv("encryption.passphrase", func(c *StateConfig) *string { return &c.Encryption.Passphrase }),
	stringEnv("encryption.passphrase_file", func(c *StateConfig) *string { return &c.Encryption.PassphraseFile }),
	stringEnv("encryption.local_key_file", func(c *StateConfig) *string { return &c.Encryption.LocalKeyFile }),
	stringEnv("encryption.awskms.key_id", func(c *StateConfig) *string { return &c.Encryption.AWSKMS.KeyID }),
	stringEnv("encryption.awskms.region", func(c *StateConfig) *string { return &c.Encryption.AWSKMS.Region }),
	stringEnv("encryption.awskms.endpoint", func(c *StateConfig) *string { return &c.Encryption.AWSKMS.Endpoint }),
	stringEnv("encryption.awskms.access_key_id", func(c *StateConfig) *string {
		return &c.Encryption.AWSKMS.AccessKeyID
	}),
	stringEnv("encryption.awskms.secret_access_key", func(c *StateConfig) *string {
		return &c.Encryption.AWSKMS.SecretAccessKey
	}),
	stringEnv("encryption.gcpkms.key_name", func(c *StateConfig) *string { return &c.Encryption.GCPKMS.KeyName }),
	stringEnv("encryption.gcpkms.credentials_file", func(c *StateConfig) *string {
		return &c.Encryption.GCPKMS.CredentialsFile
	}),
	stringEnv("encryption.azurekeyvault.vault_url", func(c *StateConfig) *string {
		return &c.Encryption.AzureKeyVault.VaultURL
	}),
	stringEnv("encryption.azurekeyvault.key_name", func(c *StateConfig) *string {
		return &c.Encryption.AzureKeyVault.KeyName
	}),
	stringEnv("encryption.azurekeyvault.key_version", func(c *StateConfig) *string {
		return &c.Encryption.AzureKeyVault.KeyVersion
	}),
	stringEnv("encryption.azurekeyvault.algorithm", func(c *StateConfig) *string {
		return &c.Encryption.AzureKeyVault.Algorithm
	}),

	stringEnv("objectstore.provider", func(c *StateConfig) *string { return &c.ObjectStore.Provider }),
	stringEnv("objectstore.prefix", func(c *StateConfig) *string { return &c.ObjectStore.Prefix }),
	stringEnv("objectstore.s3.bucket", func(c *StateConfig) *string { return &c.ObjectStore.S3.Bucket }),
	stringEnv("objectstore.s3.region", func(c *StateConfig) *string { return &c.ObjectStore.S3.Region }),
	stringEnv("objectstore.s3.endpoint", func(c *StateConfig) *string { return &c.ObjectStore.S3.Endpoint }),
	boolEnv("objectstore.s3.use_path_style", func(c *StateConfig) *bool { return &c.ObjectStore.S3.UsePathStyle }),
	stringEnv("objectstore.s3.access_key_id", func(c *StateConfig) *string {
		return &c.ObjectStore.S3.AccessKeyID
	}),
	stringEnv("objectstore.s3.secret_access_key", func(c *StateConfig) *string {
		return &c.ObjectStore.S3.SecretAccessKey
	}),
	stringEnv("objectstore.gcs.bucket", func(c *StateConfig) *string { return &c.ObjectStore.GCS.Bucket }),
	stringEnv("objectstore.gcs.endpoint", func(c *StateConfig) *string { return &c.ObjectStore.GCS.Endpoint }),
	boolEnv("objectstore.gcs.without_authentication", func(c *StateConfig) *bool {
		return &c.ObjectStore.GCS.WithoutAuthentication
	}),
	stringEnv("objectstore.azureblob.service_url", func(c *StateConfig) *string {
		return &c.ObjectStore.AzureBlob.ServiceURL
	}),
	stringEnv("objectstore.azureblob.account_name", func(c *StateConfig) *string {
		return &c.ObjectStore.AzureBlob.AccountName
	}),
	stringEnv("objectstore.azureblob.account_key", func(c *StateConfig) *string {
		return &c.ObjectStore.AzureBlob.AccountKey
	}),
	stringEnv("objectstore.azureblob.container", func(c *StateConfig) *string {
		return &c.ObjectStore.AzureBlob.Container
	}),
}

func applyStateEnvOverrides(
	config *StateConfig,
	lookupEnv func(string) (string, bool),
) error {
	for _, binding := range stateEnvBindings {
		name := stateEnvVarName(binding.key)
		// The deploy engine's config library treats a variable set to an empty string as
		// unset, which is what templates that expand a missing value to "" rely on.
		value, isSet := lookupEnv(name)
		if !isSet || value == "" {
			continue
		}

		if err := binding.apply(config, value); err != nil {
			return &StateAccessError{
				Code:    ErrCodeStateConfigInvalid,
				Message: fmt.Sprintf("invalid value for %s", name),
				Err:     err,
			}
		}
	}

	return nil
}

func stateEnvVarName(key string) string {
	return engineStateEnvPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
}

func stringEnv(key string, field func(*StateConfig) *string) stateEnvBinding {
	return stateEnvBinding{
		key: key,
		apply: func(config *StateConfig, value string) error {
			*field(config) = value
			return nil
		},
	}
}

func intEnv(key string, field func(*StateConfig) *int) stateEnvBinding {
	return stateEnvBinding{
		key: key,
		apply: func(config *StateConfig, value string) error {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			*field(config) = parsed
			return nil
		},
	}
}

func boolEnv(key string, field func(*StateConfig) *bool) stateEnvBinding {
	return stateEnvBinding{
		key: key,
		apply: func(config *StateConfig, value string) error {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			*field(config) = parsed
			return nil
		},
	}
}
