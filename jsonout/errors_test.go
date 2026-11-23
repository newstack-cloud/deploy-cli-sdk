package jsonout

import (
	"errors"
	"fmt"
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	"github.com/stretchr/testify/suite"
)

type StateErrorOutputTestSuite struct {
	suite.Suite
}

func (s *StateErrorOutputTestSuite) Test_reports_the_code_of_a_state_access_error() {
	err := fmt.Errorf("opening state: %w", &stateio.StateAccessError{
		Code:    stateio.ErrCodeStateDecryptionFailed,
		Message: "the state could not be decrypted",
		Hint:    "Check the configured key.",
		Err:     errors.New("crypto: envelope failed authentication"),
	})

	output := NewErrorOutput(err)

	s.False(output.Success)
	s.Equal(ErrorDetail{
		Type:    "state_decryption_failed",
		Message: "the state could not be decrypted",
		Hint:    "Check the configured key.",
		Cause:   "envelope failed authentication",
	}, output.Error)
}

func (s *StateErrorOutputTestSuite) Test_reports_the_code_of_a_wrapped_import_error() {
	err := fmt.Errorf("failed to read input file: %w", &stateio.ImportError{
		Code:    stateio.ErrCodeFileNotFound,
		Message: "input file not found",
	})

	output := NewErrorOutput(err)

	s.Equal("file_not_found", output.Error.Type)
	s.Equal("input file not found", output.Error.Message)
}

func (s *StateErrorOutputTestSuite) Test_reports_the_code_of_an_import_error() {
	output := NewErrorOutput(&stateio.ImportError{
		Code:    stateio.ErrCodeInstanceOperationInProgress,
		Message: "nothing was imported",
	})

	s.Equal("instance_operation_in_progress", output.Error.Type)
	s.Equal("nothing was imported", output.Error.Message)
}

func (s *StateErrorOutputTestSuite) Test_reports_the_code_of_an_export_error() {
	output := NewErrorOutput(&stateio.ExportError{
		Code:    stateio.ErrCodeFileEncryptionFailed,
		Message: "failed to encrypt the output file",
	})

	s.Equal("file_encryption_failed", output.Error.Type)
}

func TestStateErrorOutputTestSuite(t *testing.T) {
	suite.Run(t, new(StateErrorOutputTestSuite))
}
