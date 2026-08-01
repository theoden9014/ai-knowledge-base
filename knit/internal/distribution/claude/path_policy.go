package claude

import (
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// pathPolicy is the inventory.PathPolicy implementation for the Claude
// distribution target. It encodes the set of legal top-level segments and
// the single flat file allowed at the inventory root.
//
// Accepted artifact paths:
//   - skills/<...>
//   - agents/<...>
type pathPolicy struct{}

// newPathPolicy returns the singleton-style PathPolicy for Claude. The
// type carries no state; a fresh value is fine.
func newPathPolicy() pathPolicy { return pathPolicy{} }

// Target returns the Claude target identifier.
func (pathPolicy) Target() source.Target { return Target }

// Validate applies Claude's path rules to p.
func (pathPolicy) Validate(p source.ArtifactPath) error {
	if p.IsZero() {
		return source.ErrInvalidArtifactPath
	}
	top := p.TopSegment()
	switch top {
	case "skills", "agents":
		if p.String() == top {
			// Bare top segment with no child is not a valid artifact path.
			return source.ErrInvalidArtifactPath
		}
		return nil
	default:
		return source.ErrInvalidArtifactPath
	}
}

// Compile-time interface assertion.
var _ inventory.PathPolicy = pathPolicy{}

// buildResolver assembles a PathResolver for Claude from the user and
// project Inventory root strings. userRoot must be a non-empty absolute
// path; empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time. Validation failures are
// returned as errors so the CLI layer can surface them instead of
// panicking deep inside a constructor.
func buildResolver(userRoot, projectRoot string) (*inventory.PathResolver, error) {
	uRoot, err := inventory.NewInventoryRoot(userRoot)
	if err != nil {
		return nil, fmt.Errorf("claude: invalid user root %q: %w", userRoot, err)
	}
	var pRoot inventory.InventoryRoot
	if projectRoot != "" {
		pRoot, err = inventory.NewInventoryRoot(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("claude: invalid project root %q: %w", projectRoot, err)
		}
	}
	roots, err := inventory.NewInventoryRoots(uRoot, pRoot)
	if err != nil {
		return nil, fmt.Errorf("claude: invalid inventory roots: %w", err)
	}
	resolver, err := inventory.NewPathResolver(newPathPolicy(), roots)
	if err != nil {
		return nil, fmt.Errorf("claude: build path resolver: %w", err)
	}
	return resolver, nil
}
