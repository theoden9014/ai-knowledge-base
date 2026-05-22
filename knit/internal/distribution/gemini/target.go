package gemini

import (
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Target is the source.Target constant representing the distribution target
// handled by this package.
// Its value is the kebab-case string "gemini", matching both the key in
// Entry.Tools and the string that appears in Pack.DefaultTools.
//
// The Target() methods of this package's Builder / Installer / Uninstaller /
// Lister all return this value.
const Target source.Target = "gemini"

// Sentinels is the SentinelMap that wraps neutral inventory sentinels into
// Gemini-specific ones.
var Sentinels = inventory.SentinelMap{
	InvalidArtifactPath:      ErrInvalidArtifactPath,
	ProjectRootNotConfigured: ErrProjectRootNotConfigured,
	UnmanagedArtifactExists:  ErrUnmanagedArtifactExists,
}
