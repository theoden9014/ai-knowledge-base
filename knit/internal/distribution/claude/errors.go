package claude

import "errors"

// ErrProjectRootNotConfigured is returned when an operation involving
// ScopeProject is requested but no project root
// (the base for `<project>/.claude`) was provided to this package's constructor.
//
// This error is distinct from inventory.ErrInvalidScope. It indicates that the
// Scope value itself is valid, but the context required to resolve that Scope
// is missing (= the CLI layer could not discover the project root).
var ErrProjectRootNotConfigured = errors.New("claude: project root not configured")

// ErrInvalidArtifactPath is returned when Artifact.Path is any of the following:
//   - an empty string
//   - an absolute path
//   - a path that escapes the Inventory root via parent traversal (".."), etc.
//   - a path whose first segment does not match Claude Code directory
//     conventions (this package accepts only "skills/" and "agents/")
var ErrInvalidArtifactPath = errors.New("claude: invalid artifact path")

// ErrUnmanagedArtifactExists is returned by Installer.Install when a file
// already exists at the destination absolute path but the corresponding label
// sidecar does not.
//
// If knit already manages an Installation at the same
// (Target, Scope, destination), inventory.ErrAlreadyInstalled is returned
// instead. This error is a distinct sentinel that acts as a safety guard so
// we do not accidentally overwrite a non-knit artifact (for example, a file
// created manually by the user).
//
// Callers can guide users to either move the existing file aside or use an
// explicit force-install option before bringing it under knit management.
var ErrUnmanagedArtifactExists = errors.New("claude: unmanaged artifact already exists at destination")
