package codex

import (
	"context"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is the Codex CLI implementation of inventory.Installer.
type Installer struct {
	core *inventory.TransactionalInstaller
}

// NewInstaller constructs an Installer. userRoot must be non-empty
// absolute; empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) (*Installer, error) {
	return NewInstallerWithRoots(Roots{
		UserSkills:    userRoot,
		ProjectSkills: projectRoot,
		UserAgents:    userRoot,
		ProjectAgents: projectRoot,
	}, labels)
}

// NewInstallerWithRoots constructs an Installer with independent roots for
// skills and agents.
func NewInstallerWithRoots(roots Roots, labels inventory.LabelStore) (*Installer, error) {
	resolver, err := buildResolver(roots)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalInstaller(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("codex: construct installer: %w", err)
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
