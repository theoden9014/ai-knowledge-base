package codex

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Uninstaller is the Codex CLI implementation of inventory.Uninstaller.
type Uninstaller struct {
	core *inventory.TransactionalUninstaller
}

// NewUninstaller constructs an Uninstaller. The argument contract matches
// [NewInstaller].
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Uninstaller {
	resolver, _ := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalUninstaller(store, labels, resolver)
	if err != nil {
		panic(fmt.Errorf("codex: construct uninstaller: %w", err))
	}
	return &Uninstaller{core: core}
}

// Target returns the distribution target handled by this Uninstaller.
func (u *Uninstaller) Target() source.Target { return Target }

// Uninstall delegates to the shared transactional uninstaller.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	if err := u.core.Uninstall(ctx, installation); err != nil {
		return translateUninstallError(err, installation.Artifact.Path)
	}
	return nil
}

func translateUninstallError(err error, artifactPath string) error {
	switch {
	case errors.Is(err, source.ErrInvalidArtifactPath):
		return fmt.Errorf("%w: %s", ErrInvalidArtifactPath, artifactPath)
	case errors.Is(err, inventory.ErrProjectRootNotConfigured):
		return ErrProjectRootNotConfigured
	default:
		return err
	}
}

var _ inventory.Uninstaller = (*Uninstaller)(nil)
