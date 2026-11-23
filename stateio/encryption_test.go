package stateio

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	encryptionConfigHint = "Check state.encryption in the deploy engine config, and any " +
		"BLUELINK_DEPLOY_ENGINE_STATE_ENCRYPTION_* environment variables."
	stateDecryptionHint = "Check that the configured key is the one this state was encrypted with. " +
		"If the key has been rotated, list the old key under state.encryption.previous_keys. " +
		"If the state was written before encryption was enabled, encrypt it with the " +
		"deploy engine's reencrypt-state command."
	plaintextExportWarning = "State is encrypted at rest but the export file is not; it can contain " +
		"resolved secrets, so encrypt the export or store it securely and remove it once it is no " +
		"longer needed"
	missingPassphraseCause = "one of state.encryption.passphrase or state.encryption.passphrase_file " +
		`is required for the "passphrase" key provider`
)

func unsupportedEncryptionError(engine string) string {
	return `state_config_invalid: state encryption is only supported for the "memfile" storage engine, ` +
		`the "` + engine + `" engine relies on encryption at rest in the database or bucket it ` +
		"persists to (" + encryptionConfigHint + ")"
}

type StateEncryptionTestSuite struct {
	suite.Suite
	fs       afero.Fs
	stateDir string
}

func (s *StateEncryptionTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
	s.stateDir = "/test/state"
	s.Require().NoError(s.fs.MkdirAll(s.stateDir, 0700))
}

func (s *StateEncryptionTestSuite) engineConfig(encryption StateEncryptionConfig) *EngineConfig {
	return &EngineConfig{
		State: StateConfig{
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: s.stateDir,
			Encryption:      encryption,
		},
	}
}

func passphraseEncryption(passphrase string, previous ...StateEncryptionKeyConfig) StateEncryptionConfig {
	return StateEncryptionConfig{
		Enabled: true,
		StateEncryptionKeyConfig: StateEncryptionKeyConfig{
			Provider:   "passphrase",
			Passphrase: passphrase,
		},
		PreviousKeys: previous,
	}
}

func testInstances() []state.InstanceState {
	return []state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "orders",
			Status:       core.InstanceStatusDeployed,
			Resources: map[string]*state.ResourceState{
				"res-001": {
					ResourceID: "res-001",
					Name:       "ordersTable",
					Type:       "aws/dynamodb/table",
					SpecData:   core.MappingNodeFromString("resolved-secret-value"),
				},
			},
			ResourceIDs: map[string]string{"ordersTable": "res-001"},
		},
	}
}

func (s *StateEncryptionTestSuite) importInstances(config *EngineConfig) error {
	data, err := json.Marshal(testInstances())
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/input.json",
		EngineConfig: config,
		FileSystem:   s.fs,
		FileData:     data,
	})
	return err
}

func (s *StateEncryptionTestSuite) export(config *EngineConfig) (*ExportResult, error) {
	return Export(ExportParams{
		FilePath:     "/test/export.json",
		EngineConfig: config,
		FileSystem:   s.fs,
	})
}

func (s *StateEncryptionTestSuite) saltPath() string {
	return filepath.Join(s.stateDir, "encryption_salt.json")
}

func (s *StateEncryptionTestSuite) Test_import_encrypts_state_written_to_the_store() {
	err := s.importInstances(s.engineConfig(passphraseEncryption("correct horse")))
	s.Require().NoError(err)

	exists, err := afero.Exists(s.fs, s.saltPath())
	s.Require().NoError(err)
	s.True(exists, "expected the import to create an encryption salt")

	stateFiles := 0
	walkErr := afero.Walk(s.fs, s.stateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == s.saltPath() {
			return err
		}
		stateFiles += 1
		contents, readErr := afero.ReadFile(s.fs, path)
		s.Require().NoError(readErr)
		s.True(crypto.IsEnvelope(contents), "expected %s to be encrypted", path)
		s.NotContains(string(contents), "resolved-secret-value")
		return nil
	})
	s.Require().NoError(walkErr)
	s.Positive(stateFiles)
}

