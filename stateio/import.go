package stateio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

// ImportParams contains the parameters for an import operation.
type ImportParams struct {
	// FilePath is the path to the input file (local or remote URL).
	FilePath string
	// EngineConfig contains the deploy engine configuration.
	// Used to determine the storage backend (memfile or postgres).
	EngineConfig *EngineConfig
	// FileSystem is the filesystem to use for operations.
	FileSystem afero.Fs
	// Logger is the logger to use for logging.
	Logger core.Logger
	// FileData contains the raw file data to import.
	// If provided, FilePath is ignored for reading (but may be used for logging).
	FileData []byte
	// RemoteOptions contains options for downloading from remote storage.
	RemoteOptions *RemoteDownloadOptions
	// Importer is an optional StateImporter for import.
	// If not provided, a default importer will be created based on EngineConfig.
	Importer StateImporter
	// FileEncryption is the key to decrypt the input file with when it is encrypted.
	// Encrypted files are detected, so this is only needed for those.
	FileEncryption *StateEncryptionKeyConfig
	// RequireEncryptedFile refuses an input file that is not encrypted. Set it when a key
	// was given for this import in particular, so that a file someone replaced with a
	// plaintext one is not imported as if it had been decrypted. A key that applies to
	// every import, such as a passphrase in a pipeline's environment, should not set it.
	RequireEncryptedFile bool
	// Force allows importing over instances that have an operation in progress.
	// See WithForce. It does not apply to a custom Importer, which is created with its
	// own options.
	Force bool
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Success        bool   `json:"success"`
	InstancesCount int    `json:"instancesCount,omitempty"`
	Message        string `json:"message"`
	// Encrypted is true when the input file was an encrypted export file.
	Encrypted bool `json:"encrypted,omitempty"`
	// FileExportedAt is when an encrypted input file was exported. It is bound to the
	// file's encryption, so it can be relied on to tell an older export from the one
	// expected. It is zero for a plaintext file.
	FileExportedAt time.Time `json:"fileExportedAt,omitzero"`
}

// Import performs a state import operation based on the provided parameters.
// The input file must be a JSON array of blueprint instances.
func Import(params ImportParams) (*ImportResult, error) {
	if params.FileSystem == nil {
		params.FileSystem = afero.NewOsFs()
	}

	fileData, err := getInputData(params)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	ctx := context.Background()
	file, err := openExportFile(ctx, params.FileEncryption, params.RequireEncryptedFile, fileData)
	if err != nil {
		return nil, err
	}

	// Parsed before the state is opened, since opening encrypted state for writing creates
	// its encryption salt, and input that cannot be imported should leave the store alone.
	instances, err := ParseInstancesJSON(file.data)
	if err != nil {
		return nil, err
	}

	importer := params.Importer
	discardSetup := func() {}
	if importer == nil {
		backend, err := openDefaultBackend(
			ctx, params.EngineConfig, params.FileSystem, params.Logger, backendAccessWrite,
		)
		if err != nil {
			return nil, err
		}
		defer backend.close()

		importer = NewContainerStateImporter(backend.container, WithForce(params.Force))
		discardSetup = backend.discardSetup
	}

	if err := importer.ImportInstances(ctx, instances); err != nil {
		// An import refused by its checks, or that could not read the state it checks
		// against, has not written anything, so whatever opening the state set up, such as a
		// new encryption salt, is not needed. A failure part way through saving keeps it,
		// since the state saved so far depends on it.
		if failedBeforeWriting(err) {
			discardSetup()
		}
		return nil, err
	}

	count := countInstances(instances)
	if count == 0 {
		discardSetup()
	}
	return &ImportResult{
		Success:        true,
		InstancesCount: count,
		Message:        fmt.Sprintf("Successfully imported %d instances", count),
		Encrypted:      file.encrypted,
		FileExportedAt: file.exportedAt,
	}, nil
}

func getInputData(params ImportParams) ([]byte, error) {
	if params.FileData != nil {
		return params.FileData, nil
	}

	if IsRemoteFile(params.FilePath) {
		return DownloadRemoteFile(context.Background(), params.FilePath, params.RemoteOptions)
	}

	return os.ReadFile(params.FilePath)
}

func failedBeforeWriting(err error) bool {
	if _, refused := errors.AsType[*ImportError](err); refused {
		return true
	}
	_, unreadable := errors.AsType[*StateAccessError](err)
	return unreadable
}
