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
	resolver *PathResolver
}

// NewTransactionalUninstaller validates dependencies and returns an
// Uninstaller.
func NewTransactionalUninstaller(store ArtifactStore, labels LabelStore, resolver *PathResolver) (*TransactionalUninstaller, error) {
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
	rel, err := source.NewArtifactPath(installation.Artifact.Path)
	if err != nil {
		return err
	}
	abs, err := u.resolver.Resolve(scope, rel)
	if err != nil {
		return err
	}
	id, err := NewInstallationIDFromArtifactPath(rel)
	if err != nil {
		return err
	}
	root, err := u.resolver.ResolveRoot(scope)
	if err != nil {
		return err
	}

	// Confirm the label exists; that is the canonical "is the entity in
	// this Inventory" signal per refactoring-conceptual-model.md.
	if _, err := u.labels.Get(ctx, scope, id); err != nil {
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
		if err := u.store.PruneAncestorsWithin(ctx, abs, root); err != nil && !errors.Is(err, ErrPruneBoundaryViolation) {
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
