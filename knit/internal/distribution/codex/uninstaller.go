package codex

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

// Uninstaller is the Codex CLI implementation of inventory.Uninstaller.
//
// It always serves [Target]. Sharing the same pathResolver / LabelStore as
// Installer and Lister keeps the physical inventory representation
// centralized.
//
// Scope is not passed as a separate argument; per the inventory.Uninstaller
// contract, Installation.Label.Scope is treated as the source of truth.
type Uninstaller struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewUninstaller constructs an Uninstaller from the inventory roots and a
// LabelStore. See [NewInstaller] for the meaning of each argument.
//
// If projectRoot is empty, Uninstall calls for installations labeled with
// ScopeProject return ErrProjectRootNotConfigured.
func NewUninstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Uninstaller {
	return &Uninstaller{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Uninstaller. It always
// returns [Target].
func (u *Uninstaller) Target() source.Target {
	return Target
}

// Uninstall removes the specified Installation from the inventory.
//
// Error precedence:
//  1. inventory.ErrTargetMismatch   (Label.Target ≠ [Target])
//  2. inventory.ErrInvalidScope     (Label.Scope is invalid)
//  3. ErrProjectRootNotConfigured   (scope is ScopeProject but projectRoot is unset)
//  4. inventory.ErrInstallationNotFound (label does not exist)
//
// Removal applies to both the artifact file and its label. For skills stored
// as `skills/<name>/SKILL.md`, the parent directory is removed when it becomes
// empty, but it is preserved when unmanaged support files such as scripts/ or
// references/ remain.
func (u *Uninstaller) Uninstall(ctx context.Context, installation inventory.Installation) error {
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
	absPath, err := rs.artifactPath(artifactPath)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("codex: remove artifact: %w", err)
	}
	// Best-effort empty-dir cleanup: only succeeds when the parent is empty,
	// so unmanaged sibling files keep the directory in place.
	_ = os.Remove(filepath.Dir(absPath))

	if err := u.labels.Delete(ctx, scope, id); err != nil {
		return fmt.Errorf("codex: remove label: %w", err)
	}
	return nil
}

// Static interface check for early detection of signature drift.
var _ inventory.Uninstaller = (*Uninstaller)(nil)
