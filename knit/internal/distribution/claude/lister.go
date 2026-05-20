package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is the Claude Code implementation of inventory.Lister.
//
// The handled target is always [Target]. Starting from LabelStore, it lists
// only artifacts that are definitively managed by knit.
type Lister struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewLister constructs a Lister from Inventory roots and a LabelStore.
// The meaning of each argument matches [NewInstaller].
//
// If projectRoot is an empty string, List calls for ScopeProject return
// ErrProjectRootNotConfigured.
func NewLister(userRoot, projectRoot string, labels inventory.LabelStore) *Lister {
	return &Lister{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Lister. It always returns [Target].
func (l *Lister) Target() source.Target {
	return Target
}

// List enumerates labels under the given Scope and returns the corresponding
// Installations.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (scope is neither ScopeUser nor ScopeProject)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
//
// Behavior:
//   - If nothing has been installed, the method returns nil or an empty slice
//     successfully.
//   - If the artifact file referenced by a label no longer exists, the entry is
//     excluded as a leftover so callers see only live Installations.
func (l *Lister) List(ctx context.Context, scope inventory.Scope) ([]inventory.Installation, error) {
	r, err := l.resolver.resolve(scope)
	if err != nil {
		return nil, err
	}
	entries, err := l.labels.List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("claude: list labels: %w", err)
	}
	var result []inventory.Installation
	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		absArtifact := filepath.Join(r.Root(), ent.Data.ArtifactPath)
		if _, aErr := os.Stat(absArtifact); aErr != nil {
			continue
		}
		result = append(result, inventory.Installation{
			ID:    ent.ID,
			Label: inventory.Label{Target: Target, Scope: scope},
			Provenance: inventory.Provenance{
				SourceEntryIDs: append([]string(nil), ent.Data.SourceEntryIDs...),
			},
			Artifact: source.Artifact{
				Target: Target,
				Path:   ent.Data.ArtifactPath,
			},
		})
	}
	return result, nil
}

// Helper for static type checking to catch signature changes early.
var _ inventory.Lister = (*Lister)(nil)
