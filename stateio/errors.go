package stateio

import "fmt"

// ImportErrorCode represents the type of import error.
type ImportErrorCode string

const (
	// ErrCodeInvalidJSON indicates the JSON input is malformed.
	ErrCodeInvalidJSON ImportErrorCode = "invalid_json"
	// ErrCodeFileNotFound indicates the input file was not found.
	ErrCodeFileNotFound ImportErrorCode = "file_not_found"
	// ErrCodeRemoteAccessFail indicates a remote file could not be accessed.
	ErrCodeRemoteAccessFail ImportErrorCode = "remote_access_failed"
	// ErrCodeFileKeyRequired indicates the input file is encrypted and no key was provided.
	ErrCodeFileKeyRequired ImportErrorCode = "file_key_required"
	// ErrCodeFileDecryptionFailed indicates the input file could not be decrypted
	// with the provided key.
	ErrCodeFileDecryptionFailed ImportErrorCode = "file_decryption_failed"
	// ErrCodeFileNotEncrypted indicates the input file is plaintext although the import
	// required an encrypted one.
	ErrCodeFileNotEncrypted ImportErrorCode = "file_not_encrypted"
	// ErrCodeInstanceNameConflict indicates a new instance in the input uses a name
	// another instance already has.
	ErrCodeInstanceNameConflict ImportErrorCode = "instance_name_conflict"
	// ErrCodeInstanceNameImmutable indicates the input renames an existing instance.
	ErrCodeInstanceNameImmutable ImportErrorCode = "instance_name_immutable"
	// ErrCodeInstanceOperationInProgress indicates the input would overwrite an instance
	// an engine is still working on.
	ErrCodeInstanceOperationInProgress ImportErrorCode = "instance_operation_in_progress"
	// ErrCodeChildBlueprintCycle indicates instances in the input are children of each
	// other in a cycle, so none of them is a top-level instance to import them under.
	ErrCodeChildBlueprintCycle ImportErrorCode = "child_blueprint_cycle"
	// ErrCodeInstanceCopiesDiffer indicates an instance appears more than once in the
	// input and the copies are not the same.
	ErrCodeInstanceCopiesDiffer ImportErrorCode = "instance_copies_differ"
	// ErrCodeChildBlueprintMultipleParents indicates the input lists an instance as a child
	// blueprint of more than one parent, or under more than one name in one parent.
	ErrCodeChildBlueprintMultipleParents ImportErrorCode = "child_blueprint_multiple_parents"
)

// ImportError represents an error that occurred during import.
type ImportError struct {
	Code    ImportErrorCode
	Message string
	// Hint says what to check or change to fix the error, if anything.
	Hint string
	Err  error
}

func (e *ImportError) Error() string {
	return formatError(string(e.Code), e.Message, e.Hint, e.Err)
}

func (e *ImportError) Unwrap() error {
	return e.Err
}

// StateAccessErrorCode represents the reason state could not be opened
// for an import or export.
type StateAccessErrorCode string

const (
	// ErrCodeEncryptionKeyUnavailable indicates the key state is encrypted with
	// could not be built from the engine config, for example a missing passphrase,
	// an unreadable key file or an unreachable key management service.
	ErrCodeEncryptionKeyUnavailable StateAccessErrorCode = "encryption_key_unavailable"
	// ErrCodeStateDecryptionFailed indicates encrypted state could not be decrypted
	// with the configured key.
	ErrCodeStateDecryptionFailed StateAccessErrorCode = "state_decryption_failed"
	// ErrCodeStateConfigInvalid indicates the deploy engine's state configuration, from its
	// config file or environment variables, cannot be used as it is.
	ErrCodeStateConfigInvalid StateAccessErrorCode = "state_config_invalid"
	// ErrCodeStateUnavailable indicates the configured state could not be reached or loaded,
	// for example a database or bucket that cannot be connected to.
	ErrCodeStateUnavailable StateAccessErrorCode = "state_unavailable"
)

// The state settings come from the engine config file, overridden by environment
// variables. The section is a config key under state, such as "objectstore", or empty for
// the whole of state.
func stateConfigHint(section string) string {
	if section == "" {
		return "Check state in the deploy engine config, and any " + engineStateEnvPrefix +
			"* environment variables."
	}
	return fmt.Sprintf(
		"Check state.%s in the deploy engine config, and any %s_* environment variables.",
		section, stateEnvVarName(section),
	)
}

// StateAccessError represents an error opening the state an import or export works on.
type StateAccessError struct {
	Code StateAccessErrorCode
	// Message is a short summary of what failed.
	Message string
	// Hint says what to check or change to fix the error, if anything.
	Hint string
	Err  error
}

func (e *StateAccessError) Error() string {
	return formatError(string(e.Code), e.Message, e.Hint, e.Err)
}

func (e *StateAccessError) Unwrap() error {
	return e.Err
}

// ExportErrorCode represents the type of export error.
type ExportErrorCode string

const (
	// ErrCodeExportFailed indicates a general export failure.
	ErrCodeExportFailed ExportErrorCode = "export_failed"
	// ErrCodeInstanceNotFound indicates one or more instances were not found.
	ErrCodeInstanceNotFound ExportErrorCode = "not_found"
	// ErrCodeRemoteUploadFailed indicates a remote upload failed.
	ErrCodeRemoteUploadFailed ExportErrorCode = "remote_upload_failed"
	// ErrCodeFileEncryptionFailed indicates the output file could not be encrypted
	// with the provided key.
	ErrCodeFileEncryptionFailed ExportErrorCode = "file_encryption_failed"
)

// ExportError represents an error that occurred during export.
type ExportError struct {
	Code    ExportErrorCode
	Message string
	// Hint says what to check or change to fix the error, if anything.
	Hint string
	Err  error
}

func (e *ExportError) Error() string {
	return formatError(string(e.Code), e.Message, e.Hint, e.Err)
}

func (e *ExportError) Unwrap() error {
	return e.Err
}

func formatError(code string, message string, hint string, err error) string {
	formatted := fmt.Sprintf("%s: %s", code, message)
	if hint != "" {
		formatted = fmt.Sprintf("%s (%s)", formatted, hint)
	}
	if err != nil {
		return fmt.Sprintf("%s: %v", formatted, err)
	}
	return formatted
}
