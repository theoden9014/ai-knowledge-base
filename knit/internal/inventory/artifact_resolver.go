package inventory

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// ArtifactResolver maps a logical artifact path to its physical inventory
// location. A target may route different artifact families to different
// roots while preserving one logical installation ID namespace.
type ArtifactResolver interface {
	Target() source.Target
	// ValidateScope reports whether the scope is available for the resolver's
	// complete logical inventory. Multi-root resolvers must either configure
	// every family for a scope or reject the configuration at construction.
	ValidateScope(Scope) error
	Resolve(Scope, source.ArtifactPath) (AbsoluteArtifactPath, error)
}
