package codex

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type Lister struct {
	core *inventory.TransactionalLister
}

func NewLister(userRoot, projectRoot string, labels inventory.LabelStore) (*Lister, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalLister(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("codex: construct lister: %w", err)
	}
	return &Lister{core: core}, nil
}

func (l *Lister) Target() source.Target { return Target }

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
