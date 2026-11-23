package stateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

// StateConfig holds the state-related configuration from an engine config file.
// This is a subset of the deploy engine's StateConfig, containing only
// the fields needed for state import/export operations.
type StateConfig struct {
	StorageEngine               string `json:"storage_engine"`
	MemFileStateDir             string `json:"memfile_state_dir"`
	PostgresUser                string `json:"postgres_user"`
	PostgresPassword            string `json:"postgres_password"`
	PostgresHost                string `json:"postgres_host"`
	PostgresPort                int    `json:"postgres_port"`
	PostgresDatabase            string `json:"postgres_database"`
	PostgresSSLMode             string `json:"postgres_ssl_mode"`
	PostgresPoolMaxConns        int    `json:"postgres_pool_max_conns"`
	PostgresPoolMaxConnLifetime string `json:"postgres_pool_max_conn_lifetime"`
	// Encryption holds the configuration for encrypting persisted state at rest.
	// The deploy engine only supports this for the memfile storage engine.
	Encryption StateEncryptionConfig `json:"encryption"`
	// ObjectStore holds the configuration for the objectstore storage engine.
	ObjectStore ObjectStoreConfig `json:"objectstore"`
}

// StateEncryptionConfig mirrors the deploy engine's state encryption configuration.
type StateEncryptionConfig struct {
	Enabled bool `json:"enabled"`
	// StateEncryptionKeyConfig is embedded so the active key's fields sit directly
	// under "encryption", matching the deploy engine's config file layout.
	StateEncryptionKeyConfig
	// PreviousKeys holds keys that state is no longer encrypted with but that
	// objects written before a key change may still be sealed with.
	PreviousKeys []StateEncryptionKeyConfig `json:"previous_keys"`
}

// StateEncryptionKeyConfig identifies one key that state is, or was, encrypted with.
type StateEncryptionKeyConfig struct {
	// Provider is one of "passphrase", "localkey", "awskms", "gcpkms" or "azurekeyvault".
	Provider       string                             `json:"provider"`
	Passphrase     string                             `json:"passphrase"`
	PassphraseFile string                             `json:"passphrase_file"`
	LocalKeyFile   string                             `json:"local_key_file"`
	AWSKMS         StateEncryptionAWSKMSConfig        `json:"awskms"`
	GCPKMS         StateEncryptionGCPKMSConfig        `json:"gcpkms"`
	AzureKeyVault  StateEncryptionAzureKeyVaultConfig `json:"azurekeyvault"`
}

// StateEncryptionAWSKMSConfig configures state encryption against AWS KMS.
type StateEncryptionAWSKMSConfig struct {
	KeyID           string `json:"key_id"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

// StateEncryptionGCPKMSConfig configures state encryption against Google Cloud KMS.
type StateEncryptionGCPKMSConfig struct {
	KeyName         string `json:"key_name"`
	CredentialsFile string `json:"credentials_file"`
}

// StateEncryptionAzureKeyVaultConfig configures state encryption against Azure Key Vault.
type StateEncryptionAzureKeyVaultConfig struct {
	VaultURL   string `json:"vault_url"`
	KeyName    string `json:"key_name"`
	KeyVersion string `json:"key_version"`
	Algorithm  string `json:"algorithm"`
}

// ObjectStoreConfig mirrors the deploy engine's objectstore storage engine configuration.
type ObjectStoreConfig struct {
	// Provider is one of "s3", "gcs" or "azureblob".
	Provider string `json:"provider"`
	// Prefix is prepended to every storage key in the bucket or container.
	Prefix    string                     `json:"prefix"`
	S3        ObjectStoreS3Config        `json:"s3"`
	GCS       ObjectStoreGCSConfig       `json:"gcs"`
	AzureBlob ObjectStoreAzureBlobConfig `json:"azureblob"`
}

// ObjectStoreS3Config configures the S3-backed objectstore storage engine.
type ObjectStoreS3Config struct {
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	UsePathStyle    bool   `json:"use_path_style"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

// ObjectStoreGCSConfig configures the Google Cloud Storage-backed objectstore storage engine.
type ObjectStoreGCSConfig struct {
	Bucket                string `json:"bucket"`
	Endpoint              string `json:"endpoint"`
	WithoutAuthentication bool   `json:"without_authentication"`
}

// ObjectStoreAzureBlobConfig configures the Azure Blob Storage-backed objectstore storage engine.
type ObjectStoreAzureBlobConfig struct {
	ServiceURL  string `json:"service_url"`
	AccountName string `json:"account_name"`
	AccountKey  string `json:"account_key"`
	Container   string `json:"container"`
}

// EngineConfig represents the deploy engine configuration file structure.
// Only the state-related fields are parsed.
type EngineConfig struct {
	State StateConfig `json:"state"`
}

const (
	StorageEngineMemfile     = "memfile"
	StorageEnginePostgres    = "postgres"
	StorageEngineObjectstore = "objectstore"
)

const (
	defaultEncryptionKeyProvider = "passphrase"
	defaultObjectStorePrefix     = "bluelink-state/"
)

// LoadEngineConfig loads and parses a deploy engine configuration file.
//
// State settings provided through the deploy engine's environment variables
// (BLUELINK_DEPLOY_ENGINE_STATE_*) take precedence over the file, matching how
// the deploy engine resolves its own configuration. Secrets such as encryption
// passphrases and storage credentials are commonly supplied this way.
func LoadEngineConfig(configFilePath string) (*EngineConfig, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, &StateAccessError{
			Code:    ErrCodeStateConfigInvalid,
			Message: "failed to read engine config file",
			Err:     err,
		}
	}

	var config EngineConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, &StateAccessError{
			Code:    ErrCodeStateConfigInvalid,
			Message: fmt.Sprintf("failed to parse engine config file %s", configFilePath),
			Err:     err,
		}
	}

	return completeEngineConfig(&config, hasObjectStorePrefix(data))
}

