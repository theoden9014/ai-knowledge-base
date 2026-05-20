package gemini

import "errors"

// ErrProjectRootNotConfigured is returned when an operation involving
// ScopeProject is requested but no project root (the base of
// `<project>/.gemini`) was provided to this package's constructor.
//
// This error is distinct from inventory.ErrInvalidScope. It indicates that the
// Scope value itself is valid, but the context required to resolve that Scope
// is missing, meaning the cli layer could not discover the project root.
var ErrProjectRootNotConfigured = errors.New("gemini: project root not configured")

// ErrInvalidArtifactPath is returned when Artifact.Path is any of the
// following:
//   - an empty string
//   - an absolute path
//   - a path that escapes the Inventory root via parent-directory references
//     such as ".."
//   - a path whose leading segment does not match Gemini CLI directory
//     conventions (this package accepts only "skills/", "agents/",
//     "commands/", or "GEMINI.md" as valid destinations)
var ErrInvalidArtifactPath = errors.New("gemini: invalid artifact path")

// ErrFrontmatterMergeConflict is returned when
// Entry.Tools["gemini"].Frontmatter structurally conflicts with reserved
// fields produced by the neutral conversion or with the generated artifact
// format itself.
//
// Keys with the same names as normal neutral fields (name / description, etc.)
// are allowed as overrides. This error refers instead to structurally
// unrepresentable cases, such as requiring frontmatter for a kind that has no
// frontmatter.
//
// Specifically, this error is returned in the following case:
//   - when a source.KindRule Entry has non-empty Tools["gemini"].Frontmatter
//     (the generated GEMINI.md cannot have frontmatter under the Gemini CLI
//     specification)
//
// KindPrompt (commands/<name>.toml) uses TOML rather than a frontmatter-bearing
// format, but this error is not returned there because
// Tools["gemini"].Frontmatter can still be reflected as TOML top-level keys.
var ErrFrontmatterMergeConflict = errors.New("gemini: frontmatter merge conflict")

// ErrUnmanagedArtifactExists is returned by Installer.Install when a file
// already exists at the destination absolute path but the corresponding Label
// sidecar does not exist.
//
// If knit already manages an Installation at the same
// (Target, Scope, destination), inventory.ErrAlreadyInstalled is returned
// instead. This error is a separate sentinel used as a safeguard to avoid
// accidentally overwriting artifacts not created by knit, such as files
// created manually by the user.
//
// Callers can use this to tell users that, to bring the artifact under knit
// management, they must either move the existing file aside or use an explicit
// force-install option.
var ErrUnmanagedArtifactExists = errors.New("gemini: unmanaged artifact already exists at destination")

// ErrUnsupportedFrontmatterValue is returned when a value in
// Entry.Tools["gemini"].Frontmatter has a Go type that cannot be represented
// during TOML encoding, such as a function value, a channel, or a nested
// structure containing cyclic references.
//
// This error can arise only when generating a KindPrompt artifact
// (commands/<name>.toml). Values that can arrive through YAML
// (string / int / float / bool / array / map[string]any) are defined to
// encode to TOML in the same shape, so normal use of the neutral format does
// not trigger it.
var ErrUnsupportedFrontmatterValue = errors.New("gemini: unsupported frontmatter value for TOML encoding")
