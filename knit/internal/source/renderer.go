package source

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnsupportedKind is returned by RendererRegistry.Build when an entry's
// Kind has no renderer registered for the target.
var ErrUnsupportedKind = errors.New("source: unsupported kind")

// KindRenderer converts a single Entry into one or more target-specific
// Artifacts. Implementations live alongside each distribution target
// (claude / codex / gemini). The interface itself stays target-agnostic
// so the per-kind dispatch logic can be shared across targets.
//
// Render must return a non-empty slice on success: returning (nil, nil) or
// an empty slice with a nil error is a contract violation, because an
// entry that should produce no artifacts (for example because it is not
// enabled for this target) is filtered out by Pack.EntriesFor before
// reaching the renderer.
//
// When a renderer emits multiple artifacts (e.g. a skill body plus its
// sibling assets), every returned Artifact must carry the same
// SourceEntryIDs slice so downstream Provenance.BelongsToPack lookups can
// treat the set as a unit.
type KindRenderer interface {
	// Kind reports the entry kind this renderer handles.
	Kind() Kind

	// Render produces one or more Artifacts for entry. The pack argument
	// is provided for renderers that need pack-level metadata (most do
	// not).
	Render(entry *Entry, pack *Pack) ([]Artifact, error)
}

// RendererRegistry is the per-target table that maps Kind to the right
// renderer. Builder implementations construct a Registry at
// initialization time, then call Build for each pack.
type RendererRegistry struct {
	target    Target
	renderers map[Kind]KindRenderer
}

// NewRendererRegistry returns an empty Registry bound to target.
func NewRendererRegistry(target Target) *RendererRegistry {
	return &RendererRegistry{
		target:    target,
		renderers: make(map[Kind]KindRenderer),
	}
}

// Register adds renderer to the table. Registering the same Kind again
// overwrites the previous renderer.
func (r *RendererRegistry) Register(renderer KindRenderer) {
	r.renderers[renderer.Kind()] = renderer
}

// Target returns the target this registry serves.
func (r *RendererRegistry) Target() Target { return r.target }

// Build walks pack.EntriesFor(target) and produces artifacts using the
// registered renderers.
//
// A nil pack returns (nil, nil) so the contract matches across every
// target Builder rather than depending on each implementation to guard.
func (r *RendererRegistry) Build(ctx context.Context, pack *Pack) ([]Artifact, error) {
	if pack == nil {
		return nil, nil
	}
	entries := pack.EntriesFor(r.target)
	var artifacts []Artifact
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		renderer, ok := r.renderers[e.Kind]
		if !ok {
			return nil, fmt.Errorf("%w: kind=%s target=%s", ErrUnsupportedKind, e.Kind, r.target)
		}
		arts, err := renderer.Render(e, pack)
		if err != nil {
			return nil, err
		}
		if len(arts) == 0 {
			return nil, fmt.Errorf("%w: renderer returned no artifacts: kind=%s entry=%s target=%s", ErrUnsupportedKind, e.Kind, e.ID, r.target)
		}
		artifacts = append(artifacts, arts...)
	}
	return artifacts, nil
}
