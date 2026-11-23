package stateio

import (
	"fmt"
	"net/url"
	"strings"
)

// FileEncryptionKeyOptions holds the ways a CLI can name the key an export file is
// encrypted or decrypted with.
type FileEncryptionKeyOptions struct {
	// Key names a key held outside the CLI as "<provider>:<id>", for example
	// "awskms:alias/state-export", "gcpkms:projects/p/locations/l/keyRings/r/cryptoKeys/k",
	// "azurekeyvault:https://vault.vault.azure.net/keys/name" or "localkey:/path/to/key".
	Key string
	// PassphraseFile is the path to a file holding the passphrase.
	PassphraseFile string
	// Passphrase is a passphrase taken from the environment. It is never accepted as a
	// command line flag value, which would leave it in shell history and CI logs.
	Passphrase string
}

var keyIDProviders = []string{
	localKeyFileProvider,
	awsKMSKeyProvider,
	gcpKMSKeyProvider,
	azureKeyVaultKeyProvider,
}

// ResolveFileEncryptionKey returns the key configuration an export file is encrypted
// or decrypted with, or nil when no key was given.
//
// A key or passphrase file given explicitly takes precedence over a passphrase from the
// environment, which may be set for every job in a pipeline. Giving both a key and a
// passphrase file is an error, since it is unclear which was meant.
func ResolveFileEncryptionKey(opts FileEncryptionKeyOptions) (*StateEncryptionKeyConfig, error) {
	if opts.Key != "" && opts.PassphraseFile != "" {
		return nil, fmt.Errorf("provide either a key or a passphrase file for the state file, not both")
	}

	if opts.Key != "" {
		return parseFileEncryptionKey(opts.Key)
	}

	if opts.PassphraseFile != "" {
		return &StateEncryptionKeyConfig{
			Provider:       passphraseKeyProvider,
			PassphraseFile: opts.PassphraseFile,
		}, nil
	}

	if opts.Passphrase != "" {
		return &StateEncryptionKeyConfig{
			Provider:   passphraseKeyProvider,
			Passphrase: opts.Passphrase,
		}, nil
	}

	return nil, nil
}

func parseFileEncryptionKey(key string) (*StateEncryptionKeyConfig, error) {
	// Split on the first separator only, since AWS key ARNs and Azure key URLs
	// contain colons of their own.
	provider, id, hasSeparator := strings.Cut(key, ":")
	if !hasSeparator || id == "" {
		// The value is left out of the error, since a passphrase given as a key by mistake
		// would otherwise end up in logs.
		return nil, fmt.Errorf("the key must be in the form <provider>:<id>")
	}

	switch provider {
	case localKeyFileProvider:
		return &StateEncryptionKeyConfig{Provider: provider, LocalKeyFile: id}, nil
	case awsKMSKeyProvider:
		return &StateEncryptionKeyConfig{
			Provider: provider,
			AWSKMS:   StateEncryptionAWSKMSConfig{KeyID: id},
		}, nil
	case gcpKMSKeyProvider:
		return &StateEncryptionKeyConfig{
			Provider: provider,
			GCPKMS:   StateEncryptionGCPKMSConfig{KeyName: id},
		}, nil
	case azureKeyVaultKeyProvider:
		return parseAzureKeyVaultKey(id)
	case passphraseKeyProvider:
		return nil, fmt.Errorf(
			"a passphrase cannot be given as a key, use a passphrase file or the passphrase environment variable",
		)
	default:
		return nil, fmt.Errorf(
			"the key's provider is not supported, supported providers are: %s",
			strings.Join(keyIDProviders, ", "),
		)
	}
}

// Takes the key identifier Azure Key Vault itself hands out,
// https://<vault>.vault.azure.net/keys/<name>[/<version>], rather than inventing a format.
// The value is left out of the errors for the same reason as in parseFileEncryptionKey.
func parseAzureKeyVaultKey(keyURL string) (*StateEncryptionKeyConfig, error) {
	parsed, err := url.Parse(keyURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf(
			"an azure key vault key must be a key identifier URL of the form " +
				"https://<vault>.vault.azure.net/keys/<name>[/<version>]",
		)
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 || len(segments) > 3 || segments[0] != "keys" || segments[1] == "" {
		return nil, fmt.Errorf("an azure key vault key must name a key under /keys/<name>[/<version>]")
	}

	keyConfig := &StateEncryptionKeyConfig{
		Provider: azureKeyVaultKeyProvider,
		AzureKeyVault: StateEncryptionAzureKeyVaultConfig{
			VaultURL: fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host),
			KeyName:  segments[1],
		},
	}
	if len(segments) == 3 {
		keyConfig.AzureKeyVault.KeyVersion = segments[2]
	}

	return keyConfig, nil
}
