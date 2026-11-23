package stateio

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type FileEncryptionTestSuite struct {
	suite.Suite
	fs afero.Fs
}

func (s *FileEncryptionTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
}

func (s *FileEncryptionTestSuite) memfileConfig(stateDir string, encryption StateEncryptionConfig) *EngineConfig {
	s.Require().NoError(s.fs.MkdirAll(stateDir, 0700))
	return &EngineConfig{
		State: StateConfig{
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: stateDir,
			Encryption:      encryption,
		},
	}
}

func (s *FileEncryptionTestSuite) seededSource(encryption StateEncryptionConfig) *EngineConfig {
	config := s.memfileConfig("/source/state", encryption)
	data, err := json.Marshal(testInstances())
	s.Require().NoError(err)

	_, err = Import(ImportParams{EngineConfig: config, FileSystem: s.fs, FileData: data})
	s.Require().NoError(err)
	return config
}

func (s *FileEncryptionTestSuite) exportTo(
	config *EngineConfig,
	path string,
	key *StateEncryptionKeyConfig,
) *ExportResult {
	result, err := Export(ExportParams{
		FilePath:       path,
		EngineConfig:   config,
		FileSystem:     s.fs,
		FileEncryption: key,
	})
	s.Require().NoError(err)
	return result
}

func (s *FileEncryptionTestSuite) importFile(path string, key *StateEncryptionKeyConfig) (*ImportResult, error) {
	data, err := afero.ReadFile(s.fs, path)
	s.Require().NoError(err)

	return Import(ImportParams{
		FilePath:       path,
		FileData:       data,
		EngineConfig:   s.memfileConfig("/target/state", StateEncryptionConfig{}),
		FileSystem:     s.fs,
		FileEncryption: key,
	})
}

func (s *FileEncryptionTestSuite) importedInstances() []state.InstanceState {
	_, err := Export(ExportParams{
		FilePath:     "/target-export.json",
		EngineConfig: s.memfileConfig("/target/state", StateEncryptionConfig{}),
		FileSystem:   s.fs,
	})
	s.Require().NoError(err)

	data, err := afero.ReadFile(s.fs, "/target-export.json")
	s.Require().NoError(err)
	var instances []state.InstanceState
	s.Require().NoError(json.Unmarshal(data, &instances))
	return instances
}

func passphraseKey(passphrase string) *StateEncryptionKeyConfig {
	return &StateEncryptionKeyConfig{
		Provider:   "passphrase",
		Passphrase: passphrase,
	}
}

func (s *FileEncryptionTestSuite) Test_round_trips_a_passphrase_encrypted_file() {
	source := s.seededSource(StateEncryptionConfig{})

	result := s.exportTo(source, "/export.json", passphraseKey("correct horse"))
	s.True(result.Encrypted)

	contents, err := afero.ReadFile(s.fs, "/export.json")
	s.Require().NoError(err)
	s.NotContains(string(contents), "resolved-secret-value")
	s.NotContains(string(contents), "orders")

	_, err = s.importFile("/export.json", passphraseKey("correct horse"))
	s.Require().NoError(err)
	instances := s.importedInstances()
	s.Require().Len(instances, 1)
	s.Equal("orders", instances[0].InstanceName)
}

func (s *FileEncryptionTestSuite) Test_round_trips_a_file_encrypted_with_a_local_key() {
	keyBytes, err := crypto.GenerateLocalKey()
	s.Require().NoError(err)
	keyPath := filepath.Join(s.T().TempDir(), "export.key")
	s.Require().NoError(os.WriteFile(keyPath, keyBytes, 0600))
	key, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{Key: "localkey:" + keyPath})
	s.Require().NoError(err)

	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", key)

	_, err = s.importFile("/export.json", key)
	s.Require().NoError(err)
	s.Len(s.importedInstances(), 1)
}

func (s *FileEncryptionTestSuite) Test_decrypts_a_file_after_it_is_moved_and_renamed() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/ci/artifacts/state.json", passphraseKey("correct horse"))
	contents, err := afero.ReadFile(s.fs, "/ci/artifacts/state.json")
	s.Require().NoError(err)
	s.Require().NoError(afero.WriteFile(s.fs, "/downloads/renamed-state.json", contents, 0600))

	_, err = s.importFile("/downloads/renamed-state.json", passphraseKey("correct horse"))

	s.Require().NoError(err)
}

func (s *FileEncryptionTestSuite) Test_requires_a_key_to_import_an_encrypted_file() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", passphraseKey("correct horse"))

	_, err := s.importFile("/export.json", nil)

	s.assertImportErrorCode(err, ErrCodeFileKeyRequired)
}

