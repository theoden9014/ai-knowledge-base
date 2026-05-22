package inventory

import (
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// installationIDSeparator is the encoded form of '/' used when an
// InstallationID is rendered into a single-segment base name (for example,
// a sidecar file name). The IDs knit emits are derived from generated artifact
// paths whose path segments do not contain underscores.
const installationIDSeparator = "_"

// NewInstallationIDFromArtifactPath derives an InstallationID from rel.
// The path component of an Installation's identity is the ArtifactPath
// itself: per the design contract, the InstallationID is opaque to consumers
// but is round-trippable to a base name via EncodedBaseName.
//
// Returns ErrInvalidInstallationID when rel is the zero ArtifactPath.
func NewInstallationIDFromArtifactPath(rel source.ArtifactPath) (InstallationID, error) {
	if rel.IsZero() {
		return "", ErrInvalidInstallationID
	}
	return InstallationID(rel.String()), nil
}

// EncodedBaseName returns the InstallationID rendered as a single path
// segment by replacing '/' with installationIDSeparator. The result is
// suitable for use as a file base name.
func (id InstallationID) EncodedBaseName() string {
	return strings.ReplaceAll(string(id), "/", installationIDSeparator)
}

// InstallationIDFromBaseName restores an InstallationID from its
// EncodedBaseName form. Returns (zero, false) when base is empty.
func InstallationIDFromBaseName(base string) (InstallationID, bool) {
	if base == "" {
		return "", false
	}
	return InstallationID(strings.ReplaceAll(base, installationIDSeparator, "/")), true
}