func (s *StateEncryptionTestSuite) Test_round_trips_encrypted_state_and_warns_the_export_is_plaintext() {
	config := s.engineConfig(passphraseEncryption("correct horse"))
	s.Require().NoError(s.importInstances(config))

	result, err := s.export(config)

	s.Require().NoError(err)
	s.Equal(1, result.InstancesCount)
	s.Require().Len(result.Warnings, 1)
	s.Equal(plaintextExportWarning, result.Warnings[0])

	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)
	var exported []state.InstanceState
	s.Require().NoError(json.Unmarshal(data, &exported))
	s.Require().Len(exported, 1)
	s.Equal("orders", exported[0].InstanceName)
	s.Equal("resolved-secret-value", core.StringValue(exported[0].Resources["res-001"].SpecData))
}

func (s *StateEncryptionTestSuite) Test_exports_unencrypted_state_without_warnings() {
	config := s.engineConfig(StateEncryptionConfig{})
	s.Require().NoError(s.importInstances(config))

	result, err := s.export(config)

	s.Require().NoError(err)
	s.Empty(result.Warnings)
}

func (s *StateEncryptionTestSuite) Test_reads_state_sealed_with_a_previous_key() {
	s.Require().NoError(s.importInstances(s.engineConfig(passphraseEncryption("old passphrase"))))

	rotated := passphraseEncryption(
		"new passphrase",
		StateEncryptionKeyConfig{Provider: "passphrase", Passphrase: "old passphrase"},
	)
	result, err := s.export(s.engineConfig(rotated))

	s.Require().NoError(err)
	s.Equal(1, result.InstancesCount)
}

func (s *StateEncryptionTestSuite) Test_fails_with_a_decryption_error_for_the_wrong_key() {
	s.Require().NoError(s.importInstances(s.engineConfig(passphraseEncryption("correct horse"))))

	_, err := s.export(s.engineConfig(passphraseEncryption("battery staple")))

	s.assertStateAccessError(err, ErrCodeStateDecryptionFailed)
	explanation := ExplainError(err)
	s.Equal("the state could not be decrypted", explanation.Summary)
	s.Equal(stateDecryptionHint, explanation.Hint)
	s.True(strings.HasPrefix(explanation.Cause, "envelope was sealed with key"), explanation.Cause)
}

func (s *StateEncryptionTestSuite) Test_suggests_reencrypting_state_written_before_encryption_was_enabled() {
	s.Require().NoError(s.importInstances(s.engineConfig(StateEncryptionConfig{})))

	err := s.importInstances(s.engineConfig(passphraseEncryption("correct horse")))

	s.assertStateAccessError(err, ErrCodeStateDecryptionFailed)
	explanation := ExplainError(err)
	s.Equal(
		`"/test/state/instance_index.json" is not encrypted, so this state was written before `+
			"encryption was enabled; re-encrypt the state directory or start from an empty one",
		explanation.Cause,
	)
	s.Equal(stateDecryptionHint, explanation.Hint)
}

func (s *StateEncryptionTestSuite) Test_a_refused_import_leaves_an_empty_encrypted_store_without_a_salt() {
	inputs := map[string][]byte{
		"invalid input": []byte(`{"not": "an array"}`),
		"a conflicting name": []byte(
			`[{"id":"inst-1","name":"orders","status":2},{"id":"inst-2","name":"orders","status":2}]`,
		),
	}
	for description, input := range inputs {
		_, err := Import(ImportParams{
			FilePath:     "/test/input.json",
			EngineConfig: s.engineConfig(passphraseEncryption("correct horse")),
			FileSystem:   s.fs,
			FileData:     input,
		})

		s.Require().Error(err, description)
		s.assertNoSalt(description)
	}
}

func (s *StateEncryptionTestSuite) Test_an_import_of_no_instances_leaves_no_salt() {
	for _, input := range []string{`[]`, `null`} {
		result, err := Import(ImportParams{
			FilePath:     "/test/input.json",
			EngineConfig: s.engineConfig(passphraseEncryption("correct horse")),
			FileSystem:   s.fs,
			FileData:     []byte(input),
		})

		s.Require().NoError(err, input)
		s.Equal(0, result.InstancesCount, input)
		s.assertNoSalt("an import of " + input)
	}
}

func (s *StateEncryptionTestSuite) Test_an_import_that_cannot_set_up_the_key_leaves_no_salt() {
	err := s.importInstances(s.engineConfig(passphraseEncryption("")))

	s.assertStateAccessError(err, ErrCodeEncryptionKeyUnavailable)
	s.assertNoSalt("missing passphrase")
}

