package cli

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors defined by this package.
//
// These are separated so callers can distinguish usage errors from
// runtime failures via errors.Is. Runtime failures are expected to wrap
// sentinel errors defined by the source, inventory, and distribution
// layers, while this package defines only CLI-specific errors.
var (
	// ErrUsage represents incorrect CLI usage such as an unknown
	// subcommand, unknown flag, missing required argument, or out-of-range
	// flag value. When Execute detects this error it writes the subcommand
	// usage to stderr and exits with ExitUsage.
	ErrUsage = errors.New("cli: usage error")

	// ErrUnknownCommand is returned when the requested subcommand is not
	// found. It is treated as a subclass of ErrUsage
	// (errors.Is(err, ErrUsage) is true).
	ErrUnknownCommand = errors.New("cli: unknown command")

	// ErrMissingArgument is returned when a required positional argument,
	// such as install's <pack>, is missing. It is treated as a subclass
	// of ErrUsage.
	ErrMissingArgument = errors.New("cli: missing argument")

	// ErrInvalidFlagValue is returned when a flag value is outside the
	// allowed range, such as --target=foo. It is treated as a subclass of
	// ErrUsage.
	ErrInvalidFlagValue = errors.New("cli: invalid flag value")

	// ErrHomeNotSet is returned when the HOME environment variable is
	// missing while resolving scope=user. It maps to ExitConfig.
	ErrHomeNotSet = errors.New("cli: HOME environment variable is not set")

	// ErrProjectRootNotFound is returned when the project root cannot be
	// found while resolving scope=project. It maps to ExitConfig.
	ErrProjectRootNotFound = errors.New("cli: project root not found")

	// ErrKnowledgeDirNotFound is returned when knowledge/ cannot be found,
	// either because --knowledge-dir was not specified or because
	// auto-detection also failed. It maps to ExitConfig.
	ErrKnowledgeDirNotFound = errors.New("cli: knowledge directory not found")

	// ErrPartialFailure represents a run across multiple Targets, such as
	// --target=all, where some Targets failed. Failure details are exposed
	// through [AggregateError], which wraps this error.
	ErrPartialFailure = errors.New("cli: partial failure across targets")
)

// AggregateError represents a multi-Target run such as --target=all in
// which some Targets succeeded and others failed.
//
// Failures stores pairs of Target name and Target error in the same
// order as the original Target arguments.
//
// # Consumer Inspection Paths
//
//  1. errors.Is(err, ErrPartialFailure) always returns true for this
//     type via [AggregateError.Is]. That is the path used by ExitCode
//     mapping to ask whether an aggregate failure occurred.
//  2. Checks for specific child sentinels, such as
//     errors.Is(err, inventory.ErrAlreadyInstalled), work via the Go
//     1.20+ `Unwrap() []error` contract and can inspect all child errors
//     transitively.
type AggregateError struct {
	// Failures is the set of failed Targets and their errors.
	Failures []TargetFailure
}

// TargetFailure is one element of an AggregateError.
type TargetFailure struct {
	// Target is the failed Target name in kebab-case, such as "claude".
	Target string
	// Err is the error produced by that Target.
	Err error
}

// Error returns a human-readable error message.
func (e *AggregateError) Error() string {
	n := len(e.Failures)
	noun := "failures"
	if n == 1 {
		noun = "failure"
	}
	if n == 0 {
		return fmt.Sprintf("%s (0 %s)", ErrPartialFailure.Error(), noun)
	}
	parts := make([]string, 0, n)
	for _, f := range e.Failures {
		parts = append(parts, f.Target+": "+f.Err.Error())
	}
	return fmt.Sprintf("%s (%d %s): [%s]", ErrPartialFailure.Error(), n, noun, strings.Join(parts, "; "))
}

// Unwrap returns each Failure.Err using the Go 1.20+ multi-error Unwrap
// contract. This lets errors.Is and errors.As traverse any child
// sentinel inside AggregateError transparently, for example
// errors.Is(err, inventory.ErrAlreadyInstalled).
//
// The returned slice preserves Failure order. The child errors are owned
// by AggregateError and must not be mutated by callers.
func (e *AggregateError) Unwrap() []error {
	if len(e.Failures) == 0 {
		return nil
	}
	out := make([]error, 0, len(e.Failures))
	for _, f := range e.Failures {
		out = append(out, f.Err)
	}
	return out
}

// Is returns true when target is [ErrPartialFailure]. Unwrap() []error
// provides transparent matches for child sentinels, but
// ErrPartialFailure represents the aggregate failure itself rather than
// a child error, so it is handled here.
//
// For any target other than ErrPartialFailure, Is returns false and
// leaves matching to the Unwrap() []error path.
func (e *AggregateError) Is(target error) bool {
	return target == ErrPartialFailure
}
