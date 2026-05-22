package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is the Claude Code implementation of inventory.Installer. The
// actual install transaction lives in inventory.TransactionalInstaller;
// this wrapper translates neutral inventory sentinels into the
// Claude-specific ones the CLI already reacts to.
type Installer struct {
	core *inventory.TransactionalInstaller
}

// NewInstaller constructs an Installer from the user / project Inventory
// roots and a LabelStore. userRoot must be a non-empty absolute path.
// Empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) (*Installer, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalInstaller(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("claude: construct installer: %w", err)
	}
	return &Installer{core: core}, nil
}

// Target returns the distribution target handled by this Installer.
func (i *Installer) Target() source.Target { return Target }

// Install delegates to the shared transactional installer and remaps the
// neutral inventory sentinels to the Claude-specific ones.
func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	installed, err := i.core.Install(ctx, scope, artifact)
	if err != nil {
		return inventory.Installation{}, translateInstallError(err, artifact.Path)
	}
	return installed, nil
}

// translateInstallError maps the neutral inventory sentinels into the
// Claude-specific sentinels that this package exposes to consumers.
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
