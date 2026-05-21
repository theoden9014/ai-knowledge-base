package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// LabelSchemaVersion is the LabelData.SchemaVersion value written by
// TransactionalInstaller. Older readers may still understand prior versions.
const LabelSchemaVersion = 1

// TransactionalInstaller is the target-neutral implementation of the
// inventory.Installer contract. It binds an ArtifactStore, a LabelStore,
// and a PathResolver at construction time so the per-Install path becomes
// the pure two-step (artifact write + label set) transaction described in
// refactoring-interface-design.md.
type TransactionalInstaller struct {
	store    ArtifactStore
	labels   LabelStore
	resolver *PathResolver
}

// NewTransactionalInstaller validates dependencies and returns an Installer.
func NewTransactionalInstaller(store ArtifactStore, labels LabelStore, resolver *PathResolver) (*TransactionalInstaller, error) {
	if store == nil {
		return nil, errors.New("inventory: transactional installer requires artifact store")
	}
	if labels == nil {
		return nil, errors.New("inventory: transactional installer requires label store")
	}
	if resolver == nil {
		return nil, errors.New("inventory: transactional installer requires path resolver")
	}
	return &TransactionalInstaller{store: store, labels: labels, resolver: resolver}, nil
}

// Target returns the distribution target bound at construction time.
func (i *TransactionalInstaller) Target() source.Target { return i.resolver.Target() }

// Install places artifact under scope.
//
// The error precedence is fixed by refactoring-interface-design.md:
//  1. Target mismatch (ErrTargetMismatch)
//  2. ArtifactPath structural invariant (ErrInvalidArtifactPath)
//  3. Scope and root resolution (ErrInvalidScope, ErrProjectRootNotConfigured)
//  4. Preflight 2x2 (ErrAlreadyInstalled or ErrUnmanagedArtifactExists)
//  5. Write + Label.Set (with rollback on label failure)
func (i *TransactionalInstaller) Install(ctx context.Context, scope Scope, artifact source.Artifact) (Installation, error) {
	if err := ctx.Err(); err != nil {
		return Installation{}, err
	}
	if artifact.Target != i.Target() {
		return Installation{}, ErrTargetMismatch
	}
	// Validate scope (and project-root configuration) before parsing the
	// artifact path so the documented error precedence (scope -> path) is
	// preserved.
	if _, err := i.resolver.ResolveRoot(scope); err != nil {
		return Installation{}, err
	}
	rel, err := source.NewArtifactPath(artifact.Path)
	if err != nil {
		return Installation{}, err
	}
	abs, err := i.resolver.Resolve(scope, rel)
	if err != nil {
		return Installation{}, err
	}
	id, err := NewInstallationIDFromArtifactPath(rel)
	if err != nil {
		return Installation{}, err
	}

	labelPresent, err := i.labelExists(ctx, scope, id)
	if err != nil {
		return Installation{}, err
	}
	filePresent, err := i.store.Exists(ctx, abs)
	if err != nil {
		return Installation{}, fmt.Errorf("inventory: preflight exists: %w", err)
	}
	switch {
	case labelPresent:
		// Both (Label-yes, File-yes) and (Label-yes, File-no) collapse to
		// ErrAlreadyInstalled: the caller must Uninstall first.
		return Installation{}, ErrAlreadyInstalled
	case filePresent:
		return Installation{}, ErrUnmanagedArtifactExists
	}

	if err := i.store.Write(ctx, abs, artifact.Content, artifact.Mode); err != nil {
		return Installation{}, fmt.Errorf("inventory: write artifact: %w", err)
	}
	data := LabelData{
		SchemaVersion:  LabelSchemaVersion,
		ArtifactPath:   rel.String(),
		SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
	}
	if err := i.labels.Set(ctx, scope, id, data); err != nil {
		// Best-effort rollback: remove the artifact we just wrote so the
		// inventory invariant (label-yes <=> file-yes) is preserved.
		_ = i.store.Remove(ctx, abs)
		return Installation{}, fmt.Errorf("inventory: set label: %w", err)
	}

	return Installation{
		ID:    id,
		Label: Label{Target: i.Target(), Scope: scope},
		Provenance: Provenance{
			SourceEntryIDs: append([]string(nil), artifact.SourceEntryIDs...),
		},
		Artifact: artifact,
	}, nil
}

// labelExists reports whether a label exists for (scope, id). It folds
// ErrInstallationNotFound into "not present" and surfaces every other error
// untouched so the Install caller can react.
func (i *TransactionalInstaller) labelExists(ctx context.Context, scope Scope, id InstallationID) (bool, error) {
	_, err := i.labels.Get(ctx, scope, id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrInstallationNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("inventory: preflight label: %w", err)
}

// Static type check.
var _ Installer = (*TransactionalInstaller)(nil)