// The deploy engine only applies the default prefix when none is set, so a prefix set to
// an empty string, for state at the root of the bucket, is kept. An environment variable
// set to an empty string counts as unset, so only the file can set an empty prefix.
func hasObjectStorePrefix(data []byte) bool {
	var probe struct {
		State struct {
			ObjectStore struct {
				Prefix *string `json:"prefix"`
			} `json:"objectstore"`
		} `json:"state"`
	}
	return json.Unmarshal(data, &probe) == nil && probe.State.ObjectStore.Prefix != nil
}

// LoadEngineConfigFromEnv builds a deploy engine configuration from the deploy engine's
// environment variables (BLUELINK_DEPLOY_ENGINE_STATE_*) and defaults alone, for a deploy
// engine that runs without a config file, as the deploy engine itself can.
func LoadEngineConfigFromEnv() (*EngineConfig, error) {
	return completeEngineConfig(&EngineConfig{}, false)
}

func completeEngineConfig(config *EngineConfig, prefixIsSet bool) (*EngineConfig, error) {
	if err := applyStateEnvOverrides(&config.State, os.LookupEnv); err != nil {
		return nil, err
	}

	config.State.MemFileStateDir = expandEngineEnv(config.State.MemFileStateDir)

	applyStateConfigDefaults(&config.State)
	if config.State.ObjectStore.Prefix == "" && !prefixIsSet {
		config.State.ObjectStore.Prefix = defaultObjectStorePrefix
	}

	return config, nil
}

var windowsEnvVarReference = regexp.MustCompile(`%[^%]+%`)

// Expands environment variables the way the deploy engine does in the state directory, so a
// config shared with the engine resolves to the same place. Windows style %VAR% references
// are expanded on every OS as well as $VAR and ${VAR}.
func expandEngineEnv(value string) string {
	unixStyle := windowsEnvVarReference.ReplaceAllStringFunc(value, func(reference string) string {
		return "${" + reference[1:len(reference)-1] + "}"
	})
	return os.ExpandEnv(unixStyle)
}

func applyStateConfigDefaults(config *StateConfig) {
	if config.StorageEngine == "" {
		config.StorageEngine = StorageEngineMemfile
	}
	if config.MemFileStateDir == "" {
		config.MemFileStateDir = GetDefaultStateDir()
	}
	if config.PostgresHost == "" {
		config.PostgresHost = "localhost"
	}
	if config.PostgresPort == 0 {
		config.PostgresPort = 5432
	}
	if config.PostgresSSLMode == "" {
		config.PostgresSSLMode = "disable"
	}
	if config.PostgresPoolMaxConns == 0 {
		config.PostgresPoolMaxConns = 100
	}
	if config.PostgresPoolMaxConnLifetime == "" {
		config.PostgresPoolMaxConnLifetime = "1h30m"
	}
	if config.Encryption.Provider == "" {
		config.Encryption.Provider = defaultEncryptionKeyProvider
	}
}

