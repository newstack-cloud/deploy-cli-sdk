package stateio

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
)

// ExportParams contains the parameters for an export operation.
type ExportParams struct {
	// FilePath is the path to the output file (local or remote URL).
	FilePath string
	// InstanceFilters is a list of instance IDs or names to export.
	// If empty, all instances are exported.
	InstanceFilters []string
	// EngineConfig contains the deploy engine configuration.
	// Used to determine the storage backend (memfile or postgres).
	EngineConfig *EngineConfig
	// FileSystem is the filesystem to use for local file operations.
	FileSystem afero.Fs
	// Logger is the logger to use for logging.
	Logger core.Logger
	// RemoteOptions contains options for uploading to remote storage.
	RemoteOptions *RemoteUploadOptions
	// Exporter is an optional StateExporter for export.
	// If not provided, a default exporter will be created based on EngineConfig.
	Exporter StateExporter
	// FileEncryption is the key the output file is encrypted with.
	// When nil, the file is written as plaintext JSON.
	FileEncryption *StateEncryptionKeyConfig
}

// ExportResult contains the result of an export operation.
type ExportResult struct {
	Success        bool   `json:"success"`
	InstancesCount int    `json:"instancesCount,omitempty"`
	FilePath       string `json:"filePath,omitempty"`
	Message        string `json:"message"`
	// Encrypted is true when the output file was encrypted.
	Encrypted bool `json:"encrypted,omitempty"`
	// Warnings holds anything the caller should be told about the exported file,
	// such as it holding state in plaintext that is encrypted at rest.
	Warnings []string `json:"warnings,omitempty"`
}

// Export performs a state export operation based on the provided parameters.
// The output is a JSON array of blueprint instances.
func Export(params ExportParams) (*ExportResult, error) {
	if params.FileSystem == nil {
		params.FileSystem = afero.NewOsFs()
	}

	ctx := context.Background()
	exporter := params.Exporter
	var warnings []string
	if exporter == nil {
		backend, err := openDefaultBackend(
			ctx, params.EngineConfig, params.FileSystem, params.Logger, backendAccessRead,
		)
		if err != nil {
			return nil, err
		}
		defer backend.close()

		exporter = NewContainerStateExporter(backend.container)
		warnings = exportWarnings(backend, params.FileEncryption)
	}

	result, err := ExecuteInstancesExport(ctx, exporter, params.InstanceFilters)
	if err != nil {
		return nil, err
	}

	output, err := prepareOutputData(ctx, params.FileEncryption, result.Data)
	if err != nil {
		return nil, err
	}

	err = writeOutputData(params, output)
	if err != nil {
		return nil, err
	}

	return &ExportResult{
		Success:        true,
		InstancesCount: result.InstancesCount,
		FilePath:       params.FilePath,
		Message:        fmt.Sprintf("Successfully exported %d instances to %s", result.InstancesCount, params.FilePath),
		Encrypted:      params.FileEncryption != nil,
		Warnings:       warnings,
	}, nil
}

func prepareOutputData(
	ctx context.Context,
	fileEncryption *StateEncryptionKeyConfig,
	data []byte,
) ([]byte, error) {
	if fileEncryption == nil {
		return data, nil
	}

	sealed, err := sealExportFile(ctx, fileEncryption, data, time.Now())
	if err != nil {
		return nil, &ExportError{
			Code:    ErrCodeFileEncryptionFailed,
			Message: "the output file could not be encrypted",
			Err:     err,
		}
	}

	return sealed, nil
}

// Resource specs, changesets and link data can hold resolved secrets, so taking state
// out of an encrypted store into a plaintext file is called out rather than left for
// someone to notice later.
func exportWarnings(backend *stateBackend, fileEncryption *StateEncryptionKeyConfig) []string {
	if !backend.encrypted || fileEncryption != nil {
		return nil
	}

	return []string{
		"State is encrypted at rest but the export file is not; it can contain resolved " +
			"secrets, so encrypt the export or store it securely and remove it once it is " +
			"no longer needed",
	}
}

func writeOutputData(params ExportParams, data []byte) error {
	if IsRemoteFile(params.FilePath) {
		return UploadRemoteFile(context.Background(), params.FilePath, data, params.RemoteOptions)
	}

	// Readable by the owner only, since a plaintext export can hold resolved secrets. The
	// mode given to WriteFile only applies to a new file, so an existing one is changed
	// before anything is written to it.
	err := params.FileSystem.Chmod(params.FilePath, 0600)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return afero.WriteFile(params.FileSystem, params.FilePath, data, 0600)
}
