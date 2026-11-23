package stateio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/common/crypto"
)

const (
	exportFileFormat  = "bluelink-state-export"
	exportFileScope   = "bluelink-state-export"
	exportFileVersion = 1
)

// The wrapper an encrypted export file is written as. The envelope records which key
// sealed the payload but not a passphrase's salt, and a file has no store to keep one in,
// so the salt travels in the wrapper. The salt needs no other protection, since a changed
// salt derives a key that cannot open the payload.
type encryptedExportFile struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	// When the file was exported, in RFC 3339 form. It is bound to the payload, so it
	// cannot be changed to pass an older export off as a newer one.
	ExportedAt string `json:"exported_at"`
	Salt       []byte `json:"salt,omitempty"`
	Payload    []byte `json:"payload"`
}

// Export files are meant to be moved and renamed, so the ciphertext is bound to the format
// and the export time rather than to a path, the way state objects are bound to their keys.
func (f *encryptedExportFile) associatedData() []byte {
	return fmt.Appendf(nil, "%s:v%d:%s", exportFileFormat, f.Version, f.ExportedAt)
}

// The contents of an export file, and when it was exported if the file records it.
type openedExportFile struct {
	data       []byte
	encrypted  bool
	exportedAt time.Time
}

func sealExportFile(
	ctx context.Context,
	keyConfig *StateEncryptionKeyConfig,
	plaintext []byte,
	exportedAt time.Time,
) ([]byte, error) {
	salt, err := exportFileSalt(keyConfig)
	if err != nil {
		return nil, err
	}

	provider, err := createKeyProviderForKey(ctx, keyConfig, salt, "")
	if err != nil {
		return nil, err
	}

	wrapper := &encryptedExportFile{
		Format:     exportFileFormat,
		Version:    exportFileVersion,
		ExportedAt: exportedAt.UTC().Format(time.RFC3339),
		Salt:       salt,
	}
	wrapper.Payload, err = crypto.Seal(ctx, provider, exportFileScope, plaintext, wrapper.associatedData())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(wrapper, "", "  ")
}

// A fresh salt for every file, so two exports sealed with the same passphrase do not
// share a derived key.
func exportFileSalt(keyConfig *StateEncryptionKeyConfig) ([]byte, error) {
	if keyConfig.Provider != passphraseKeyProvider {
		return nil, nil
	}

	return crypto.GenerateSalt()
}

// Returns the instances JSON held in an export file, decrypting it when it is an
// encrypted export file. Plaintext exports are a bare JSON array and pass through as is,
// unless an encrypted file is required.
func openExportFile(
	ctx context.Context,
	keyConfig *StateEncryptionKeyConfig,
	requireEncrypted bool,
	data []byte,
) (*openedExportFile, error) {
	// Editors on Windows may save a file with a byte order mark, which JSON parsing rejects.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	wrapper, err := parseEncryptedExportFile(data)
	if err != nil {
		return nil, err
	}
	if wrapper == nil && requireEncrypted {
		return nil, &ImportError{
			Code:    ErrCodeFileNotEncrypted,
			Message: "the input file is not encrypted, but a key was given to decrypt it",
			Hint: "Check that the file is the export you expect, since an encrypted file may have " +
				"been replaced. To import a plaintext file, leave out the key.",
		}
	}
	if wrapper == nil {
		return &openedExportFile{data: data}, nil
	}

	if keyConfig == nil {
		return nil, &ImportError{
			Code:    ErrCodeFileKeyRequired,
			Message: "the input file is encrypted",
			Hint:    "Provide the key the file was encrypted with to import it.",
		}
	}

	return decryptExportFile(ctx, keyConfig, wrapper)
}

// A plaintext export is a JSON array, so any other JSON object is either an encrypted
// export file or not a state export at all. It is reported here rather than left to fail
// as an array of instances, which would not say what is wrong with the file.
func parseEncryptedExportFile(data []byte) (*encryptedExportFile, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return nil, nil
	}

	wrapper := &encryptedExportFile{}
	if err := json.Unmarshal(data, wrapper); err != nil {
		return nil, &ImportError{
			Code:    ErrCodeInvalidJSON,
			Message: "the input file is not valid JSON",
			Err:     err,
		}
	}
	if wrapper.Format != exportFileFormat {
		return nil, &ImportError{
			Code:    ErrCodeInvalidJSON,
			Message: "the input file is not a state export",
			Hint: fmt.Sprintf(
				"A state export is a JSON array of instances, or an encrypted export whose "+
					"\"format\" is %q. If this is an encrypted export, its format field has been changed.",
				exportFileFormat,
			),
		}
	}

	return wrapper, nil
}

func decryptExportFile(
	ctx context.Context,
	keyConfig *StateEncryptionKeyConfig,
	wrapper *encryptedExportFile,
) (*openedExportFile, error) {
	opened, err := openEncryptedPayload(ctx, keyConfig, wrapper)
	if err != nil {
		return nil, &ImportError{
			Code:    ErrCodeFileDecryptionFailed,
			Message: "the input file could not be decrypted",
			Hint: "Check that the key is the one the file was encrypted with, and that the file " +
				"has not been changed since it was exported.",
			Err: err,
		}
	}
	return opened, nil
}

func openEncryptedPayload(
	ctx context.Context,
	keyConfig *StateEncryptionKeyConfig,
	wrapper *encryptedExportFile,
) (*openedExportFile, error) {
	exportedAt, err := checkEncryptedExportFile(keyConfig, wrapper)
	if err != nil {
		return nil, err
	}

	provider, err := createKeyProviderForKey(ctx, keyConfig, wrapper.Salt, "")
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.Open(ctx, provider, wrapper.Payload, wrapper.associatedData())
	if err != nil {
		return nil, err
	}
	return &openedExportFile{
		data:       plaintext,
		encrypted:  true,
		exportedAt: exportedAt,
	}, nil
}

func checkEncryptedExportFile(keyConfig *StateEncryptionKeyConfig, wrapper *encryptedExportFile) (time.Time, error) {
	if wrapper.Version != exportFileVersion {
		return time.Time{}, fmt.Errorf(
			"unsupported encrypted export file version %d, this version supports %d",
			wrapper.Version,
			exportFileVersion,
		)
	}

	if keyConfig.Provider == passphraseKeyProvider && len(wrapper.Salt) == 0 {
		return time.Time{}, fmt.Errorf(
			"the file was not encrypted with a passphrase, it has no salt to derive the key from",
		)
	}

	exportedAt, err := time.Parse(time.RFC3339, wrapper.ExportedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("the file's export time %q is not valid", wrapper.ExportedAt)
	}
	return exportedAt, nil
}