// GetDefaultStateDir returns the default state directory based on the OS.
func GetDefaultStateDir() string {
	if runtime.GOOS == "windows" {
		// os.ExpandEnv only expands $VAR, not the %VAR% form Windows paths are written in.
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "NewStack", "Bluelink", "engine", "state")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/state")
}

// GetDefaultEngineConfigPath returns the default engine config file path.
func GetDefaultEngineConfigPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "NewStack", "Bluelink", "engine", "config.json")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/config.json")
}

// The extensions the deploy engine's config library reads a config file with, besides
// JSON, which is the only one this package parses.
var nonJSONEngineConfigExtensions = []string{
	"toml", "yaml", "yml", "properties", "props", "prop", "hcl", "tfvars", "dotenv", "env", "ini",
}

// FindEngineConfigFile returns the deploy engine config file the deploy engine itself
// would read: config.json in the default config directory, or else in the directory
// BLUELINK_DEPLOY_ENGINE_CONFIG_PATH names. It returns an empty path when there is none,
// in which case the deploy engine runs on its environment variables and defaults alone.
//
// The deploy engine also reads config files in other formats, which this package cannot
// parse, so finding one of those is an error rather than a reason to fall back to the
// defaults and use the wrong state. The deploy engine's working directory is searched
// first too, but it is not searched here: it is rarely the directory the CLI runs in, and
// another project's config.json there would be taken for the engine's.
func FindEngineConfigFile() (string, error) {
	dirs := []string{filepath.Dir(GetDefaultEngineConfigPath())}
	if customDir, isSet := os.LookupEnv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH"); isSet && customDir != "" {
		dirs = append(dirs, customDir)
	}

	for _, dir := range dirs {
		jsonPath := filepath.Join(dir, "config.json")
		exists, err := engineConfigFileExists(jsonPath)
		if err != nil {
			return "", err
		}
		if exists {
			return jsonPath, nil
		}
		if err := checkNoOtherEngineConfigFormat(dir); err != nil {
			return "", err
		}
	}

	return "", nil
}

func checkNoOtherEngineConfigFormat(dir string) error {
	for _, extension := range nonJSONEngineConfigExtensions {
		otherPath := filepath.Join(dir, "config."+extension)
		exists, err := engineConfigFileExists(otherPath)
		if err != nil {
			return err
		}
		if exists {
			return &StateAccessError{
				Code: ErrCodeStateConfigInvalid,
				Message: fmt.Sprintf(
					"the deploy engine config file %s is not JSON, which is the only format that can be read here",
					otherPath,
				),
				Hint: "Pass a JSON copy of it with --engine-config-file.",
			}
		}
	}
	return nil
}

// Only a file that is not there moves the search on, and a directory of that name counts as
// not there, as it does for the engine. A file that cannot be checked, such as one in a
// directory without read access, is reported, since falling back to the defaults would
// import into or export from state other than the engine's.
func engineConfigFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, &StateAccessError{
		Code:    ErrCodeStateConfigInvalid,
		Message: fmt.Sprintf("the deploy engine config file %s could not be checked", path),
		Err:     err,
	}
}

// BuildPostgresDatabaseURL constructs a postgres connection URL from config.
// The user, password and database name are escaped, so a password containing
// characters such as "@", "/" or "#" connects as it is written.
func BuildPostgresDatabaseURL(config *StateConfig) string {
	query := url.Values{}
	query.Set("sslmode", config.PostgresSSLMode)
	query.Set("pool_max_conns", strconv.Itoa(config.PostgresPoolMaxConns))
	query.Set("pool_max_conn_lifetime", config.PostgresPoolMaxConnLifetime)

	databaseURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.PostgresUser, config.PostgresPassword),
		Host:     net.JoinHostPort(config.PostgresHost, strconv.Itoa(config.PostgresPort)),
		Path:     "/" + config.PostgresDatabase,
		RawQuery: query.Encode(),
	}
	return databaseURL.String()
}
