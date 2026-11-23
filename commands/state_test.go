package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type StateCommandSuite struct {
	suite.Suite
	tempDir string
}

func (s *StateCommandSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
	// The commands read the deploy engine's and the CLI's environment variables, so any
	// set in the shell running the tests are cleared; an empty value counts as unset.
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "BLUELINK_DEPLOY_ENGINE_") || strings.HasPrefix(name, "TESTCLI_") {
			s.T().Setenv(name, "")
		}
	}
}

func (s *StateCommandSuite) runStateCommand(args ...string) error {
	return s.runStateCommandWithProvider(config.NewProvider(), args...)
}

func (s *StateCommandSuite) runStateCommandWithProvider(confProvider *config.Provider, args ...string) error {
	rootCmd := &cobra.Command{Use: "testcli"}
	SetupStateCommand(rootCmd, confProvider, &CLIConfig{
		CLIName:      "testcli",
		EnvVarPrefix: "TESTCLI",
		Palette:      stylespkg.NewBluelinkPalette(),
	})
	rootCmd.SetArgs(append([]string{"state"}, args...))
	rootCmd.SilenceUsage = true
	return rootCmd.Execute()
}

func (s *StateCommandSuite) engineConfigFile(name string) (string, *stateio.EngineConfig) {
	stateDir := filepath.Join(s.tempDir, name, "state")
	s.Require().NoError(os.MkdirAll(stateDir, 0700))
	engineConfig := &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine:   stateio.StorageEngineMemfile,
			MemFileStateDir: stateDir,
		},
	}

	data, err := json.Marshal(map[string]any{
		"state": map[string]any{"storage_engine": "memfile", "memfile_state_dir": stateDir},
	})
	s.Require().NoError(err)
	path := filepath.Join(s.tempDir, name, "engine-config.json")
	s.Require().NoError(os.WriteFile(path, data, 0600))
	return path, engineConfig
}

func (s *StateCommandSuite) Test_rejects_a_key_on_an_export_without_encrypt() {
	err := s.runStateCommand("export", "--file", filepath.Join(s.tempDir, "out.json"), "--key", "localkey:/k")

	s.Require().Error(err)
	s.Equal("--key and --passphrase-file only apply to an export with --encrypt", err.Error())
}

func (s *StateCommandSuite) Test_rejects_an_encrypt_setting_that_is_not_a_boolean() {
	s.T().Setenv("TESTCLI_STATE_FILE_PASSPHRASE", "correct horse")
	s.T().Setenv("TESTCLI_STATE_EXPORT_ENCRYPT", "yes")
	exportPath := filepath.Join(s.tempDir, "out.json")
	// An engine config of the test's own, so a regression that went ahead with the export
	// would read the test's empty state rather than the engine config found on this machine.
	engineConfigPath, _ := s.engineConfigFile("source")

	err := s.runStateCommand("--engine-config-file", engineConfigPath, "export", "--file", exportPath)

	s.Require().Error(err)
	s.Equal(`--encrypt must be true or false, got "yes"`, err.Error())
	_, statErr := os.Stat(exportPath)
	s.True(os.IsNotExist(statErr), "expected no file to be written, got %v", statErr)
}

func (s *StateCommandSuite) Test_requires_a_key_for_an_encrypted_export() {
	err := s.runStateCommand("export", "--file", filepath.Join(s.tempDir, "out.json"), "--encrypt")

	s.Require().Error(err)
	s.Equal(
		"--encrypt needs a key: provide --key, --passphrase-file or set TESTCLI_STATE_FILE_PASSPHRASE",
		err.Error(),
	)
}

func (s *StateCommandSuite) Test_rejects_both_a_key_and_a_passphrase_file() {
	err := s.runStateCommand(
		"import", "--file", filepath.Join(s.tempDir, "in.json"),
		"--key", "localkey:/k", "--passphrase-file", "/p",
	)

	s.Require().Error(err)
	s.Equal("provide either a key or a passphrase file for the state file, not both", err.Error())
}

func (s *StateCommandSuite) Test_round_trips_an_encrypted_file_through_the_cli() {
	sourceConfigPath, sourceConfig := s.engineConfigFile("source")
	targetConfigPath, targetConfig := s.engineConfigFile("target")
	_, err := stateio.Import(stateio.ImportParams{
		EngineConfig: sourceConfig,
		FileData:     []byte(`[{"id":"inst-1","name":"orders","status":2}]`),
	})
	s.Require().NoError(err)

	exportPath := filepath.Join(s.tempDir, "state.json")
	s.T().Setenv("TESTCLI_STATE_FILE_PASSPHRASE", "correct horse")
	err = s.runStateCommand("--engine-config-file", sourceConfigPath, "export", "--file", exportPath, "--encrypt")
	s.Require().NoError(err)

	contents, err := os.ReadFile(exportPath)
	s.Require().NoError(err)
	s.NotContains(string(contents), "orders")

	passphrasePath := filepath.Join(s.tempDir, "passphrase.txt")
	s.Require().NoError(os.WriteFile(passphrasePath, []byte("correct horse\n"), 0600))
	s.T().Setenv("TESTCLI_STATE_FILE_PASSPHRASE", "")
	err = s.runStateCommand(
		"--engine-config-file", targetConfigPath,
		"import", "--file", exportPath, "--passphrase-file", passphrasePath,
	)
	s.Require().NoError(err)

	verifyPath := filepath.Join(s.tempDir, "verify.json")
	_, err = stateio.Export(stateio.ExportParams{
		FilePath:     verifyPath,
		EngineConfig: targetConfig,
		FileSystem:   afero.NewOsFs(),
	})
	s.Require().NoError(err)
	verified, err := os.ReadFile(verifyPath)
	s.Require().NoError(err)
	s.Contains(string(verified), "orders")
}

