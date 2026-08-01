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
//     conventions (this package accepts only "skills/" and "agents/")
var ErrInvalidArtifactPath = errors.New("gemini: invalid artifact path")

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

// ErrUnsupportedSkillInvocation is returned when a manual-only skill targets
// Gemini CLI, which has no per-skill metadata for disabling implicit use.
var ErrUnsupportedSkillInvocation = errors.New("gemini: manual-only skill invocation is unsupported")
