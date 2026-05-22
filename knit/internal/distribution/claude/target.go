package claude

import (
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Target is the source.Target constant representing the distribution target
// handled by this package.
// Its value is the kebab-case string "claude", matching the string that appears
// as a key in Entry.Tools and as an element of Pack.DefaultTools.
//
// The Target() methods of this package's Builder / Installer / Uninstaller /
// Lister all return this value.
const Target source.Target = "claude"

// Sentinels is the SentinelMap that wraps neutral inventory sentinels into
// Claude-specific ones. Used by the Installer / Uninstaller / Lister
// thin wrappers around the shared inventory.Transactional* types.
var Sentinels = inventory.SentinelMap{
	InvalidArtifactPath:      ErrInvalidArtifactPath,
	ProjectRootNotConfigured: ErrProjectRootNotConfigured,
	UnmanagedArtifactExists:  ErrUnmanagedArtifactExists,
}
