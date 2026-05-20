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

// Installer is the Codex CLI implementation of inventory.Installer.
//
// It always serves [Target] and owns destination path resolution and the
// transactional protocol between writing artifact files and persisting Labels.
// Label persistence is delegated to inventory.LabelStore so that the
// underlying backend (sidecar, xattr, etc.) can be swapped without changing
// this Installer.
type Installer struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewInstaller constructs an Installer from the inventory roots and a
// LabelStore.
//
// Arguments:
//   - userRoot   : ScopeUser inventory root absolute path, typically
//     "$HOME/.codex" (or $CODEX_HOME).
//   - projectRoot: ScopeProject inventory root absolute path, typically
//     "<project>/.codex". An empty string causes ScopeProject operations to
//     return ErrProjectRootNotConfigured.
//   - labels     : Label persistence backend (typically
//     inventory.NewSidecarLabelStore(Target, ...)), wired by the cli factory.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Installer {
	return &Installer{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Installer. It always
// returns [Target].
func (i *Installer) Target() source.Target {
	return Target
}

// Install places the given Artifact into the inventory for the specified Scope,
// persists a label via LabelStore, and returns the resulting Installation.
//
// Error precedence (highest to lowest):
//  1. inventory.ErrTargetMismatch   (artifact.Target != [Target])
//  2. inventory.ErrInvalidScope     (scope is invalid)
//  3. ErrProjectRootNotConfigured   (scope is ScopeProject but projectRoot is unset)
//  4. ErrInvalidArtifactPath        (artifact.Path violates the conventions)
//  5. inventory.ErrAlreadyInstalled (the label already exists)
//  6. ErrUnmanagedArtifactExists    (the label is missing but the destination file exists)
func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	if err := ctx.Err(); err != nil {
		return inventory.Installation{}, err
	}
	if artifact.Target != Target {
		return inventory.Installation{}, fmt.Errorf("%w: artifact.Target=%q installer.Target=%q",
			inventory.ErrTargetMismatch, artifact.Target, Target)
	}
	rs, err := i.resolver.resolve(scope)
	if err != nil {
		return inventory.Installation{}, err
	}
	absArtifactPath, err := rs.artifactPath(artifact.Path)
	if err != nil {
		return inventory.Installation{}, err
	}
	id := i.resolver.installationID(artifact.Path)

	// Preflight: an existing label means this Installation is knit-managed.
	if _, gErr := i.labels.Get(ctx, scope, id); gErr == nil {
		return inventory.Installation{}, fmt.Errorf("%w: %s", inventory.ErrAlreadyInstalled, id)
	} else if !errors.Is(gErr, inventory.ErrInstallationNotFound) {
		return inventory.Installation{}, fmt.Errorf("codex: probe label: %w", gErr)
	}
	// No label, but an artifact file already exists -> unmanaged.
	if _, aErr := os.Stat(absArtifactPath); aErr == nil {
		return inventory.Installation{}, fmt.Errorf("%w: path=%s", ErrUnmanagedArtifactExists, artifact.Path)
	} else if !errors.Is(aErr, fs.ErrNotExist) {
		return inventory.Installation{}, fmt.Errorf("codex: stat artifact: %w", aErr)
	}

	if err := os.MkdirAll(filepath.Dir(absArtifactPath), 0o755); err != nil {
		return inventory.Installation{}, fmt.Errorf("codex: create artifact dir: %w", err)
	}
	mode := artifact.Mode
	if mode == 0 {
		mode = 0o644
	}
	if err := os.WriteFile(absArtifactPath, artifact.Content, mode); err != nil {
		return inventory.Installation{}, fmt.Errorf("codex: write artifact: %w", err)
	}

	data := inventory.LabelData{
		SchemaVersion:  1,
		ArtifactPath:   artifact.Path,
		SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
	}
	if err := i.labels.Set(ctx, scope, id, data); err != nil {
		_ = os.Remove(absArtifactPath)
		if errors.Is(err, inventory.ErrLabelAlreadyExists) {
			return inventory.Installation{}, fmt.Errorf("%w: %s", inventory.ErrAlreadyInstalled, id)
		}
		return inventory.Installation{}, fmt.Errorf("codex: persist label: %w", err)
	}

	return inventory.Installation{
		ID:    id,
		Label: inventory.Label{Target: Target, Scope: scope},
		Provenance: inventory.Provenance{
			SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
		},
		Artifact: artifact,
	}, nil
}

// fileExists reports whether path exists. fs.ErrNotExist maps to (false, nil);
// other I/O errors are wrapped.
func fileExists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("codex: stat %q: %w", p, err)
}

// Static interface check for early detection of signature drift.
var _ inventory.Installer = (*Installer)(nil)
