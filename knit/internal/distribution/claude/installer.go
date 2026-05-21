package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is the Claude Code implementation of inventory.Installer.
//
// The actual installation transaction lives in inventory.TransactionalInstaller,
// which handles preflight checks, file writes, label persistence, and
// rollback uniformly across distribution targets. This wrapper exists to
// translate Claude-specific sentinels (such as ErrUnmanagedArtifactExists)
// and to preserve the existing constructor signature so the CLI layer does
// not need to be touched in lockstep.
type Installer struct {
	core   *inventory.TransactionalInstaller
	policy pathPolicy
}

// NewInstaller constructs an Installer from the user / project Inventory
// roots and a LabelStore. Empty projectRoot keeps ScopeProject operations
// returning ErrProjectRootNotConfigured at call time (matching the prior
// contract).
//
// Returns a pointer that always satisfies inventory.Installer. A nil error
// in this signature would propagate to every distribution-side caller, so
// the function panics on programmer errors (nil store, malformed roots);
// the CLI factory layer is responsible for passing valid arguments.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Installer {
	resolver, policy := buildResolver(userRoot, projectRoot)
	store := inventory.NewFsArtifactStore()
	core, err := inventory.NewTransactionalInstaller(store, labels, resolver)
	if err != nil {
		// Pre-validated by buildResolver and the CLI factory; treat as
		// programmer error.
		panic(fmt.Errorf("claude: construct installer: %w", err))
	}
	return &Installer{core: core, policy: policy}
}

// Target returns the distribution target handled by this Installer.
func (i *Installer) Target() source.Target { return Target }

// Install delegates to the shared transactional installer and remaps the
// neutral inventory sentinels to the Claude-specific ones the CLI already
// reacts to.
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

// Compile-time interface assertion.
var _ inventory.Installer = (*Installer)(nil)
