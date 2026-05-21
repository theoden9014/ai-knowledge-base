package gemini

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is the Gemini CLI implementation of inventory.Lister.
type Lister struct {
	core *inventory.TransactionalLister
}

// NewLister constructs a Lister.
func NewLister(userRoot, projectRoot string, labels inventory.LabelStore) *Lister {
	resolver, _ := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalLister(store, labels, resolver)
	if err != nil {
		panic(fmt.Errorf("gemini: construct lister: %w", err))
	}
	return &Lister{core: core}
}

// Target returns the distribution target handled by this Lister.
func (l *Lister) Target() source.Target { return Target }

// List delegates to the shared transactional lister.
func (l *Lister) List(ctx context.Context, scope inventory.Scope) ([]inventory.Installation, error) {
	out, err := l.core.List(ctx, scope)
	if err != nil {
		if errors.Is(err, inventory.ErrProjectRootNotConfigured) {
			return nil, ErrProjectRootNotConfigured
		}
		return nil, err
	}
	return out, nil
}

var _ inventory.Lister = (*Lister)(nil)
var _ inventory.Installer = (*Installer)(nil)
var _ source.Builder = (*Builder)(nil)
