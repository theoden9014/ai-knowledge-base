package source

import "context"

// Builder converts a loaded Pack into target-specific Artifacts. Each
// distribution target (Claude / Codex / Gemini ...) provides its own Builder
// implementation under internal/distribution/<target>.
//
// Builder receives the whole Pack rather than individual Entries so that an
// implementation can emit auxiliary files alongside a primary artifact.
//
// Implementations should start from pack.EntriesFor(b.Target()) instead of
// iterating Pack.Entries directly; that helper applies the single
// per-target / DefaultTools resolution rule defined on Pack and frees
// Builder implementations from re-deriving "enabled for me" themselves.
type Builder interface {
	// Target returns the Target this builder produces artifacts for.
	Target() Target

	// Build converts the given Pack into zero or more Artifacts. The
	// returned slice is owned by the caller. Implementations must honor
	// ctx for cancellation.
	Build(ctx context.Context, pack *Pack) ([]Artifact, error)
}
