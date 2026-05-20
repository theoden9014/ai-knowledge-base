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

// Installer is the Claude Code implementation of inventory.Installer.
//
// The handled target is always [Target]. It encapsulates destination-path
// resolution and delegates label persistence to inventory.LabelStore. This
// type is responsible for writing artifact files and orchestrating the
// transaction with LabelStore (write artifact -> persist label, with rollback
// on failure).
//
// By sharing the same pathResolver / LabelStore with Uninstaller and Lister,
// this type keeps both the physical Inventory representation and the label
// persistence mechanism centralized across the three roles.
type Installer struct {
	resolver *pathResolver
	labels   inventory.LabelStore
}

// NewInstaller constructs an Installer from Inventory roots and a LabelStore.
//
// Arguments:
//   - userRoot   : absolute Inventory root path for ScopeUser. Usually
//     "$HOME/.claude". Behavior is undefined if an empty string is passed.
//   - projectRoot: absolute Inventory root path for ScopeProject. Usually
//     "<project>/.claude". If empty, ScopeProject operations return
//     ErrProjectRootNotConfigured.
//   - labels     : [inventory.LabelStore] used for label persistence. The
//     store's labelsRoot configuration is the CLI (factory) layer's
//     responsibility; this Installer does not validate that consistency.
func NewInstaller(userRoot, projectRoot string, labels inventory.LabelStore) *Installer {
	return &Installer{
		resolver: newPathResolver(userRoot, projectRoot),
		labels:   labels,
	}
}

// Target returns the distribution target handled by this Installer. It always returns [Target].
func (i *Installer) Target() source.Target {
	return Target
}

// Install places the given Artifact into the Inventory for the specified Scope,
// persists a label via LabelStore, and returns the resulting Installation.
//
// Error precedence (when multiple conditions hold at once):
//  1. inventory.ErrTargetMismatch  (artifact.Target ≠ [Target])
//  2. inventory.ErrInvalidScope    (scope is neither ScopeUser nor ScopeProject)
//  3. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
//  4. ErrInvalidArtifactPath       (artifact.Path violates the convention)
//  5. inventory.ErrAlreadyInstalled (the label already exists for this id)
//  6. ErrUnmanagedArtifactExists    (no label exists, but an artifact file already exists at the destination)
//
// Existing-object policy:
//   - inventory.ErrAlreadyInstalled is determined by the existence of the label
//     (= the same Installation previously placed by knit). If re-installation is
//     needed, remove it first with Uninstaller.
//   - If no label exists but the artifact file already exists at the
//     destination, it is treated as a pre-existing file not managed by knit
//     and installation is rejected with ErrUnmanagedArtifactExists.
func (i *Installer) Install(ctx context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	if artifact.Target != Target {
		return inventory.Installation{}, fmt.Errorf("%w: artifact.Target=%q installer.Target=%q",
			inventory.ErrTargetMismatch, artifact.Target, Target)
	}
	r, err := i.resolver.resolve(scope)
	if err != nil {
		return inventory.Installation{}, err
	}
	absArtifactPath, err := r.ResolveArtifactPath(artifact.Path)
	if err != nil {
		return inventory.Installation{}, err
	}
	id := i.resolver.installationID(artifact.Path)

	// Preflight: an existing label means a knit-managed installation.
	if _, gErr := i.labels.Get(ctx, scope, id); gErr == nil {
		return inventory.Installation{}, inventory.ErrAlreadyInstalled
	} else if !errors.Is(gErr, inventory.ErrInstallationNotFound) {
		return inventory.Installation{}, fmt.Errorf("claude: probe label: %w", gErr)
	}
	// No label, but an artifact file already exists -> unmanaged file.
	if _, aErr := os.Stat(absArtifactPath); aErr == nil {
		return inventory.Installation{}, fmt.Errorf("%w: path=%s", ErrUnmanagedArtifactExists, artifact.Path)
	} else if !errors.Is(aErr, fs.ErrNotExist) {
		return inventory.Installation{}, fmt.Errorf("claude: stat artifact: %w", aErr)
	}

	// Write the artifact file.
	if err := os.MkdirAll(filepath.Dir(absArtifactPath), 0o755); err != nil {
		return inventory.Installation{}, fmt.Errorf("claude: create artifact dir: %w", err)
	}
	mode := artifact.Mode
	if mode == 0 {
		mode = 0o644
	}
	if err := os.WriteFile(absArtifactPath, artifact.Content, mode); err != nil {
		return inventory.Installation{}, fmt.Errorf("claude: write artifact: %w", err)
	}

	// Persist the label.
	data := inventory.LabelData{
		SchemaVersion:  1,
		ArtifactPath:   artifact.Path,
		SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
	}
	if err := i.labels.Set(ctx, scope, id, data); err != nil {
		// Roll back the artifact file so we do not leave an unmanaged leftover.
		_ = os.Remove(absArtifactPath)
		if errors.Is(err, inventory.ErrLabelAlreadyExists) {
			return inventory.Installation{}, inventory.ErrAlreadyInstalled
		}
		return inventory.Installation{}, fmt.Errorf("claude: persist label: %w", err)
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

// Helper for static type checking to catch signature changes early.
var _ inventory.Installer = (*Installer)(nil)
