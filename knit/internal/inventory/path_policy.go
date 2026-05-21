package inventory

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// PathPolicy is the target-specific rule set that decides whether a
// relative ArtifactPath is legal for that target's Inventory. It is a pure
// data declaration: distribution/<target> packages provide one
// implementation per target and PathResolver composes it with
// InventoryRoots to produce AbsoluteArtifactPath values.
type PathPolicy interface {
	// Target identifies which target this policy belongs to. Resolver and
	// Transactional types compare this against Artifact.Target /
	// Installation.Label.Target to detect ErrTargetMismatch.
	Target() source.Target

	// Validate reports whether p satisfies the target's path rules
	// (allowed top-level segments, allowed flat file names, and so on).
	// Returns ErrInvalidArtifactPath when p does not satisfy the rules.
	Validate(p source.ArtifactPath) error
}
