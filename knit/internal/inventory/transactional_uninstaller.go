package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// TransactionalUninstaller is the target-neutral implementation of the
// inventory.Uninstaller contract. It mirrors TransactionalInstaller's
// dependency layout (store, labels, resolver) so the two operate as
// inverses on Inventory state.
type TransactionalUninstaller struct {
	store    ArtifactStore
	labels   LabelStore
	resolver ArtifactResolver
}

// NewTransactionalUninstaller validates dependencies and returns an
// Uninstaller.
func NewTransactionalUninstaller(store ArtifactStore, labels LabelStore, resolver ArtifactResolver) (*TransactionalUninstaller, error) {
	if store == nil {
		return nil, errors.New("inventory: transactional uninstaller requires artifact store")
	}
	if labels == nil {
		return nil, errors.New("inventory: transactional uninstaller requires label store")
	}
	if resolver == nil {
		return nil, errors.New("inventory: transactional uninstaller requires path resolver")
	}
	return &TransactionalUninstaller{store: store, labels: labels, resolver: resolver}, nil
}

// Target returns the distribution target bound at construction time.
func (u *TransactionalUninstaller) Target() source.Target { return u.resolver.Target() }

// Uninstall removes the Installation. The error precedence is:
//  1. Target mismatch (ErrTargetMismatch) on Installation.Label.Target
//  2. ArtifactPath structural invariant
//  3. Scope and root resolution
//  4. Installation not found (ErrInstallationNotFound)
//  5. File delete -> Label delete -> PruneAncestorsWithin
//
// Installation.ID selects the persisted LabelData, whose ArtifactPath is the
// authoritative deletion path. The caller-provided Artifact.Path is never
// trusted when an ID is present.
//
// Orphan-label case (Label present, file absent) is handled by skipping the
// file delete and proceeding with label deletion only.
func (u *TransactionalUninstaller) Uninstall(ctx context.Context, installation Installation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if installation.Label.Target != u.Target() {
		return ErrTargetMismatch
	}
	scope := installation.Label.Scope
	if err := u.resolver.ValidateScope(scope); err != nil {
		return err
	}
	id := installation.ID
	if id == "" {
		return ErrInvalidInstallationID
	}

	// The persisted label is authoritative for placement. Never derive the
	// deletion target from the caller's mutable Artifact.Path when an ID is
	// available.
	data, err := u.labels.Get(ctx, scope, id)
	if err != nil {
		return err
	}
	rel, err := source.NewArtifactPath(data.ArtifactPath)
	if err != nil {
		return err
	}
	persistedID, err := NewInstallationIDFromArtifactPath(rel)
	if err != nil {
		return err
	}
	if persistedID != id {
		return fmt.Errorf(
			"%w: requested=%s persisted=%s",
			ErrInstallationIdentityMismatch, id, persistedID,
		)
	}
	abs, err := u.resolver.Resolve(scope, rel)
	if err != nil {
		return err
	}

	filePresent, err := u.store.Exists(ctx, abs)
	if err != nil {
		return fmt.Errorf("inventory: exists: %w", err)
	}
	if filePresent {
		if err := u.store.Remove(ctx, abs); err != nil {
			return fmt.Errorf("inventory: remove artifact: %w", err)
		}
	}
	if err := u.labels.Delete(ctx, scope, id); err != nil {
		return fmt.Errorf("inventory: delete label: %w", err)
	}
	if filePresent {
		if err := u.store.PruneAncestorsWithin(ctx, abs, abs.Root()); err != nil && !errors.Is(err, ErrPruneBoundaryViolation) {
			// Boundary violation must never occur here because Resolve
			// already produced a path within root. Other prune errors are
			// best-effort cleanup; do not fail the uninstall over them.
			return nil
		}
	}
	return nil
}

// Static type check.
var _ Uninstaller = (*TransactionalUninstaller)(nil)
