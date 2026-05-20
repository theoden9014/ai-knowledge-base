package codex

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is the Codex CLI implementation of inventory.Lister.
//
// It always serves [Target]. Enumeration is performed via inventory.LabelStore;
// orphaned labels whose artifact files no longer exist are excluded so callers
// see only live Installations.
type Lister struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewLister constructs a Lister from the inventory roots and a LabelStore.
// See [NewInstaller] for the meaning of each argument.
func NewLister(userRoot, projectRoot string, labels inventory.LabelStore) *Lister {
	return &Lister{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Lister. It always
// returns [Target].
func (l *Lister) Target() source.Target {
	return Target
}

// List enumerates labeled installations for the given Scope.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (scope is invalid)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
func (l *Lister) List(ctx context.Context, scope inventory.Scope) ([]inventory.Installation, error) {
	rs, err := l.resolver.resolve(scope)
	if err != nil {
		return nil, err
	}
	entries, err := l.labels.List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("codex: list labels: %w", err)
	}
	var result []inventory.Installation
	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		artifactAbs := filepath.Join(rs.Root(), filepath.FromSlash(ent.Data.ArtifactPath))
		if exists, _ := fileExists(artifactAbs); !exists {
			continue
		}
		result = append(result, inventory.Installation{
			ID:    ent.ID,
			Label: inventory.Label{Target: Target, Scope: scope},
			Provenance: inventory.Provenance{
				SourceEntryIDs: append([]string(nil), ent.Data.SourceEntryIDs...),
			},
			Artifact: source.Artifact{
				Target:         Target,
				Path:           ent.Data.ArtifactPath,
				SourceEntryIDs: append([]string(nil), ent.Data.SourceEntryIDs...),
			},
		})
	}
	return result, nil
}

// Static interface check for early detection of signature drift.
var _ inventory.Lister = (*Lister)(nil)
