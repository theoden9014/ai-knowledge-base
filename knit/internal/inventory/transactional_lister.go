package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// TransactionalLister is the target-neutral implementation of the
// inventory.Lister contract. It enumerates labels from LabelStore, drops
// orphan entries whose artifact has disappeared from the storage backend,
// and returns Installations that match the (label-yes, file-yes) cell of
// the Inventory state table.
type TransactionalLister struct {
	reader   ArtifactReader
	labels   LabelStore
	resolver *PathResolver
}

// NewTransactionalLister validates dependencies and returns a Lister. It
// accepts ArtifactReader rather than ArtifactStore so list-only paths
// cannot mutate the storage backend.
func NewTransactionalLister(reader ArtifactReader, labels LabelStore, resolver *PathResolver) (*TransactionalLister, error) {
	if reader == nil {
		return nil, errors.New("inventory: transactional lister requires artifact reader")
	}
	if labels == nil {
		return nil, errors.New("inventory: transactional lister requires label store")
	}
	if resolver == nil {
		return nil, errors.New("inventory: transactional lister requires path resolver")
	}
	return &TransactionalLister{reader: reader, labels: labels, resolver: resolver}, nil
}

// Target returns the distribution target bound at construction time.
func (l *TransactionalLister) Target() source.Target { return l.resolver.Target() }

// List enumerates labeled Installations under scope. Orphan labels (label
// present but artifact absent) are silently excluded from the result.
func (l *TransactionalLister) List(ctx context.Context, scope Scope) ([]Installation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Validate scope (and the project-root configuration when applicable)
	// before talking to the label store.
	if _, err := l.resolver.ResolveRoot(scope); err != nil {
		return nil, err
	}
	entries, err := l.labels.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	var out []Installation
	for _, e := range entries {
		rel, err := source.NewArtifactPath(e.Data.ArtifactPath)
		if err != nil {
			// Skip malformed labels rather than failing the whole list:
			// they cannot point to a real Installation.
			continue
		}
		abs, err := l.resolver.Resolve(scope, rel)
		if err != nil {
			continue
		}
		present, err := l.reader.Exists(ctx, abs)
		if err != nil {
			return nil, fmt.Errorf("inventory: exists: %w", err)
		}
		if !present {
			// Orphan label: file disappeared since installation. Skip.
			continue
		}
		out = append(out, Installation{
			ID:    e.ID,
			Label: Label{Target: l.Target(), Scope: scope},
			Provenance: Provenance{
				SourceEntryIDs: append([]string(nil), e.Data.SourceEntryIDs...),
				SourceRef:      e.Data.SourceRef,
			},
			Artifact: source.Artifact{
				Target:         l.Target(),
				Path:           rel.String(),
				SourceEntryIDs: append([]string(nil), e.Data.SourceEntryIDs...),
				SourceRef:      e.Data.SourceRef,
			},
		})
	}
	return out, nil
}

// Static type check.
var _ Lister = (*TransactionalLister)(nil)
