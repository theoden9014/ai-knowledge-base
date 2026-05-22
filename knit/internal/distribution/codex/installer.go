package codex

import (
	"context"
	"errors"
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
	resolver, err := buildResolver(userRoot, projectRoot)
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
		return inventory.Installation{}, translateInstallError(err, artifact.Path)
	}
	return installed, nil
}

func translateInstallError(err error, artifactPath string) error {
	switch {
	case errors.Is(err, source.ErrInvalidArtifactPath):
		return fmt.Errorf("%w: %s", ErrInvalidArtifactPath, artifactPath)
	case errors.Is(err, inventory.ErrProjectRootNotConfigured):
		return ErrProjectRootNotConfigured
	case errors.Is(err, inventory.ErrUnmanagedArtifactExists):
		return fmt.Errorf("%w: path=%s", ErrUnmanagedArtifactExists, artifactPath)
	default:
		return err
	}
}

var _ inventory.Installer = (*Installer)(nil)