func (s *FileEncryptionTestSuite) Test_fails_to_import_an_encrypted_file_with_the_wrong_key() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", passphraseKey("correct horse"))

	_, err := s.importFile("/export.json", passphraseKey("battery staple"))

	s.assertImportErrorCode(err, ErrCodeFileDecryptionFailed)
	s.Empty(s.importedInstances(), "a failed decryption must not write to the target store")
}

func (s *FileEncryptionTestSuite) Test_imports_a_plaintext_file_when_a_key_is_provided() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", nil)

	_, err := s.importFile("/export.json", passphraseKey("correct horse"))

	s.Require().NoError(err)
	s.Len(s.importedInstances(), 1)
}

func (s *FileEncryptionTestSuite) Test_refuses_a_plaintext_file_when_an_encrypted_one_is_required() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", nil)
	data, err := afero.ReadFile(s.fs, "/export.json")
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:             "/export.json",
		FileData:             data,
		EngineConfig:         s.memfileConfig("/target/state", StateEncryptionConfig{}),
		FileSystem:           s.fs,
		FileEncryption:       passphraseKey("correct horse"),
		RequireEncryptedFile: true,
	})

	s.assertImportErrorCode(err, ErrCodeFileNotEncrypted)
	s.Empty(s.importedInstances())
}

func (s *FileEncryptionTestSuite) Test_does_not_warn_when_encrypted_state_is_exported_to_an_encrypted_file() {
	source := s.seededSource(passphraseEncryption("state passphrase"))

	result := s.exportTo(source, "/export.json", passphraseKey("file passphrase"))

	s.True(result.Encrypted)
	s.Empty(result.Warnings)
}

func (s *FileEncryptionTestSuite) Test_reports_when_an_encrypted_file_was_exported() {
	before := time.Now().Add(-time.Second)
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", passphraseKey("correct horse"))

	result, err := s.importFile("/export.json", passphraseKey("correct horse"))

	s.Require().NoError(err)
	s.True(result.Encrypted)
	s.WithinRange(result.FileExportedAt, before, time.Now())
}

func (s *FileEncryptionTestSuite) Test_refuses_an_encrypted_file_whose_export_time_was_changed() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", passphraseKey("correct horse"))
	s.rewriteExportFile("/export.json", func(file map[string]any) {
		file["exported_at"] = "2099-01-01T00:00:00Z"
	})

	_, err := s.importFile("/export.json", passphraseKey("correct horse"))

	s.assertImportErrorCode(err, ErrCodeFileDecryptionFailed)
	s.Empty(s.importedInstances())
}

func (s *FileEncryptionTestSuite) Test_imports_a_plaintext_file_saved_with_a_byte_order_mark() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", nil)
	contents, err := afero.ReadFile(s.fs, "/export.json")
	s.Require().NoError(err)
	s.Require().NoError(afero.WriteFile(s.fs, "/export.json", append([]byte("\xef\xbb\xbf"), contents...), 0600))

	_, err = s.importFile("/export.json", nil)

	s.Require().NoError(err)
	s.Len(s.importedInstances(), 1)
}

func (s *FileEncryptionTestSuite) Test_reports_a_changed_format_field_as_not_a_state_export() {
	s.exportTo(s.seededSource(StateEncryptionConfig{}), "/export.json", passphraseKey("correct horse"))
	s.rewriteExportFile("/export.json", func(file map[string]any) {
		file["format"] = "something-else"
	})

	_, err := s.importFile("/export.json", passphraseKey("correct horse"))

	s.assertImportErrorCode(err, ErrCodeInvalidJSON)
	s.Equal(
		"invalid_json: the input file is not a state export (A state export is a JSON array of instances, "+
			`or an encrypted export whose "format" is "bluelink-state-export". If this is an encrypted `+
			"export, its format field has been changed.)",
		err.Error(),
	)
}

func (s *FileEncryptionTestSuite) rewriteExportFile(path string, change func(file map[string]any)) {
	contents, err := afero.ReadFile(s.fs, path)
	s.Require().NoError(err)
	file := map[string]any{}
	s.Require().NoError(json.Unmarshal(contents, &file))
	change(file)
	contents, err = json.Marshal(file)
	s.Require().NoError(err)
	s.Require().NoError(afero.WriteFile(s.fs, path, contents, 0600))
}

func (s *FileEncryptionTestSuite) assertImportErrorCode(err error, expected ImportErrorCode) {
	s.Require().Error(err)
	var importErr *ImportError
	s.Require().True(errors.As(err, &importErr), "expected an ImportError, got %v", err)
	s.Equal(expected, importErr.Code)
}

func TestFileEncryptionTestSuite(t *testing.T) {
	suite.Run(t, new(FileEncryptionTestSuite))
}

type ResolveFileEncryptionKeyTestSuite struct {
	suite.Suite
}

