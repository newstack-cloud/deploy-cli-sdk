package stateio

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/awskms"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/azurekeyvault"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/keyproviders/gcpkms"
	"github.com/newstack-cloud/bluelink/libs/common/crypto"
	"google.golang.org/api/option"
)

// The key provider set up in this file mirrors the deploy engine's, which lives in an
// internal package of the engine and cannot be imported. It must stay in step with the
// engine so that state the engine encrypts can be read here, and the reverse. The one
// difference is that an AWS KMS key ARN's region is used when no region is configured,
// where the engine uses the region from the AWS environment.

const (
	passphraseKeyProvider    = "passphrase"
	localKeyFileProvider     = "localkey"
	awsKMSKeyProvider        = "awskms"
	gcpKMSKeyProvider        = "gcpkms"
	azureKeyVaultKeyProvider = "azurekeyvault"
)

var supportedStateKeyProviders = []string{
	passphraseKeyProvider,
	localKeyFileProvider,
	awsKMSKeyProvider,
	gcpKMSKeyProvider,
	azureKeyVaultKeyProvider,
}

// Builds a key set when previous keys are configured so objects sealed with a key the
// deployment has since moved off remain readable, which is also the state a store is in
// part-way through re-encryption.
func createStateKeyProvider(
	ctx context.Context,
	encryptionConfig *StateEncryptionConfig,
	salt []byte,
	saltKey string,
) (crypto.KeyProvider, error) {
	active, err := createKeyProviderForKey(ctx, &encryptionConfig.StateEncryptionKeyConfig, salt, saltKey)
	if err != nil {
		return nil, err
	}

	if len(encryptionConfig.PreviousKeys) == 0 {
		return active, nil
	}

	previous := make([]crypto.KeyProvider, 0, len(encryptionConfig.PreviousKeys))
	for index := range encryptionConfig.PreviousKeys {
		provider, err := createKeyProviderForKey(ctx, &encryptionConfig.PreviousKeys[index], salt, saltKey)
		if err != nil {
			return nil, fmt.Errorf("state.encryption.previous_keys[%d]: %w", index, err)
		}

		previous = append(previous, provider)
	}

	return crypto.NewKeySet(active, previous...)
}

func createKeyProviderForKey(
	ctx context.Context,
	keyConfig *StateEncryptionKeyConfig,
	salt []byte,
	saltKey string,
) (crypto.KeyProvider, error) {
	switch keyConfig.Provider {
	case passphraseKeyProvider:
		return createPassphraseKeyProvider(keyConfig, salt, saltKey)
	case localKeyFileProvider:
		if keyConfig.LocalKeyFile == "" {
			return nil, requiredForProviderError("state.encryption.local_key_file", localKeyFileProvider)
		}
		return crypto.NewLocalFileKeyProvider(keyConfig.LocalKeyFile)
	case awsKMSKeyProvider:
		return createAWSKMSKeyProvider(ctx, &keyConfig.AWSKMS)
	case gcpKMSKeyProvider:
		return createGCPKMSKeyProvider(ctx, &keyConfig.GCPKMS)
	case azureKeyVaultKeyProvider:
		return createAzureKeyVaultKeyProvider(&keyConfig.AzureKeyVault)
	default:
		return nil, fmt.Errorf(
			"unsupported state encryption key provider %q, supported providers are: %s",
			keyConfig.Provider,
			strings.Join(supportedStateKeyProviders, ", "),
		)
	}
}

func createPassphraseKeyProvider(
	keyConfig *StateEncryptionKeyConfig,
	salt []byte,
	saltKey string,
) (crypto.KeyProvider, error) {
	passphrase, err := statePassphrase(keyConfig)
	if err != nil {
		return nil, err
	}

	// Only reachable on export, which does not create a salt. Without the salt the
	// passphrase cannot derive the key, so state was either never encrypted with a
	// passphrase or the salt has been lost.
	if salt == nil {
		return nil, fmt.Errorf(
			"no encryption salt was found at %q, so the passphrase cannot derive the key "+
				"the state was encrypted with; check that state.memfile_state_dir points at "+
				"state written with encryption enabled",
			saltKey,
		)
	}

	return crypto.NewPassphraseKeyProvider(passphrase, salt)
}

