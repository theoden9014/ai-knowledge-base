package gemini

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Uninstaller is the Gemini CLI implementation of inventory.Uninstaller.
//
// The handled Target is always [Target]. Taking Installation.Label.Scope as
// authoritative, it removes both the artifact file and its label via the
// shared inventory.LabelStore.
type Uninstaller struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewUninstaller constructs an Uninstaller from the Inventory roots and a
// LabelStore. See [NewInstaller] for the meaning of each argument.
//
// If projectRoot is empty, Uninstall calls involving ScopeProject Installations
// return ErrProjectRootNotConfigured.
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Uninstaller {
	return &Uninstaller{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Uninstaller. It
// always returns [Target].
func (u *Uninstaller) Target() source.Target {
	return Target
}

// Uninstall removes the given Installation from the Inventory.
//
// Error precedence:
//  1. inventory.ErrTargetMismatch   (Label.Target ≠ [Target])
//  2. inventory.ErrInvalidScope     (Label.Scope is invalid)
//  3. ErrProjectRootNotConfigured   (scope is ScopeProject but projectRoot is unset)
//  4. inventory.ErrInstallationNotFound (label does not exist)
//
// Deletion order: artifact -> label. The artifact's parent directory is
// removed when it becomes empty, so unmanaged sibling files keep the directory
// in place.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if installation.Label.Target != Target {
		return fmt.Errorf("%w: got %q, want %q", inventory.ErrTargetMismatch, installation.Label.Target, Target)
	}
	scope := installation.Label.Scope
	rs, err := u.resolver.resolve(scope)
	if err != nil {
		return err
	}

	artifactPath := installation.Artifact.Path
	id := installation.ID
	if id == "" && artifactPath != "" {
		id = u.resolver.installationID(artifactPath)
	}
	if id == "" {
		return fmt.Errorf("%w: installation has empty ID and Artifact.Path", inventory.ErrInstallationNotFound)
	}

	// Pass the error through unwrapped so callers can errors.Is(ErrInstallationNotFound).
	data, gErr := u.labels.Get(ctx, scope, id)
	if gErr != nil {
		return gErr
	}
	if artifactPath == "" {
		artifactPath = data.ArtifactPath
	}
	abs, err := rs.artifactPath(artifactPath)
	if err != nil {
		return err
	}

	if err := os.Remove(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("gemini: remove artifact: %w", err)
	}
	_ = os.Remove(filepath.Dir(abs))

	if err := u.labels.Delete(ctx, scope, id); err != nil {
		return fmt.Errorf("gemini: remove label: %w", err)
	}
	return nil
}

// Helper for static type checking to catch signature changes early.
var _ source.Target = Target
var _ inventory.Uninstaller = (*Uninstaller)(nil)
