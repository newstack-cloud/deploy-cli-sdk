package stateio

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrorExplanation splits an import or export error into the parts a person reads:
// what failed, what to do about it, and the underlying cause.
type ErrorExplanation struct {
	// Code is the error's code, such as "state_decryption_failed", or empty for an error
	// that is not from this package.
	Code string
	// Summary is a short description of what failed. It can span several lines, such as
	// the list of problems that stopped an import.
	Summary string
	// Hint says what to check or change to fix the error, or is empty.
	Hint string
	// Cause is the innermost error behind the summary, or is empty when there is none
	// or it says nothing the summary does not.
	Cause string
}

// ExplainError explains an error returned by Import or Export. Errors from this package
// are split into their summary, hint and innermost cause; any other error is returned
// whole as the summary.
func ExplainError(err error) ErrorExplanation {
	if err == nil {
		return ErrorExplanation{}
	}

	if accessErr, ok := errors.AsType[*StateAccessError](err); ok {
		return newErrorExplanation(
			string(accessErr.Code), accessErr.Message, accessErr.Hint, accessErr.Err,
		)
	}

	if importErr, ok := errors.AsType[*ImportError](err); ok {
		return newErrorExplanation(
			string(importErr.Code), importErr.Message, importErr.Hint, importErr.Err,
		)
	}

	if exportErr, ok := errors.AsType[*ExportError](err); ok {
		return newErrorExplanation(
			string(exportErr.Code), exportErr.Message, exportErr.Hint, exportErr.Err,
		)
	}

	return ErrorExplanation{Summary: err.Error()}
}

func newErrorExplanation(code string, summary string, hint string, err error) ErrorExplanation {
	cause := innermostCause(err)
	if cause == summary {
		cause = ""
	}
	return ErrorExplanation{Code: code, Summary: summary, Hint: hint, Cause: cause}
}

// Matches the package prefix Go errors conventionally start with, such as "crypto: ".
var errorPackagePrefix = regexp.MustCompile(`^[a-z][a-z0-9]*: `)

// Skips the wrapping layers libraries add with a package prefix, such as "statestore: ",
// which only say where the error passed through. It stops at a typed error such as
// *fs.PathError, which carries detail like the path, and at layers this package adds, such
// as "state.encryption.previous_keys[1]: ", which name the setting at fault.
func innermostCause(err error) string {
	for err != nil && isLibraryContextLayer(err) {
		err = errors.Unwrap(err)
	}
	if err == nil {
		return ""
	}
	return errorPackagePrefix.ReplaceAllString(strings.TrimSpace(err.Error()), "")
}

func isLibraryContextLayer(err error) bool {
	return fmt.Sprintf("%T", err) == "*fmt.wrapError" && errorPackagePrefix.MatchString(err.Error())
}
