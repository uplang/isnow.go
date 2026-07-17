package command

import (
	"errors"
	"fmt"
	"io"

	isnow "github.com/uplang/isnow.go"
	"github.com/uplang/isnow.go/internal/constants"
)

// Report writes a diagnostic for err (unless it is nil or the silent
// "does not hold" verdict, whose exit code is the answer) and returns the exit
// code. It is the composition root's single place for surfacing errors.
func Report(w io.Writer, err error) int {
	if err != nil && !errors.Is(err, ErrNotHolds) {
		_, _ = fmt.Fprintf(w, "isnow: %s\n", err)
	}
	return ExitCode(err)
}

// ErrNotHolds signals the membership test failed (CLI exit code 1).
var ErrNotHolds = errors.New("isnow does not hold")

// usageError is a bad-argument condition (CLI exit code 2).
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

// userErrors are the sentinels that map to exit code 2 (invalid input).
var userErrors = []error{
	isnow.ErrSyntax, isnow.ErrSymbol, isnow.ErrRange, isnow.ErrContext,
	constants.ErrBadTime, constants.ErrBadZone,
	constants.ErrMissingCommand, constants.ErrReadTab,
}

// ExitCode maps a command error to the CLI exit code (specs/contracts/cli.md).
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrNotHolds):
		return 1
	case isUserError(err):
		return 2
	default:
		return 3
	}
}

func isUserError(err error) bool {
	var ue usageError
	if errors.As(err, &ue) {
		return true
	}
	for _, sentinel := range userErrors {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
