package codex

import "errors"

// ErrProjectRootNotConfigured is returned when an operation targeting
// ScopeProject is requested but the package constructor was not given the
// project root that anchors `<project>/.codex`.
//
// This error is distinct from inventory.ErrInvalidScope. The scope value itself
// may be valid, while the context required to resolve that scope is missing
// because the CLI layer could not determine the project root.
var ErrProjectRootNotConfigured = errors.New("codex: project root not configured")

// ErrInvalidArtifactPath is returned when Artifact.Path is any of the
// following:
//   - an empty string
//   - an absolute path
//   - a path that escapes the inventory root via a parent-directory reference
//     such as ".."
//   - a path whose leading segment does not match the Codex CLI directory
//     conventions; this package accepts only "skills/" and "agents/"
var ErrInvalidArtifactPath = errors.New("codex: invalid artifact path")

// ErrUnmanagedArtifactExists is returned from Installer.Install when the
// destination absolute path already contains a file and that file is considered
// unmanaged by knit because no corresponding label sidecar exists.
//
// This error exists to avoid silently overwriting a pre-existing file that a
// user placed manually. To retry installation, the caller must first move or
// delete the existing file explicitly.
//
// It is distinct from inventory.ErrAlreadyInstalled:
//   - inventory.ErrAlreadyInstalled means the same (Target, Scope, ID)
//     installation was already placed by knit, as proven by an existing
//     sidecar.
//   - ErrUnmanagedArtifactExists means a non-knit file already exists at the
//     destination and no sidecar exists.
//
// Installer.Install evaluates errors in this order, aligned with the claude and
// gemini packages:
//
//	inventory.ErrTargetMismatch → inventory.ErrInvalidScope
//	→ ErrProjectRootNotConfigured → ErrInvalidArtifactPath
//	→ inventory.ErrAlreadyInstalled → ErrUnmanagedArtifactExists
var ErrUnmanagedArtifactExists = errors.New("codex: unmanaged artifact already exists at destination")

// ErrInvalidSkillMetadata is returned when a skill's agents/openai.yaml
// cannot be decoded as a YAML mapping while applying target policy.
var ErrInvalidSkillMetadata = errors.New("codex: invalid skill metadata")
