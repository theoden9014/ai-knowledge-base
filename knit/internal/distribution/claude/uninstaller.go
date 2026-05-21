package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Uninstaller is the Claude Code implementation of inventory.Uninstaller.
// It delegates the shared uninstall transaction (file delete -> label
// delete -> empty-parent pruning, with orphan-label tolerance) to
// inventory.TransactionalUninstaller.
type Uninstaller struct {
	core *inventory.TransactionalUninstaller
}

// NewUninstaller constructs an Uninstaller. The argument contract matches
// [NewInstaller]: empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Uninstaller {
	resolver, _ := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalUninstaller(store, labels, resolver)
	if err != nil {
		panic(fmt.Errorf("claude: construct uninstaller: %w", err))
	}
	return &Uninstaller{core: core}
}

// Target returns the distribution target handled by this Uninstaller.
func (u *Uninstaller) Target() source.Target { return Target }

// Uninstall delegates to the shared transactional uninstaller and remaps
// the neutral inventory sentinels to the Claude-specific ones the CLI
// already reacts to.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	if err := u.core.Uninstall(ctx, installation); err != nil {
		return translateUninstallError(err, installation.Artifact.Path)
	}
	return nil
}

// translateUninstallError maps the neutral inventory sentinels into the
// Claude-specific sentinels for the uninstall path.
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

// Compile-time interface assertion.
var _ inventory.Uninstaller = (*Uninstaller)(nil)
