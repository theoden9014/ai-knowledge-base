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
//   - CLAUDE.md (flat, at the root)
//   - skills/<...>
//   - agents/<...>
//   - commands/<...>
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
	case "CLAUDE.md":
		if p.String() != "CLAUDE.md" {
			// CLAUDE.md must be the entire path; reject "CLAUDE.md/x".
			return source.ErrInvalidArtifactPath
		}
		return nil
	case "skills", "agents", "commands":
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
// project Inventory root strings. Empty userRoot is rejected as a
// programmer error (the CLI factory is responsible for resolving it);
// empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func buildResolver(userRoot, projectRoot string) (*inventory.PathResolver, pathPolicy) {
	policy := newPathPolicy()
	uRoot, err := inventory.NewInventoryRoot(userRoot)
	if err != nil {
		panic(fmt.Errorf("claude: invalid user root %q: %w", userRoot, err))
	}
	var pRoot inventory.InventoryRoot
	if projectRoot != "" {
		pRoot, err = inventory.NewInventoryRoot(projectRoot)
		if err != nil {
			panic(fmt.Errorf("claude: invalid project root %q: %w", projectRoot, err))
		}
	}
	roots, err := inventory.NewInventoryRoots(uRoot, pRoot)
	if err != nil {
		panic(fmt.Errorf("claude: invalid inventory roots: %w", err))
	}
	resolver, err := inventory.NewPathResolver(policy, roots)
	if err != nil {
		panic(fmt.Errorf("claude: build path resolver: %w", err))
	}
	return resolver, policy
}
