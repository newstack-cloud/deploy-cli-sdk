package stateio

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorExplanationTestSuite struct {
	suite.Suite
}

func TestErrorExplanationTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorExplanationTestSuite))
}

func (s *ErrorExplanationTestSuite) Test_skips_context_layers_to_the_cause() {
	cause := errors.New("crypto: envelope was sealed with key \"a\", this provider holds \"b\"")
	err := &StateAccessError{
		Code:    ErrCodeStateDecryptionFailed,
		Message: "the state could not be decrypted with the configured key",
		Hint:    "Check the key.",
		Err:     fmt.Errorf("statestore: decrypting %q: %w", "index.json", fmt.Errorf("crypto: unwrapping: %w", cause)),
	}

	s.Equal(ErrorExplanation{
		Code:    "state_decryption_failed",
		Summary: "the state could not be decrypted with the configured key",
		Hint:    "Check the key.",
		Cause:   "envelope was sealed with key \"a\", this provider holds \"b\"",
	}, ExplainError(err))
}

func (s *ErrorExplanationTestSuite) Test_keeps_the_detail_of_a_typed_cause() {
	pathErr := &fs.PathError{Op: "open", Path: "/keys/state.key", Err: fs.ErrNotExist}
	err := &StateAccessError{
		Code:    ErrCodeEncryptionKeyUnavailable,
		Message: "the state encryption key could not be set up",
		Err:     fmt.Errorf("localkey: reading key file: %w", pathErr),
	}

	s.Equal("open /keys/state.key: file does not exist", ExplainError(err).Cause)
}

func (s *ErrorExplanationTestSuite) Test_keeps_context_that_names_the_setting_at_fault() {
	err := &StateAccessError{
		Code:    ErrCodeEncryptionKeyUnavailable,
		Message: "the state encryption key could not be set up",
		Err: fmt.Errorf(
			"state.encryption.previous_keys[1]: %w",
			errors.New("one of state.encryption.passphrase or state.encryption.passphrase_file is required"),
		),
	}

	s.Equal(
		"state.encryption.previous_keys[1]: one of state.encryption.passphrase or "+
			"state.encryption.passphrase_file is required",
		ExplainError(err).Cause,
	)
}

func (s *ErrorExplanationTestSuite) Test_explains_import_and_export_errors() {
	importErr := &ImportError{
		Code:    ErrCodeInstanceNameImmutable,
		Message: "nothing was imported\ninstances cannot be renamed",
		Hint:    "Keep the instance's name.",
	}
	exportErr := &ExportError{
		Code:    ErrCodeExportFailed,
		Message: "failed to list instances",
		Err:     errors.New("bucket gone"),
	}

	s.Equal(ErrorExplanation{
		Code:    "instance_name_immutable",
		Summary: "nothing was imported\ninstances cannot be renamed",
		Hint:    "Keep the instance's name.",
	}, ExplainError(importErr))
	s.Equal(
		ErrorExplanation{Code: "export_failed", Summary: "failed to list instances", Cause: "bucket gone"},
		ExplainError(exportErr),
	)
}

func (s *ErrorExplanationTestSuite) Test_finds_a_wrapped_error_from_this_package() {
	err := fmt.Errorf("export: %w", &ExportError{Code: ErrCodeExportFailed, Message: "failed to list instances"})

	s.Equal(ErrorExplanation{Code: "export_failed", Summary: "failed to list instances"}, ExplainError(err))
}

func (s *ErrorExplanationTestSuite) Test_returns_any_other_error_whole() {
	s.Equal(ErrorExplanation{Summary: "connection refused"}, ExplainError(errors.New("connection refused")))
	s.Equal(ErrorExplanation{}, ExplainError(nil))
}
