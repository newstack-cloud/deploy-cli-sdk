package stateio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type EngineConfigTestSuite struct {
	suite.Suite
}

func (s *EngineConfigTestSuite) writeConfigFile(contents string) string {
	path := filepath.Join(s.T().TempDir(), "config.json")
	s.Require().NoError(os.WriteFile(path, []byte(contents), 0600))
	return path
}

func (s *EngineConfigTestSuite) Test_loads_encryption_config_with_active_key_fields_inline() {
	path := s.writeConfigFile(`{
		"state": {
			"storage_engine": "memfile",
			"encryption": {
				"enabled": true,
				"provider": "awskms",
				"awskms": {"key_id": "alias/state", "region": "eu-west-2"},
				"previous_keys": [
					{"provider": "passphrase", "passphrase_file": "/run/secrets/old-passphrase"}
				]
			}
		}
	}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	encryption := config.State.Encryption
	s.True(encryption.Enabled)
	s.Equal("awskms", encryption.Provider)
	s.Equal("alias/state", encryption.AWSKMS.KeyID)
	s.Equal("eu-west-2", encryption.AWSKMS.Region)
	s.Require().Len(encryption.PreviousKeys, 1)
	s.Equal("passphrase", encryption.PreviousKeys[0].Provider)
	s.Equal("/run/secrets/old-passphrase", encryption.PreviousKeys[0].PassphraseFile)
}

func (s *EngineConfigTestSuite) Test_loads_objectstore_config() {
	path := s.writeConfigFile(`{
		"state": {
			"storage_engine": "objectstore",
			"objectstore": {
				"provider": "s3",
				"prefix": "team-a/",
				"s3": {"bucket": "state-bucket", "region": "eu-west-2", "use_path_style": true}
			}
		}
	}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	objectStore := config.State.ObjectStore
	s.Equal(StorageEngineObjectstore, config.State.StorageEngine)
	s.Equal("s3", objectStore.Provider)
	s.Equal("team-a/", objectStore.Prefix)
	s.Equal("state-bucket", objectStore.S3.Bucket)
	s.Equal("eu-west-2", objectStore.S3.Region)
	s.True(objectStore.S3.UsePathStyle)
}

func (s *EngineConfigTestSuite) Test_applies_engine_defaults_for_encryption_and_objectstore() {
	path := s.writeConfigFile(`{"state": {}}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.False(config.State.Encryption.Enabled)
	s.Equal("passphrase", config.State.Encryption.Provider)
	s.Equal("bluelink-state/", config.State.ObjectStore.Prefix)
}

func (s *EngineConfigTestSuite) Test_engine_environment_variables_override_the_config_file() {
	path := s.writeConfigFile(`{
		"state": {
			"storage_engine": "memfile",
			"postgres_port": 5432,
			"encryption": {"enabled": true, "passphrase": "from-file"},
			"objectstore": {"gcs": {"without_authentication": false}}
		}
	}`)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE", "postgres")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_PORT", "6543")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_ENABLED", "false")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_PASSPHRASE", "from-env")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_GCS_WITHOUT_AUTHENTICATION", "true")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_AZUREBLOB_ACCOUNT_KEY", "account-key")

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.Equal(StorageEnginePostgres, config.State.StorageEngine)
	s.Equal(6543, config.State.PostgresPort)
	s.False(config.State.Encryption.Enabled)
	s.Equal("from-env", config.State.Encryption.Passphrase)
	s.True(config.State.ObjectStore.GCS.WithoutAuthentication)
	s.Equal("account-key", config.State.ObjectStore.AzureBlob.AccountKey)
}

func (s *EngineConfigTestSuite) Test_fails_for_an_invalid_environment_variable_value() {
	path := s.writeConfigFile(`{"state": {}}`)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_ENABLED", "sometimes")

	_, err := LoadEngineConfig(path)

	s.Require().Error(err)
	s.Equal(
		"state_config_invalid: invalid value for BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_ENABLED: "+
			`strconv.ParseBool: parsing "sometimes": invalid syntax`,
		err.Error(),
	)
}

func (s *EngineConfigTestSuite) Test_ignores_environment_variables_set_to_an_empty_string() {
	path := s.writeConfigFile(`{"state": {"storage_engine": "postgres", "postgres_port": 5432}}`)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE", "")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_POSTGRES_PORT", "")

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.Equal(StorageEnginePostgres, config.State.StorageEngine)
	s.Equal(5432, config.State.PostgresPort)
}

func (s *EngineConfigTestSuite) Test_loads_a_config_from_environment_variables_alone() {
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_STORAGE_ENGINE", "objectstore")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_PROVIDER", "s3")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_S3_BUCKET", "engine-state")

	config, err := LoadEngineConfigFromEnv()

	s.Require().NoError(err)
	s.Equal(StorageEngineObjectstore, config.State.StorageEngine)
	s.Equal("engine-state", config.State.ObjectStore.S3.Bucket)
	s.Equal("bluelink-state/", config.State.ObjectStore.Prefix)
}

func (s *EngineConfigTestSuite) Test_keeps_an_objectstore_prefix_set_to_the_bucket_root() {
	path := s.writeConfigFile(`{"state": {"storage_engine": "objectstore", "objectstore": {"prefix": ""}}}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.Equal("", config.State.ObjectStore.Prefix)
}

func (s *EngineConfigTestSuite) Test_expands_environment_variables_in_the_memfile_state_dir() {
	s.T().Setenv("STATE_HOME", "/srv/bluelink")
	path := s.writeConfigFile(`{"state": {"memfile_state_dir": "$STATE_HOME/state"}}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.Equal("/srv/bluelink/state", config.State.MemFileStateDir)
}

func (s *EngineConfigTestSuite) Test_expands_windows_style_environment_variables_in_the_memfile_state_dir() {
	s.T().Setenv("STATE_HOME", "/srv/bluelink")
	path := s.writeConfigFile(`{"state": {"memfile_state_dir": "%STATE_HOME%/state"}}`)

	config, err := LoadEngineConfig(path)

	s.Require().NoError(err)
	s.Equal("/srv/bluelink/state", config.State.MemFileStateDir)
}

func (s *EngineConfigTestSuite) Test_finds_the_engine_config_file_the_engine_would_read() {
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)
	customDir := s.T().TempDir()
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH", customDir)

	found, err := FindEngineConfigFile()
	s.Require().NoError(err)
	s.Equal("", found, "expected no config file when neither directory has one")

	customPath := filepath.Join(customDir, "config.json")
	s.Require().NoError(os.WriteFile(customPath, []byte(`{}`), 0600))
	found, err = FindEngineConfigFile()
	s.Require().NoError(err)
	s.Equal(customPath, found)

	defaultDir := filepath.Join(home, ".bluelink", "engine")
	s.Require().NoError(os.MkdirAll(defaultDir, 0700))
	defaultPath := filepath.Join(defaultDir, "config.json")
	s.Require().NoError(os.WriteFile(defaultPath, []byte(`{}`), 0600))
	found, err = FindEngineConfigFile()
	s.Require().NoError(err)
	s.Equal(defaultPath, found, "expected the default directory to be searched first, as the engine does")
}

func (s *EngineConfigTestSuite) Test_skips_a_directory_named_like_an_engine_config_file() {
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)
	customDir := s.T().TempDir()
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH", customDir)
	s.Require().NoError(os.MkdirAll(filepath.Join(home, ".bluelink", "engine", "config.json"), 0700))
	customPath := filepath.Join(customDir, "config.json")
	s.Require().NoError(os.WriteFile(customPath, []byte(`{}`), 0600))

	found, err := FindEngineConfigFile()

	s.Require().NoError(err)
	s.Equal(customPath, found)
}

func (s *EngineConfigTestSuite) Test_reports_an_engine_config_directory_that_cannot_be_read() {
	if os.Geteuid() == 0 {
		s.T().Skip("file permissions do not apply to root")
	}
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH", "")
	defaultDir := filepath.Join(home, ".bluelink", "engine")
	s.Require().NoError(os.MkdirAll(defaultDir, 0700))
	s.Require().NoError(os.WriteFile(filepath.Join(defaultDir, "config.json"), []byte(`{}`), 0600))
	s.Require().NoError(os.Chmod(defaultDir, 0000))
	s.T().Cleanup(func() { _ = os.Chmod(defaultDir, 0700) })

	_, err := FindEngineConfigFile()

	accessErr, ok := errors.AsType[*StateAccessError](err)
	s.Require().True(ok, "expected a StateAccessError, got %v", err)
	s.Equal(ErrCodeStateConfigInvalid, accessErr.Code)
	s.Equal(
		"the deploy engine config file "+filepath.Join(defaultDir, "config.json")+" could not be checked",
		accessErr.Message,
	)
}

func (s *EngineConfigTestSuite) Test_refuses_an_engine_config_file_that_is_not_json() {
	home := s.T().TempDir()
	s.T().Setenv("HOME", home)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_CONFIG_PATH", "")
	defaultDir := filepath.Join(home, ".bluelink", "engine")
	s.Require().NoError(os.MkdirAll(defaultDir, 0700))
	s.Require().NoError(os.WriteFile(filepath.Join(defaultDir, "config.yaml"), []byte("state: {}"), 0600))

	_, err := FindEngineConfigFile()

	accessErr, ok := errors.AsType[*StateAccessError](err)
	s.Require().True(ok, "expected a StateAccessError, got %v", err)
	s.Equal(ErrCodeStateConfigInvalid, accessErr.Code)
	s.Equal(
		"the deploy engine config file "+filepath.Join(defaultDir, "config.yaml")+
			" is not JSON, which is the only format that can be read here",
		accessErr.Message,
	)
}

func TestEngineConfigTestSuite(t *testing.T) {
	suite.Run(t, new(EngineConfigTestSuite))
}

func (s *EngineConfigTestSuite) Test_postgres_url_escapes_the_credentials_and_database_name() {
	config := &StateConfig{
		PostgresUser:                "deploy@engine",
		PostgresPassword:            "p@ss/w:rd#1?",
		PostgresHost:                "db.internal",
		PostgresPort:                5433,
		PostgresDatabase:            "state db",
		PostgresSSLMode:             "require",
		PostgresPoolMaxConns:        10,
		PostgresPoolMaxConnLifetime: "1h30m",
	}

	parsed, err := pgxpool.ParseConfig(BuildPostgresDatabaseURL(config))
	s.Require().NoError(err)

	s.Equal("deploy@engine", parsed.ConnConfig.User)
	s.Equal("p@ss/w:rd#1?", parsed.ConnConfig.Password)
	s.Equal("db.internal", parsed.ConnConfig.Host)
	s.Equal(uint16(5433), parsed.ConnConfig.Port)
	s.Equal("state db", parsed.ConnConfig.Database)
	s.Equal(int32(10), parsed.MaxConns)
	s.Equal(90*time.Minute, parsed.MaxConnLifetime)
}
