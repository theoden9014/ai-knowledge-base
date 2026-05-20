package source

import "errors"

// Sentinel errors returned by the source package. Implementations should wrap
// these with fmt.Errorf("...: %w", ErrXxx) so callers can match with
// errors.Is.
var (
	// ErrInvalidKind indicates a Kind value outside the recognized set.
	ErrInvalidKind = errors.New("source: invalid kind")

	// ErrManifestNotFound indicates that manifest.yaml is missing from
	// the pack directory.
	ErrManifestNotFound = errors.New("source: manifest.yaml not found")

	// ErrEntryNotFound indicates that an entry referenced by the
	// manifest could not be located on disk.
	ErrEntryNotFound = errors.New("source: entry file not found")

	// ErrSchemaViolation indicates that the manifest or an entry
	// frontmatter failed JSON Schema validation.
	ErrSchemaViolation = errors.New("source: schema violation")

	// ErrIDMismatch indicates that an entry's frontmatter id does not
	// match the id declared for it in manifest.entries.
	ErrIDMismatch = errors.New("source: entry id does not match manifest")

	// ErrDuplicateEntryID indicates that two entries share the same
	// neutral id within a single pack.
	ErrDuplicateEntryID = errors.New("source: duplicate entry id")

	// ErrPackDirNotFound indicates that a directory passed as a local
	// pack directory path (absolute or relative) does not exist or is
	// not a directory. The CLI layer wraps the OS error with this
	// sentinel so users can distinguish "pack name not found in
	// knowledge/" (ErrManifestNotFound after a path is constructed)
	// from "the directory you typed does not exist on disk".
	ErrPackDirNotFound = errors.New("source: pack directory not found")
)
