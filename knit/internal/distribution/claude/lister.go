package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is the Claude Code implementation of inventory.Lister. It
// delegates label enumeration and orphan filtering to
// inventory.TransactionalLister.
type Lister struct {
	core *inventory.TransactionalLister
}

// NewLister constructs a Lister. The argument contract matches
// [NewInstaller].
func NewLister(userRoot, projectRoot string, labels inventory.LabelStore) (*Lister, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalLister(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("claude: construct lister: %w", err)
	}
	return &Lister{core: core}, nil
}

// Target returns the distribution target handled by this Lister.
func (l *Lister) Target() source.Target { return Target }

// List delegates to the shared transactional lister and remaps the neutral
// inventory sentinels to the Claude-specific ones.
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
