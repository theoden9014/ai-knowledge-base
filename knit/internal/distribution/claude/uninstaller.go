package claude

import (
	"context"
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
// [NewInstaller]: userRoot must be a non-empty absolute path, and empty
// projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) (*Uninstaller, error) {
	resolver, err := buildResolver(userRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	core, err := inventory.NewTransactionalUninstaller(inventory.NewFsArtifactStore(), labels, resolver)
	if err != nil {
		return nil, fmt.Errorf("claude: construct uninstaller: %w", err)
	}
	return &Uninstaller{core: core}, nil
}

// Target returns the distribution target handled by this Uninstaller.
func (u *Uninstaller) Target() source.Target { return Target }

// Uninstall delegates to the shared transactional uninstaller and remaps
// the neutral inventory sentinels via Sentinels.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	return Sentinels.TranslateUninstallError(u.core.Uninstall(ctx, installation), installation.Artifact.Path)
}

var _ inventory.Uninstaller = (*Uninstaller)(nil)
