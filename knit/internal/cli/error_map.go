package cli

import (
	"errors"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/claude"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/codex"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/gemini"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// errorToExitCode is the single aggregation point that derives an
// ExitCode from an error.
//
// Design intent:
//   - Each subcommand Run focuses only on "returning an error", while
//     ExitCode selection is centralized here
//     (SRP: separate translation from exit-code assignment).
//   - Both the sentinel errors from the source / inventory /
//     distribution layers and the CLI-specific sentinels (errors.go) are
//     matched in order via errors.Is.
//   - Matching order goes from more specific errors to more general
//     errors, and the first match wins.
//
// Mapping:
//
//	cli.ErrUsage (and ErrUnknownCommand / ErrMissingArgument /
//	  ErrInvalidFlagValue)             -> ExitUsage
//	cli.ErrHomeNotSet                  -> ExitConfig
//	cli.ErrProjectRootNotFound         -> ExitConfig
//	cli.ErrKnowledgeDirNotFound        -> ExitConfig
//	cli.ErrPartialFailure              -> ExitPartial   (also via AggregateError.Is)
//	inventory.ErrInvalidScope          -> ExitUsage   (reduces to invalid CLI input)
//	inventory.ErrTargetMismatch        -> ExitGeneral (internal inconsistency)
//	inventory.ErrAlreadyInstalled      -> ExitConflict
//	inventory.ErrInstallationNotFound  -> ExitNotFound
//	   (The uninstall command has a path that absorbs this sentinel with
//	    errors.Is and downgrades it to a warning. This mapping remains as
//	    the final fallback if it leaks out of that layer.)
//	source.ErrManifestNotFound         -> ExitNotFound
//	source.ErrEntryNotFound            -> ExitNotFound
//	source.ErrSchemaViolation          -> ExitGeneral
//	source.ErrIDMismatch               -> ExitGeneral
//	source.ErrDuplicateEntryID         -> ExitGeneral
//	source.ErrInvalidKind              -> ExitGeneral
//	distribution/<*>.ErrProjectRootNotConfigured -> ExitConfig
//	distribution/<*>.ErrUnmanagedArtifactExists  -> ExitConflict
//	distribution/<*>.ErrInvalidArtifactPath      -> ExitGeneral
//	distribution/<*>.ErrFrontmatterMergeConflict -> ExitGeneral
//	gemini.ErrUnsupportedFrontmatterValue        -> ExitGeneral
//	remote.ErrInvalidLocator                     -> ExitUsage   (malformed URL string)
//	remote.ErrUnsupportedHost                    -> ExitConfig  (no Fetcher registered = config issue)
//	remote.ErrCloneFailed                        -> ExitGeneral
//	remote.ErrCleanupFailed                      -> ExitGeneral
//	  (anything not covered above)     -> ExitGeneral
//
// For AggregateError, [AggregateError.Is] returns true for
// ErrPartialFailure, so this function matches it on the ErrPartialFailure
// path and assigns ExitPartial. It does not recursively match child
// errors for individual Targets; once an error has been aggregated, it
// is treated as a partial failure.
//
// If nil is passed, ExitSuccess is returned.
func errorToExitCode(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}

	// Check ExitPartial (aggregate failure) first. Because
	// AggregateError.Is matches ErrPartialFailure, decide here before any
	// Unwrap-based matches against child sentinels can run.
	if errors.Is(err, ErrPartialFailure) {
		return ExitPartial
	}

	// ExitUsage family
	switch {
	case errors.Is(err, ErrUsage),
		errors.Is(err, ErrUnknownCommand),
		errors.Is(err, ErrMissingArgument),
		errors.Is(err, ErrInvalidFlagValue),
		errors.Is(err, inventory.ErrInvalidScope),
		errors.Is(err, remote.ErrInvalidLocator):
		return ExitUsage
	}

	// ExitConfig family
	switch {
	case errors.Is(err, ErrHomeNotSet),
		errors.Is(err, ErrProjectRootNotFound),
		errors.Is(err, ErrKnowledgeDirNotFound),
		errors.Is(err, claude.ErrProjectRootNotConfigured),
		errors.Is(err, codex.ErrProjectRootNotConfigured),
		errors.Is(err, gemini.ErrProjectRootNotConfigured),
		errors.Is(err, remote.ErrUnsupportedHost):
		return ExitConfig
	}

	// ExitConflict family
	switch {
	case errors.Is(err, inventory.ErrAlreadyInstalled),
		errors.Is(err, claude.ErrUnmanagedArtifactExists),
		errors.Is(err, codex.ErrUnmanagedArtifactExists),
		errors.Is(err, gemini.ErrUnmanagedArtifactExists):
		return ExitConflict
	}

	// ExitNotFound family
	switch {
	case errors.Is(err, inventory.ErrInstallationNotFound),
		errors.Is(err, source.ErrManifestNotFound),
		errors.Is(err, source.ErrEntryNotFound):
		return ExitNotFound
	}

	// Everything else (internal inconsistency, schema violation,
	// unknown error, etc.)
	return ExitGeneral
}
