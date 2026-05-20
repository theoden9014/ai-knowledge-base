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

// Installer is the Gemini CLI implementation of inventory.Installer.
//
// The handled Target is always [Target]. It encapsulates destination path
// resolution and delegates label persistence to inventory.LabelStore.
type Installer struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewInstaller constructs an Installer from the Inventory roots and a
// LabelStore. See claude.NewInstaller for the meaning of each argument;
// gemini uses "$HOME/.gemini" / "<project>/.gemini" as inventory roots.
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

// Install places the given Artifact into the Inventory for the specified Scope,
// persists a label via LabelStore, and returns the resulting Installation.
//
// Error precedence:
//  1. inventory.ErrTargetMismatch   (artifact.Target ≠ [Target])
//  2. inventory.ErrInvalidScope     (scope is invalid)
//  3. ErrProjectRootNotConfigured   (scope is ScopeProject but projectRoot is unset)
//  4. ErrInvalidArtifactPath        (artifact.Path violates conventions)
//  5. inventory.ErrAlreadyInstalled (the label already exists)
//  6. ErrUnmanagedArtifactExists    (the label is missing but the destination file exists)
func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	if err := ctx.Err(); err != nil {
		return inventory.Installation{}, err
	}
	if artifact.Target != Target {
		return inventory.Installation{}, fmt.Errorf("%w: got %q, want %q", inventory.ErrTargetMismatch, artifact.Target, Target)
	}
	rs, err := i.resolver.resolve(scope)
	if err != nil {
		return inventory.Installation{}, err
	}
	abs, err := rs.artifactPath(artifact.Path)
	if err != nil {
		return inventory.Installation{}, err
	}
	id := i.resolver.installationID(artifact.Path)

	// Preflight: an existing label means this Installation is knit-managed.
	if _, gErr := i.labels.Get(ctx, scope, id); gErr == nil {
		return inventory.Installation{}, fmt.Errorf("%w: %s", inventory.ErrAlreadyInstalled, id)
	} else if !errors.Is(gErr, inventory.ErrInstallationNotFound) {
		return inventory.Installation{}, fmt.Errorf("gemini: probe label: %w", gErr)
	}
	// No label, but an artifact file already exists -> unmanaged.
	if _, sErr := os.Stat(abs); sErr == nil {
		return inventory.Installation{}, fmt.Errorf("%w: %s", ErrUnmanagedArtifactExists, abs)
	} else if !errors.Is(sErr, fs.ErrNotExist) {
		return inventory.Installation{}, fmt.Errorf("gemini: stat artifact: %w", sErr)
	}

	mode := artifact.Mode
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return inventory.Installation{}, fmt.Errorf("gemini: mkdir artifact parent: %w", err)
	}
	if err := os.WriteFile(abs, artifact.Content, mode); err != nil {
		return inventory.Installation{}, fmt.Errorf("gemini: write artifact: %w", err)
	}

	data := inventory.LabelData{
		SchemaVersion:  1,
		ArtifactPath:   artifact.Path,
		SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
	}
	if err := i.labels.Set(ctx, scope, id, data); err != nil {
		_ = os.Remove(abs)
		if errors.Is(err, inventory.ErrLabelAlreadyExists) {
			return inventory.Installation{}, fmt.Errorf("%w: %s", inventory.ErrAlreadyInstalled, id)
		}
		return inventory.Installation{}, fmt.Errorf("gemini: persist label: %w", err)
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