func statePassphrase(keyConfig *StateEncryptionKeyConfig) (string, error) {
	if keyConfig.PassphraseFile != "" {
		contents, err := os.ReadFile(keyConfig.PassphraseFile)
		if err != nil {
			return "", fmt.Errorf("failed to read state.encryption.passphrase_file: %w", err)
		}

		// Trailing whitespace is stripped, as the deploy engine does, so a passphrase
		// written by an editor or a shell redirect derives the same key.
		passphrase := strings.TrimRight(string(contents), " \t\r\n")
		if passphrase == "" {
			return "", fmt.Errorf("state.encryption.passphrase_file %q is empty", keyConfig.PassphraseFile)
		}

		return passphrase, nil
	}

	if keyConfig.Passphrase == "" {
		return "", fmt.Errorf(
			"one of state.encryption.passphrase or state.encryption.passphrase_file "+
				"is required for the %q key provider",
			passphraseKeyProvider,
		)
	}

	return keyConfig.Passphrase, nil
}

func createAWSKMSKeyProvider(
	ctx context.Context,
	kmsConfig *StateEncryptionAWSKMSConfig,
) (crypto.KeyProvider, error) {
	if kmsConfig.KeyID == "" {
		return nil, requiredForProviderError("state.encryption.awskms.key_id", awsKMSKeyProvider)
	}

	conf, err := loadAWSConfig(ctx, awsKMSKeyRegion(kmsConfig), kmsConfig.AccessKeyID, kmsConfig.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config for state encryption: %w", err)
	}

	client := awskms.NewClient(conf, awskms.ClientOptions{Endpoint: kmsConfig.Endpoint})
	return awskms.NewKeyProvider(client, kmsConfig.KeyID)
}

// KMS only finds a key through a client in the key's own region, so without a region set
// the one in a key ARN is used, ahead of the region from the AWS environment. A key ID or
// alias name has no region, and the environment's is used.
func awsKMSKeyRegion(kmsConfig *StateEncryptionAWSKMSConfig) string {
	if kmsConfig.Region != "" {
		return kmsConfig.Region
	}

	keyARN, err := arn.Parse(kmsConfig.KeyID)
	if err != nil || keyARN.Service != "kms" {
		return ""
	}
	return keyARN.Region
}

func createGCPKMSKeyProvider(
	ctx context.Context,
	kmsConfig *StateEncryptionGCPKMSConfig,
) (crypto.KeyProvider, error) {
	if kmsConfig.KeyName == "" {
		return nil, requiredForProviderError("state.encryption.gcpkms.key_name", gcpKMSKeyProvider)
	}

	clientOpts := []option.ClientOption{}
	if kmsConfig.CredentialsFile != "" {
		// The credential type is named rather than inferred from the file, so a file holding
		// another kind of credential configuration is refused instead of being loaded.
		clientOpts = append(clientOpts, option.WithAuthCredentialsFile(
			option.ServiceAccount,
			kmsConfig.CredentialsFile,
		))
	}

	client, err := gcpkms.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud kms client: %w", err)
	}

	return gcpkms.NewKeyProvider(client, kmsConfig.KeyName)
}

func createAzureKeyVaultKeyProvider(
	vaultConfig *StateEncryptionAzureKeyVaultConfig,
) (crypto.KeyProvider, error) {
	if vaultConfig.VaultURL == "" {
		return nil, requiredForProviderError("state.encryption.azurekeyvault.vault_url", azureKeyVaultKeyProvider)
	}
	if vaultConfig.KeyName == "" {
		return nil, requiredForProviderError("state.encryption.azurekeyvault.key_name", azureKeyVaultKeyProvider)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve azure credentials for state encryption: %w", err)
	}

	client, err := azurekeyvault.NewClient(vaultConfig.VaultURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure key vault client: %w", err)
	}

	return azurekeyvault.NewKeyProvider(client, vaultConfig.KeyName, azurekeyvault.Options{
		KeyVersion: vaultConfig.KeyVersion,
		Algorithm:  azkeys.EncryptionAlgorithm(vaultConfig.Algorithm),
	})
}

func requiredForProviderError(configKey string, keyProvider string) error {
	return fmt.Errorf("%s is required for the %q state encryption key provider", configKey, keyProvider)
}
