package claude

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

// Uninstaller is the Claude Code implementation of inventory.Uninstaller.
//
// The handled target is always [Target]. Using Installation.Label.Scope as the
// source of truth, it removes the corresponding artifact file and label.
// Label removal is delegated to inventory.LabelStore.
type Uninstaller struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewUninstaller constructs an Uninstaller from Inventory roots and a
// LabelStore. The meaning of each argument matches [NewInstaller].
//
// If projectRoot is an empty string, Uninstall calls for Installations in
// ScopeProject return ErrProjectRootNotConfigured.
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Uninstaller {
	return &Uninstaller{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Uninstaller. It always returns [Target].
func (u *Uninstaller) Target() source.Target {
	return Target
}

// Uninstall removes the given Installation from Inventory.
//
// Error precedence:
//  1. inventory.ErrTargetMismatch  (Label.Target ≠ [Target])
//  2. inventory.ErrInvalidScope    (Label.Scope is neither ScopeUser nor ScopeProject)
//  3. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
//  4. inventory.ErrInstallationNotFound (label does not exist)
//
// Deletion order: artifact -> label. Even if the artifact is already missing,
// the label is still removed so reruns converge cleanly.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
	if installation.Label.Target != Target {
		return fmt.Errorf("%w: installation.Label.Target=%q uninstaller.Target=%q",
			inventory.ErrTargetMismatch, installation.Label.Target, Target)
	}
	scope := installation.Label.Scope
	r, err := u.resolver.resolve(scope)
	if err != nil {
		return err
	}

	// Source the artifact path from the input Installation, falling back to the
	// label when the caller passes only an ID (Lister populates both).
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

	absArtifactPath, err := r.ResolveArtifactPath(artifactPath)
	if err != nil {
		return err
	}
	if err := os.Remove(absArtifactPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("claude: remove artifact: %w", err)
	}
	_ = os.Remove(filepath.Dir(absArtifactPath))
	if err := u.labels.Delete(ctx, scope, id); err != nil {
		return fmt.Errorf("claude: remove label: %w", err)
	}
	return nil
}

// Helper for static type checking to catch signature changes early.
var _ inventory.Uninstaller = (*Uninstaller)(nil)
