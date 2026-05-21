package inventory

import (
	"errors"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// PathResolver composes a PathPolicy with an InventoryRoots pair to turn
// (Scope, ArtifactPath) into an AbsoluteArtifactPath. It captures the
// scope-to-root dispatch and the target-specific path rules in one place
// so Transactional* implementations do not need to repeat either.
type PathResolver struct {
	policy PathPolicy
	roots  InventoryRoots
}

// NewPathResolver constructs a PathResolver. policy must be non-nil. The
// roots pair may have a zero project root (in which case scope=project
// requests will fail with ErrProjectRootNotConfigured at call time).
func NewPathResolver(policy PathPolicy, roots InventoryRoots) (*PathResolver, error) {
	if policy == nil {
		return nil, errors.New("inventory: path resolver requires policy")
	}
	if policy.Target() == "" {
		return nil, errors.New("inventory: path resolver requires policy with non-empty target")
	}
	return &PathResolver{policy: policy, roots: roots}, nil
}

// Target returns the Target this resolver belongs to.
func (r *PathResolver) Target() source.Target { return r.policy.Target() }

// Resolve validates scope first, then p, and finally joins them. The
// scope-before-policy ordering matches the error precedence documented in
// refactoring-interface-design.md (ErrInvalidScope and
// ErrProjectRootNotConfigured precede ErrInvalidArtifactPath).
func (r *PathResolver) Resolve(scope Scope, p source.ArtifactPath) (AbsoluteArtifactPath, error) {
	root, err := r.roots.For(scope)
	if err != nil {
		return AbsoluteArtifactPath{}, err
	}
	if err := r.policy.Validate(p); err != nil {
		return AbsoluteArtifactPath{}, err
	}
	return root.Join(p)
}

// ResolveRoot returns the InventoryRoot for scope without resolving any
// individual artifact path. Useful for boundary-aware operations such as
// PruneAncestorsWithin.
func (r *PathResolver) ResolveRoot(scope Scope) (InventoryRoot, error) {
	return r.roots.For(scope)
}
