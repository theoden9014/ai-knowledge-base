package gemini

import (
	"context"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type Uninstaller struct {
	core *inventory.TransactionalUninstaller
}

func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) (*Uninstaller, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalUninstaller(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("gemini: construct uninstaller: %w", err)
	}
	return &Uninstaller{core: core}, nil
}

func (u *Uninstaller) Target() source.Target { return Target }

func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	return Sentinels.TranslateUninstallError(u.core.Uninstall(ctx, installation), installation.Artifact.Path)
}

var _ inventory.Uninstaller = (*Uninstaller)(nil)