func (s *StateCommandSuite) Test_uses_the_engine_environment_when_there_is_no_engine_config_file() {
	stateDir := filepath.Join(s.tempDir, "state")
	s.Require().NoError(os.MkdirAll(stateDir, 0700))
	s.T().Setenv("HOME", s.tempDir)
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_MEMFILE_STATE_DIR", stateDir)
	inputPath := filepath.Join(s.tempDir, "input.json")
	s.Require().NoError(os.WriteFile(inputPath, []byte(`[{"id":"inst-1","name":"orders","status":2}]`), 0600))

	s.Require().NoError(s.runStateCommand("import", "--file", inputPath))

	_, err := stateio.Export(stateio.ExportParams{
		FilePath: filepath.Join(s.tempDir, "verify.json"),
		EngineConfig: &stateio.EngineConfig{
			State: stateio.StateConfig{StorageEngine: stateio.StorageEngineMemfile, MemFileStateDir: stateDir},
		},
		FileSystem: afero.NewOsFs(),
	})
	s.Require().NoError(err)
	verified, err := os.ReadFile(filepath.Join(s.tempDir, "verify.json"))
	s.Require().NoError(err)
	s.Contains(string(verified), "orders")
}

func (s *StateCommandSuite) Test_requires_an_engine_config_file_that_is_named_explicitly() {
	missingPath := filepath.Join(s.tempDir, "missing.json")
	err := s.runStateCommand("--engine-config-file", missingPath, "export", "--file", "out.json")

	s.Require().Error(err)
	s.Equal(
		"failed to load engine config: state_config_invalid: failed to read engine config file: "+
			"open "+missingPath+": no such file or directory",
		err.Error(),
	)
}

func (s *StateCommandSuite) Test_refuses_a_plaintext_file_given_a_passphrase_file() {
	configPath, _ := s.engineConfigFile("target")
	inputPath := filepath.Join(s.tempDir, "input.json")
	s.Require().NoError(os.WriteFile(inputPath, []byte(`[{"id":"inst-1","name":"orders","status":2}]`), 0600))
	passphrasePath := filepath.Join(s.tempDir, "passphrase.txt")
	s.Require().NoError(os.WriteFile(passphrasePath, []byte("correct horse\n"), 0600))

	err := s.runStateCommand(
		"--engine-config-file", configPath, "import", "--file", inputPath, "--passphrase-file", passphrasePath,
	)
	s.Require().Error(err)

	// A passphrase in the environment may be set for every import, so it allows plaintext.
	s.T().Setenv("TESTCLI_STATE_FILE_PASSPHRASE", "correct horse")
	s.Require().NoError(s.runStateCommand("--engine-config-file", configPath, "import", "--file", inputPath))
}

func (s *StateCommandSuite) Test_force_imports_over_an_instance_with_an_operation_in_progress() {
	configPath, engineConfig := s.engineConfigFile("target")
	_, err := stateio.Import(stateio.ImportParams{
		EngineConfig: engineConfig,
		FileData:     []byte(`[{"id":"inst-1","name":"orders","status":1}]`),
	})
	s.Require().NoError(err)

	inputPath := filepath.Join(s.tempDir, "input.json")
	s.Require().NoError(os.WriteFile(inputPath, []byte(`[{"id":"inst-1","name":"orders","status":2}]`), 0600))

	err = s.runStateCommand("--engine-config-file", configPath, "import", "--file", inputPath)
	s.Require().Error(err, "expected the import to be refused without --force")

	err = s.runStateCommand("--engine-config-file", configPath, "import", "--file", inputPath, "--force")
	s.Require().NoError(err)
}

func (s *StateCommandSuite) Test_ignores_force_and_key_settings_in_the_cli_config_file() {
	configPath, engineConfig := s.engineConfigFile("target")
	_, err := stateio.Import(stateio.ImportParams{
		EngineConfig: engineConfig,
		FileData:     []byte(`[{"id":"inst-1","name":"orders","status":1}]`),
	})
	s.Require().NoError(err)

	inputPath := filepath.Join(s.tempDir, "input.json")
	s.Require().NoError(os.WriteFile(inputPath, []byte(`[{"id":"inst-1","name":"orders","status":2}]`), 0600))

	cliConfigPath := filepath.Join(s.tempDir, "testcli.json")
	s.Require().NoError(os.WriteFile(cliConfigPath, []byte(`{
		"stateImportForce": "true",
		"stateImportPassphrase": "from the config file",
		"stateImportPassphraseFile": "missing-passphrase.txt"
	}`), 0600))
	confProvider := config.NewProvider()
	s.Require().NoError(confProvider.LoadConfigFile(cliConfigPath))

	// A plaintext file imports, so the passphrase file in the config file was not applied.
	newInstancePath := filepath.Join(s.tempDir, "new-instance.json")
	s.Require().NoError(os.WriteFile(newInstancePath, []byte(`[{"id":"inst-2","name":"billing","status":2}]`), 0600))
	err = s.runStateCommandWithProvider(
		confProvider, "--engine-config-file", configPath, "import", "--file", newInstancePath,
	)
	s.Require().NoError(err)

	err = s.runStateCommandWithProvider(confProvider, "--engine-config-file", configPath, "import", "--file", inputPath)
	s.Require().Error(err, "expected the import to be refused, since --force in the config file is not applied")
}

func TestStateCommandSuite(t *testing.T) {
	suite.Run(t, new(StateCommandSuite))
}
