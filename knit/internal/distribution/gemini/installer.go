package gemini

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is the Gemini CLI implementation of inventory.Installer.
// Installation logic is delegated to inventory.TransactionalInstaller; this
// wrapper preserves the existing constructor signature and translates
// neutral sentinels into the Gemini-specific ones the CLI already reacts
// to.
type Installer struct {
	core *inventory.TransactionalInstaller
}

// NewInstaller constructs an Installer.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Installer {
	resolver, _ := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalInstaller(store, labels, resolver)
	if err != nil {
		panic(fmt.Errorf("gemini: construct installer: %w", err))
	}
	return &Installer{core: core}
}

// Target returns the distribution target handled by this Installer.
func (i *Installer) Target() source.Target { return Target }

// Install delegates to the shared transactional installer.
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
