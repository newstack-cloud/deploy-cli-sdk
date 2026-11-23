package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const stateFilePassphraseEnvVarSuffix = "_STATE_FILE_PASSPHRASE"

// The key flags are read from the command line alone, never from an environment variable
// or the CLI's config file. A key set for every job in a pipeline would otherwise turn each
// plaintext export into a usage error, and a key in a project config file would apply to
// every import run in that project. The passphrase is the exception: it is only read from
// the environment, directly, so a config file cannot hold it.
func addStateFileKeyFlags(cmd *cobra.Command) {
	cmd.Flags().String(
		"key", "",
		"Key the state file is encrypted with, as <provider>:<id>. "+
			"Providers: awskms, gcpkms, azurekeyvault (a key identifier URL) and localkey (a key file path).",
	)

	cmd.Flags().String(
		"passphrase-file", "",
		"Path to a file holding the passphrase the state file is encrypted with.",
	)
}

func readStateFileKeyOptions(flags *pflag.FlagSet, envVarPrefix string) stateio.FileEncryptionKeyOptions {
	key, _ := flags.GetString("key")
	passphraseFile, _ := flags.GetString("passphrase-file")

	return stateio.FileEncryptionKeyOptions{
		Key:            strings.TrimSpace(key),
		PassphraseFile: strings.TrimSpace(passphraseFile),
		Passphrase:     os.Getenv(envVarPrefix + stateFilePassphraseEnvVarSuffix),
	}
}

// A passphrase in the environment is only used when --encrypt is set, so a pipeline can
// set one for every job and still produce plaintext exports where it wants them. An
// explicit key flag without --encrypt is a mistake that should be reported.
func resolveExportFileKey(
	encrypt bool,
	keyOptions stateio.FileEncryptionKeyOptions,
	envVarPrefix string,
) (*stateio.StateEncryptionKeyConfig, error) {
	if !encrypt {
		if keyOptions.Key != "" || keyOptions.PassphraseFile != "" {
			return nil, fmt.Errorf("--key and --passphrase-file only apply to an export with --encrypt")
		}
		return nil, nil
	}

	key, err := stateio.ResolveFileEncryptionKey(keyOptions)
	if err != nil {
		return nil, err
	}

	if key == nil {
		passphraseEnvVar := fmt.Sprintf("%s%s", envVarPrefix, stateFilePassphraseEnvVarSuffix)
		return nil, fmt.Errorf(
			"--encrypt needs a key: provide --key, --passphrase-file or set %s",
			passphraseEnvVar,
		)
	}

	return key, nil
}
