package stateio

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

// Whether a backend is opened to read state for an export or to write state for an
// import. An export must leave the store exactly as it found it.
type backendAccess int

const (
	backendAccessRead backendAccess = iota
	backendAccessWrite
)

type stateBackend struct {
	container state.Container
	encrypted bool
	close     func()
	// Undoes anything opening the backend wrote, for an import that then writes nothing.
	discardSetup func()
}

// Opens the backend described by the engine config, for an import or export that was
// not given its own importer or exporter.
func openDefaultBackend(
	ctx context.Context,
	engineConfig *EngineConfig,
	fileSystem afero.Fs,
	logger core.Logger,
	access backendAccess,
) (*stateBackend, error) {
	if engineConfig == nil {
		return nil, fmt.Errorf("engine config is required for %s", accessOperationName(access))
	}

	if logger == nil {
		logger = core.NewNopLogger()
	}

	return openStateBackend(ctx, &engineConfig.State, fileSystem, logger, access)
}

func accessOperationName(access backendAccess) string {
	if access == backendAccessWrite {
		return "import"
	}
	return "export"
}

func openStateBackend(
	ctx context.Context,
	config *StateConfig,
	fileSystem afero.Fs,
	logger core.Logger,
	access backendAccess,
) (*stateBackend, error) {
	if err := checkEncryptionSupported(config); err != nil {
		return nil, err
	}

	switch config.StorageEngine {
	case StorageEnginePostgres:
		return openPostgresBackend(ctx, config, logger)
	case StorageEngineMemfile, "":
		return openMemfileBackend(ctx, config, fileSystem, logger, access)
	case StorageEngineObjectstore:
		return openObjectStoreBackend(ctx, config, logger)
	default:
		return nil, &StateAccessError{
			Code: ErrCodeStateConfigInvalid,
			Message: fmt.Sprintf(
				"unsupported storage engine %q, supported engines are: \"memfile\", \"postgres\", \"objectstore\"",
				config.StorageEngine,
			),
			Hint: stateConfigHint("storage_engine"),
		}
	}
}

// Mirrors the deploy engine, which only encrypts memfile state and relies on
// encryption at rest in the database or bucket for the other storage engines.
func checkEncryptionSupported(config *StateConfig) error {
	isMemfile := config.StorageEngine == StorageEngineMemfile || config.StorageEngine == ""
	if !config.Encryption.Enabled || isMemfile {
		return nil
	}

	return &StateAccessError{
		Code: ErrCodeStateConfigInvalid,
		Message: fmt.Sprintf(
			"state encryption is only supported for the %q storage engine, "+
				"the %q engine relies on encryption at rest in the database or bucket "+
				"it persists to",
			StorageEngineMemfile,
			config.StorageEngine,
		),
		Hint: stateConfigHint("encryption"),
	}
}

func openMemfileBackend(
	ctx context.Context,
	config *StateConfig,
	fileSystem afero.Fs,
	logger core.Logger,
	access backendAccess,
) (*stateBackend, error) {
	storage, discardCreatedSalt, err := memfileStorage(ctx, config, fileSystem, access)
	if err != nil {
		return nil, err
	}

	container, err := memfile.LoadStateContainerWithStorage(
		config.MemFileStateDir,
		storage,
		logger,
	)
	if err != nil {
		discardCreatedSalt()
		return nil, wrapMemfileLoadError(config, err)
	}

	return &stateBackend{
		container:    container,
		encrypted:    config.Encryption.Enabled,
		close:        func() {},
		discardSetup: discardCreatedSalt,
	}, nil
}

// Loading reads and decrypts every object in the store, so with encryption enabled a
// failure here is almost always one to decrypt: a key that does not match the one state
// was written with, or state written before encryption was enabled. A failure to read the
// files at all is reported as it is.
func wrapMemfileLoadError(config *StateConfig, err error) error {
	if _, isFileError := errors.AsType[*fs.PathError](err); !config.Encryption.Enabled || isFileError {
		return &StateAccessError{
			Code:    ErrCodeStateUnavailable,
			Message: "the memfile state could not be loaded",
			Err:     err,
		}
	}

	return &StateAccessError{
		Code:    ErrCodeStateDecryptionFailed,
		Message: "the state could not be decrypted",
		Hint: "Check that the configured key is the one this state was encrypted with. " +
			"If the key has been rotated, list the old key under state.encryption.previous_keys. " +
			"If the state was written before encryption was enabled, encrypt it with the " +
			"deploy engine's reencrypt-state command.",
		Err: err,
	}
}

func openPostgresBackend(
	ctx context.Context,
	config *StateConfig,
	logger core.Logger,
) (*stateBackend, error) {
	pool, err := pgxpool.New(ctx, BuildPostgresDatabaseURL(config))
	if err != nil {
		return nil, &StateAccessError{
			Code:    ErrCodeStateConfigInvalid,
			Message: "the postgres connection could not be set up",
			Hint: "Check the state.postgres_* settings in the deploy engine config, and any " +
				engineStateEnvPrefix + "POSTGRES_* environment variables.",
			Err: err,
		}
	}

	container, err := postgres.LoadStateContainer(ctx, pool, logger)
	if err != nil {
		pool.Close()
		return nil, &StateAccessError{
			Code:    ErrCodeStateUnavailable,
			Message: "the postgres state could not be loaded",
			Err:     err,
		}
	}

	return &stateBackend{
		container:    container,
		close:        pool.Close,
		discardSetup: func() {},
	}, nil
}
