package gemini

import (
	"context"
	"errors"
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
