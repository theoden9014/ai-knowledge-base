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

	// ErrPackMismatch indicates that the pack directory, manifest pack name,
	// or pack segment of an entry ID do not agree.
	ErrPackMismatch = errors.New("source: pack identity mismatch")

	// ErrKindMismatch indicates that an entry's frontmatter kind differs from
	// the kind segment in its manifest entry ID.
	ErrKindMismatch = errors.New("source: entry kind does not match manifest id")

	// ErrPathMismatch indicates that a manifest entry path does not use the
	// entry-name segment from its ID.
	ErrPathMismatch = errors.New("source: entry path does not match manifest id")

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

	// ErrSkillResolution is the umbrella sentinel that matches any
	// failure encountered while resolving a skill entry's root directory
	// against the source filesystem. The three concrete failures below
	// (ErrSkillPathNotFound / ErrSkillPathNotDirectory /
	// ErrSkillBodyNotFound) all satisfy errors.Is(err, ErrSkillResolution).
	ErrSkillResolution = errors.New("source: skill resolution failed")

	// ErrSkillPathNotFound indicates that the directory referenced by a
	// skill entry's manifest path does not exist on the source filesystem.
	ErrSkillPathNotFound = newSkillResolutionError("source: skill path not found")

	// ErrSkillPathNotDirectory indicates that the manifest path of a skill
	// entry refers to a file rather than a directory.
	ErrSkillPathNotDirectory = newSkillResolutionError("source: skill path is not a directory")

	// ErrSkillBodyNotFound indicates that the skill root directory exists
	// but does not contain a SKILL.md file at its top level.
	ErrSkillBodyNotFound = newSkillResolutionError("source: SKILL.md not found in skill directory")

	// ErrInvalidSkillAssetPath indicates that a SkillAsset constructor
	// received a relative path that fails the value object invariants
	// (empty, absolute, contains "..", contains "\", or equals "SKILL.md").
	ErrInvalidSkillAssetPath = errors.New("source: invalid skill asset path")

	// ErrInvalidSkillRoot indicates that a SkillMeta constructor received
	// a root path that fails the value object invariants (empty, absolute,
	// contains "..", contains a trailing slash, or contains a backslash).
	ErrInvalidSkillRoot = errors.New("source: invalid skill root")

	// ErrDuplicateSkillAsset indicates that a SkillMeta constructor
	// received an assets slice with a duplicate SkillAsset.Path entry.
	ErrDuplicateSkillAsset = errors.New("source: duplicate skill asset")

	// ErrInvalidSkillInvocation indicates that a skill invocation value is
	// neither "both" nor "manual".
	ErrInvalidSkillInvocation = errors.New("source: invalid skill invocation")
)

// skillResolutionError is a sentinel whose Is() method also matches the
// umbrella ErrSkillResolution, so callers can detect "any skill-resolution
// failure" with a single errors.Is check.
type skillResolutionError struct{ msg string }

func newSkillResolutionError(msg string) error { return &skillResolutionError{msg: msg} }

func (e *skillResolutionError) Error() string { return e.msg }

func (e *skillResolutionError) Is(target error) bool {
	return target == ErrSkillResolution
}
