package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

type OperationConfigSuite struct {
	suite.Suite
	tempDir string
}

func (s *OperationConfigSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
}

// Binds deployConfigFile the way a CLI does: a flag whose default is a path relative to the
// working directory, here placed in the test's temp directory.
func (s *OperationConfigSuite) providerWithDeployConfigFlag(args ...string) *Provider {
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("deploy-config-file", filepath.Join(s.tempDir, "bluelink.deploy.json"), "")
	s.Require().NoError(flagSet.Parse(args))

	p := NewProvider()
	p.BindPFlag("deployConfigFile", flagSet.Lookup("deploy-config-file"))
	return p
}

func (s *OperationConfigSuite) Test_a_missing_file_at_the_default_path_loads_no_config() {
	opConfig, err := LoadOperationConfig(s.providerWithDeployConfigFlag())
	s.NoError(err)
	s.Nil(opConfig)
}

func (s *OperationConfigSuite) Test_a_file_at_the_default_path_is_loaded() {
	path := filepath.Join(s.tempDir, "bluelink.deploy.json")
	s.Require().NoError(os.WriteFile(path, []byte(`{"dependencies": {"aws": "1.0.0"}}`), 0o600))

	opConfig, err := LoadOperationConfig(s.providerWithDeployConfigFlag())
	s.Require().NoError(err)
	s.Require().NotNil(opConfig)
	s.Equal(map[string]string{"aws": "1.0.0"}, opConfig.Dependencies)
}

func (s *OperationConfigSuite) Test_a_missing_file_at_a_path_set_by_the_user_is_an_error() {
	path := filepath.Join(s.tempDir, "missing.deploy.json")

	_, err := LoadOperationConfig(s.providerWithDeployConfigFlag("--deploy-config-file=" + path))
	s.ErrorIs(err, os.ErrNotExist)
}

func (s *OperationConfigSuite) Test_a_missing_file_at_a_path_set_by_env_var_is_an_error() {
	path := filepath.Join(s.tempDir, "missing.deploy.json")
	s.T().Setenv("TEST_DEPLOY_CONFIG_FILE", path)
	p := s.providerWithDeployConfigFlag()
	p.BindEnvVar("deployConfigFile", "TEST_DEPLOY_CONFIG_FILE")

	_, err := LoadOperationConfig(p)
	s.ErrorIs(err, os.ErrNotExist)
}

// A pre-command step (as in the Celerity CLI) sets the path to the file it generated, so a
// missing file there means the step failed.
func (s *OperationConfigSuite) Test_a_missing_file_at_a_path_set_by_a_pre_command_step_is_an_error() {
	p := s.providerWithDeployConfigFlag()
	p.Set("deployConfigFile", filepath.Join(s.tempDir, "generated.deploy.json"))

	_, err := LoadOperationConfig(p)
	s.ErrorIs(err, os.ErrNotExist)
}

func TestOperationConfigSuite(t *testing.T) {
	suite.Run(t, new(OperationConfigSuite))
}
