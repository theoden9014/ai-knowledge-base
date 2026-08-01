package source

import "io/fs"

// Artifact is a single file produced by a Builder for one Target. A Builder
// may emit multiple Artifacts for a single Pack (for example one SKILL.md per
// skill entry plus auxiliary files).
type Artifact struct {
	// Target identifies the distribution target that owns this artifact.
	Target Target

	// Path is the artifact's path relative to the target's installation
	// root. The Installer is responsible for combining this with the
	// scope-specific base path on the host filesystem.
	Path string

	// Content is the raw file content to be written.
	Content []byte

	// Mode is the desired file mode. A zero value is interpreted by the
	// Installer as "use the default for regular files".
	Mode fs.FileMode

	// SourceEntryIDs lists the neutral ids of every Entry that
	// contributed to this artifact. It is intended for traceability
	// (logging, labels, diagnostics) and may be empty for artifacts
	// synthesized without a 1:1 mapping to an entry.
	SourceEntryIDs []string

	// SourceRef records where the containing Pack was loaded from. The CLI uses
	// remote references to refresh by pack name and local references to explain
	// why an explicit path is required.
	SourceRef SourceRef
}