func (s *StateEncryptionTestSuite) Test_an_import_into_plaintext_state_leaves_no_salt() {
	s.Require().NoError(s.importInstances(s.engineConfig(StateEncryptionConfig{})))

	err := s.importInstances(s.engineConfig(passphraseEncryption("correct horse")))

	s.assertStateAccessError(err, ErrCodeStateDecryptionFailed)
	s.assertNoSalt("plaintext state")
}

func (s *StateEncryptionTestSuite) Test_a_refused_import_keeps_an_existing_salt() {
	config := s.engineConfig(passphraseEncryption("correct horse"))
	s.Require().NoError(s.importInstances(config))

	_, err := Import(ImportParams{
		FilePath:     "/test/input.json",
		EngineConfig: config,
		FileSystem:   s.fs,
		FileData:     []byte(`[{"id":"other","name":"orders","status":2},{"id":"another","name":"orders","status":2}]`),
	})

	s.Require().Error(err)
	exists, existsErr := afero.Exists(s.fs, s.saltPath())
	s.Require().NoError(existsErr)
	s.True(exists, "expected the salt the stored state was encrypted with to be kept")
}

func (s *StateEncryptionTestSuite) assertNoSalt(description string) {
	exists, err := afero.Exists(s.fs, s.saltPath())
	s.Require().NoError(err)
	s.False(exists, "expected the store to be left without a salt after %s", description)
}

func (s *StateEncryptionTestSuite) Test_export_does_not_create_a_salt_in_a_store_without_one() {
	s.Require().NoError(s.importInstances(s.engineConfig(StateEncryptionConfig{})))

	_, err := s.export(s.engineConfig(passphraseEncryption("correct horse")))

	s.assertStateAccessError(err, ErrCodeEncryptionKeyUnavailable)
	s.Equal(
		"encryption_key_unavailable: the state encryption key could not be set up ("+encryptionConfigHint+"): "+
			`no encryption salt was found at "/test/state/encryption_salt.json", so the passphrase cannot `+
			"derive the key the state was encrypted with; check that state.memfile_state_dir points at "+
			"state written with encryption enabled",
		err.Error(),
	)
	exists, existsErr := afero.Exists(s.fs, s.saltPath())
	s.Require().NoError(existsErr)
	s.False(exists, "expected the export to leave the store without a salt")
}

func (s *StateEncryptionTestSuite) Test_fails_when_no_passphrase_is_configured() {
	_, err := s.export(s.engineConfig(passphraseEncryption("")))

	s.assertStateAccessError(err, ErrCodeEncryptionKeyUnavailable)
	s.Equal(
		"encryption_key_unavailable: the state encryption key could not be set up ("+encryptionConfigHint+"): "+
			missingPassphraseCause,
		err.Error(),
	)
	explanation := ExplainError(err)
	s.Equal("the state encryption key could not be set up", explanation.Summary)
	s.Equal(missingPassphraseCause, explanation.Cause)
	s.Equal(encryptionConfigHint, explanation.Hint)
}

func (s *StateEncryptionTestSuite) Test_rejects_encryption_for_the_postgres_storage_engine() {
	config := s.engineConfig(passphraseEncryption("correct horse"))
	config.State.StorageEngine = StorageEnginePostgres

	_, err := s.export(config)

	s.Require().Error(err)
	s.Equal(unsupportedEncryptionError(StorageEnginePostgres), err.Error())
}

func (s *StateEncryptionTestSuite) Test_rejects_encryption_for_the_objectstore_storage_engine() {
	config := s.engineConfig(passphraseEncryption("correct horse"))
	config.State.StorageEngine = StorageEngineObjectstore

	_, err := s.export(config)

	s.Require().Error(err)
	s.Equal(unsupportedEncryptionError(StorageEngineObjectstore), err.Error())
}

func (s *StateEncryptionTestSuite) assertStateAccessError(err error, expectedCode StateAccessErrorCode) {
	s.Require().Error(err)
	var accessErr *StateAccessError
	s.Require().True(errors.As(err, &accessErr), "expected a StateAccessError, got %v", err)
	s.Equal(expectedCode, accessErr.Code)
}

func TestStateEncryptionTestSuite(t *testing.T) {
	suite.Run(t, new(StateEncryptionTestSuite))
}
