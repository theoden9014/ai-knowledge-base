package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is the Gemini CLI implementation of inventory.Lister.
//
// The handled Target is always [Target]. By enumerating labels through the
// shared inventory.LabelStore, only artifacts that originate from knit are
// returned. Manually-placed files such as "~/.gemini/skills/foo/SKILL.md"
// are excluded.
type Lister struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewLister constructs a Lister from the Inventory roots and a LabelStore.
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
//
// Orphaned labels whose artifact files no longer exist are silently excluded
// from the result, matching the existing contract.
func (l *Lister) List(ctx context.Context, scope inventory.Scope) ([]inventory.Installation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rs, err := l.resolver.resolve(scope)
	if err != nil {
		return nil, err
	}
	entries, err := l.labels.List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("gemini: list labels: %w", err)
	}
	var out []inventory.Installation
	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		abs, aErr := rs.artifactPath(ent.Data.ArtifactPath)
		if aErr != nil {
			continue
		}
		if _, sErr := os.Stat(abs); sErr != nil {
			continue
		}
		out = append(out, inventory.Installation{
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
	return out, nil
}

// Helper for static type checking to catch signature changes early.
var _ inventory.Lister = (*Lister)(nil)
var _ inventory.Installer = (*Installer)(nil)
var _ source.Builder = (*Builder)(nil)
