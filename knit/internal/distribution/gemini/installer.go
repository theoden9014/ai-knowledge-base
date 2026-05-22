package gemini

import (
	"context"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type Installer struct {
	core *inventory.TransactionalInstaller
}

func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) (*Installer, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalInstaller(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("gemini: construct installer: %w", err)
	}
	return &Installer{core: core}, nil
}

func (i *Installer) Target() source.Target { return Target }

func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	installed, err := i.core.Install(ctx, scope, artifact)
	if err != nil {
		return inventory.Installation{}, Sentinels.TranslateInstallError(err, artifact.Path)
	}
	return installed, nil
}

var _ inventory.Installer = (*Installer)(nil)
