package stateio

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/spf13/afero"
)

// The salt is read through the undecorated storage since it is an input to deriving
// the key that would decrypt it, and before the container is loaded since loading
// reads the whole store.
//
// The returned function removes the salt again if this call created it, for an import
// that fails before writing any state, so that the store is left as it was found.
func memfileStorage(
	ctx context.Context,
	config *StateConfig,
	fileSystem afero.Fs,
	access backendAccess,
) (statestore.Storage, func(), error) {
	storage := memfile.NewFSStorage(fileSystem)
	if !config.Encryption.Enabled {
		return storage, func() {}, nil
	}

	keys := statestore.NewKeyBuilder(config.MemFileStateDir)
	salt, created, err := loadEncryptionSalt(ctx, storage, keys, access)
	if err != nil {
		return nil, nil, &StateAccessError{
			Code:    ErrCodeEncryptionKeyUnavailable,
			Message: "the state encryption salt could not be loaded",
			Err:     err,
		}
	}
	discardCreatedSalt := func() {
		if created {
			_ = storage.Delete(ctx, keys.EncryptionSalt())
		}
	}

	keyProvider, err := createStateKeyProvider(ctx, &config.Encryption, salt, keys.EncryptionSalt())
	if err != nil {
		discardCreatedSalt()
		return nil, nil, &StateAccessError{
			Code:    ErrCodeEncryptionKeyUnavailable,
			Message: "the state encryption key could not be set up",
			Hint:    stateConfigHint("encryption"),
			Err:     err,
		}
	}

	return statestore.NewEncryptedStorage(storage, keyProvider), discardCreatedSalt, nil
}

// An import creates the salt on first use like the deploy engine does, so that state
// it writes into an empty directory can be read by an engine with the same passphrase.
// An export must not write to the store, so a missing salt is returned as nil and left
// for a key provider that needs one to report. Whether the salt was created is returned
// alongside it.
func loadEncryptionSalt(
	ctx context.Context,
	storage statestore.Storage,
	keys statestore.KeyBuilder,
	access backendAccess,
) ([]byte, bool, error) {
	exists, err := storage.Exists(ctx, keys.EncryptionSalt())
	if err != nil {
		return nil, false, fmt.Errorf("checking for an encryption salt at %q: %w", keys.EncryptionSalt(), err)
	}

	if !exists && access == backendAccessRead {
		return nil, false, nil
	}

	salt, err := statestore.LoadOrCreateSalt(ctx, storage, keys)
	return salt, err == nil && !exists, err
}