func (s *ResolveFileEncryptionKeyTestSuite) Test_resolves_each_key_provider() {
	cases := map[string]StateEncryptionKeyConfig{
		"awskms:arn:aws:kms:eu-west-2:123456789012:alias/state-export": {
			Provider: "awskms",
			AWSKMS:   StateEncryptionAWSKMSConfig{KeyID: "arn:aws:kms:eu-west-2:123456789012:alias/state-export"},
		},
		"gcpkms:projects/p/locations/l/keyRings/r/cryptoKeys/k": {
			Provider: "gcpkms",
			GCPKMS:   StateEncryptionGCPKMSConfig{KeyName: "projects/p/locations/l/keyRings/r/cryptoKeys/k"},
		},
		"azurekeyvault:https://team.vault.azure.net/keys/state-export/abc123": {
			Provider: "azurekeyvault",
			AzureKeyVault: StateEncryptionAzureKeyVaultConfig{
				VaultURL:   "https://team.vault.azure.net",
				KeyName:    "state-export",
				KeyVersion: "abc123",
			},
		},
		"localkey:/run/secrets/export.key": {
			Provider:     "localkey",
			LocalKeyFile: "/run/secrets/export.key",
		},
	}

	for key, expected := range cases {
		resolved, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{Key: key})
		s.Require().NoError(err, key)
		s.Equal(expected, *resolved, key)
	}
}

func (s *ResolveFileEncryptionKeyTestSuite) Test_never_repeats_a_malformed_key_in_its_error() {
	keys := []string{
		"correct-horse-battery",
		"correct:horse-battery",
		"azurekeyvault:correct-horse-battery",
		"azurekeyvault:https://vault.vault.azure.net/correct-horse-battery",
	}
	for _, key := range keys {
		_, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{Key: key})

		s.Require().Error(err, key)
		s.NotContains(err.Error(), "correct", key)
		s.NotContains(err.Error(), "horse", key)
	}
}

func (s *ResolveFileEncryptionKeyTestSuite) Test_prefers_explicit_options_over_the_environment_passphrase() {
	fromKey, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{
		Key:        "localkey:/run/secrets/export.key",
		Passphrase: "from-env",
	})
	s.Require().NoError(err)
	s.Equal("localkey", fromKey.Provider)

	fromFile, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{
		PassphraseFile: "/run/secrets/passphrase",
		Passphrase:     "from-env",
	})
	s.Require().NoError(err)
	s.Equal("/run/secrets/passphrase", fromFile.PassphraseFile)
	s.Empty(fromFile.Passphrase)
}

func (s *ResolveFileEncryptionKeyTestSuite) Test_returns_no_key_when_none_is_given() {
	resolved, err := ResolveFileEncryptionKey(FileEncryptionKeyOptions{})

	s.Require().NoError(err)
	s.Nil(resolved)
}

func (s *ResolveFileEncryptionKeyTestSuite) Test_rejects_invalid_keys() {
	invalid := map[string]FileEncryptionKeyOptions{
		"both key and passphrase file": {Key: "localkey:/k", PassphraseFile: "/p"},
		"no separator":                 {Key: "alias/state-export"},
		"empty id":                     {Key: "awskms:"},
		"unsupported provider":         {Key: "vault:secret/key"},
		"passphrase as a key":          {Key: "passphrase:hunter2"},
		"azure url without a key":      {Key: "azurekeyvault:https://team.vault.azure.net/secrets/x"},
		"azure url over http":          {Key: "azurekeyvault:http://team.vault.azure.net/keys/x"},
	}

	for name, opts := range invalid {
		_, err := ResolveFileEncryptionKey(opts)
		s.Error(err, name)
	}
}

func TestResolveFileEncryptionKeyTestSuite(t *testing.T) {
	suite.Run(t, new(ResolveFileEncryptionKeyTestSuite))
}

func (s *FileEncryptionTestSuite) Test_uses_the_region_in_an_aws_kms_key_arn() {
	s.T().Setenv("AWS_REGION", "us-east-1")
	signedRegions := make(chan string, 1)
	kms := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The credential scope in the signature names the region the client was built for.
		select {
		case signedRegions <- strings.Split(r.Header.Get("Authorization"), "/")[2]:
		default:
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer kms.Close()

	config := s.seededSource(StateEncryptionConfig{})
	_, err := Export(ExportParams{
		FilePath:     "/export.json",
		EngineConfig: config,
		FileSystem:   s.fs,
		FileEncryption: &StateEncryptionKeyConfig{
			Provider: "awskms",
			AWSKMS: StateEncryptionAWSKMSConfig{
				KeyID:           "arn:aws:kms:eu-west-2:123456789012:alias/state-export",
				Endpoint:        kms.URL,
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		},
	})
	s.Require().Error(err, "the fake KMS endpoint refuses every request")

	s.Equal("eu-west-2", <-signedRegions)
}
