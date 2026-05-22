package gemini

import (
	"context"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Builder is the Gemini CLI implementation of source.Builder. Per-kind
// dispatch is delegated to source.RendererRegistry; the kind-specific
// procedures live in skillRenderer, agentRenderer, promptRenderer (TOML),
// and ruleAggregator within this package.
type Builder struct {
	registry *source.RendererRegistry
}

// NewBuilder constructs a Builder with the Gemini-specific renderers and
// rule aggregator pre-registered.
func NewBuilder() *Builder {
	r := source.NewRendererRegistry(Target)
	r.Register(skillRenderer{})
	r.Register(agentRenderer{})
	r.Register(promptRenderer{})
	r.RegisterRuleAggregator(ruleAggregator{})
	return &Builder{registry: r}
}

// Target returns the distribution target handled by this Builder.
func (b *Builder) Target() source.Target { return Target }

// Build delegates to the registry.
func (b *Builder) Build(ctx context.Context, pack *source.Pack) ([]source.Artifact, error) {
	return b.registry.Build(ctx, pack)
}

var _ source.Builder = (*Builder)(nil)
