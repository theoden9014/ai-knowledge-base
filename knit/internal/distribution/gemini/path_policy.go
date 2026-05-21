package gemini

import (
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// pathPolicy is the inventory.PathPolicy implementation for the Gemini CLI
// distribution target.
//
// Accepted artifact paths:
//   - GEMINI.md
//   - skills/<...>
//   - agents/<...>
//   - commands/<...>
type pathPolicy struct{}

func newPathPolicy() pathPolicy { return pathPolicy{} }

func (pathPolicy) Target() source.Target { return Target }

func (pathPolicy) Validate(p source.ArtifactPath) error {
	if p.IsZero() {
		return source.ErrInvalidArtifactPath
	}
	full := p.String()
	switch p.TopSegment() {
	case "GEMINI.md":
		if full != "GEMINI.md" {
			return source.ErrInvalidArtifactPath
		}
		return nil
	case "skills", "agents", "commands":
		if full == p.TopSegment() {
			return source.ErrInvalidArtifactPath
		}
		return nil
	default:
		return source.ErrInvalidArtifactPath
	}
}

var _ inventory.PathPolicy = pathPolicy{}

// buildResolver wires the Gemini PathPolicy with the user / project
// Inventory roots and returns a PathResolver.
func buildResolver(userRoot, projectRoot string) (*inventory.PathResolver, pathPolicy) {
	policy := newPathPolicy()
	uRoot, err := inventory.NewInventoryRoot(userRoot)
	if err != nil {
		panic(fmt.Errorf("gemini: invalid user root %q: %w", userRoot, err))
	}
	var pRoot inventory.InventoryRoot
	if projectRoot != "" {
		pRoot, err = inventory.NewInventoryRoot(projectRoot)
		if err != nil {
			panic(fmt.Errorf("gemini: invalid project root %q: %w", projectRoot, err))
		}
	}
	roots, err := inventory.NewInventoryRoots(uRoot, pRoot)
	if err != nil {
		panic(fmt.Errorf("gemini: invalid inventory roots: %w", err))
	}
	resolver, err := inventory.NewPathResolver(policy, roots)
	if err != nil {
		panic(fmt.Errorf("gemini: build path resolver: %w", err))
	}
	return resolver, policy
}
