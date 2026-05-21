package codex

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is the Codex CLI implementation of inventory.Installer.
// The actual installation transaction is delegated to
// inventory.TransactionalInstaller; this wrapper preserves the existing
// constructor signature and translates neutral sentinels into the
// Codex-specific ones the CLI already reacts to.
type Installer struct {
	core *inventory.TransactionalInstaller
}

// NewInstaller constructs an Installer. Empty projectRoot keeps
// ScopeProject operations returning ErrProjectRootNotConfigured at call
// time.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Installer {
	resolver, _ := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalInstaller(store, labels, resolver)
	if err != nil {
		panic(fmt.Errorf("codex: construct installer: %w", err))
	}
	return &Installer{core: core}
}

// Target returns the distribution target handled by this Installer.
func (i *Installer) Target() source.Target { return Target }

// Install delegates to the shared transactional installer and remaps the
// neutral inventory sentinels to the Codex-specific ones.
func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	installed, err := i.core.Install(ctx, scope, artifact)
	if err != nil {
		return inventory.Installation{}, translateInstallError(err, artifact.Path)
	}
	return installed, nil
}

// translateInstallError maps neutral inventory sentinels into Codex-specific
// sentinels.
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
