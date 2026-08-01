package claude

import (
	"context"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Builder is the Claude Code implementation of source.Builder.
//
// Build delegates the per-kind dispatch to source.RendererRegistry; the
// per-kind logic lives in skillRenderer and agentRenderer in this package.
// Builder itself contains no
// switch-on-Kind branching.
type Builder struct {
	registry *source.RendererRegistry
}

// NewBuilder constructs a Builder with the Claude-specific renderers.
func NewBuilder() *Builder {
	r := source.NewRendererRegistry(Target)
	r.Register(skillRenderer{})
	r.Register(agentRenderer{})
	return &Builder{registry: r}
}

// Target returns the distribution target handled by this Builder.
func (b *Builder) Target() source.Target { return Target }

// Build delegates to the registry.
func (b *Builder) Build(ctx context.Context, pack *source.Pack) ([]source.Artifact, error) {
	return b.registry.Build(ctx, pack)
}

// Compile-time interface assertion.
var _ source.Builder = (*Builder)(nil)
