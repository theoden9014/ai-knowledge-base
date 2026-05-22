package codex

import (
	"fmt"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// pathPolicy is the inventory.PathPolicy implementation for the Codex CLI
// distribution target. It encodes the per-kind directory and file-name
// conventions documented in docs/codex-target.md.
//
// Accepted artifact paths:
//   - AGENTS.md
//   - skills/<name>/SKILL.md
//   - agents/<name>.toml
//   - prompts/<name>.md (no subdirectories)
type pathPolicy struct{}

func newPathPolicy() pathPolicy { return pathPolicy{} }

// Target returns the Codex target identifier.
func (pathPolicy) Target() source.Target { return Target }

// codexPathRules indexes Codex's per-top-segment path validators.
// Adding a new accepted top segment is one entry.
var codexPathRules = map[string]func(source.ArtifactPath) error{
	"AGENTS.md": validateAgentsMd,
	"skills":    validateSkillPath,
	"agents":    validateAgentTomlPath,
	"prompts":   validatePromptPath,
}

// Validate dispatches p to the per-top-segment rule.
func (pathPolicy) Validate(p source.ArtifactPath) error {
	if p.IsZero() {
		return source.ErrInvalidArtifactPath
	}
	rule, ok := codexPathRules[p.TopSegment()]
	if !ok {
		return source.ErrInvalidArtifactPath
	}
	return rule(p)
}

// validateAgentsMd accepts the single flat file "AGENTS.md" and nothing
// else under that prefix.
func validateAgentsMd(p source.ArtifactPath) error {
	if p.String() != "AGENTS.md" {
		return source.ErrInvalidArtifactPath
	}
	return nil
}

// validateSkillPath requires `skills/<name>/SKILL.md` exactly.
func validateSkillPath(p source.ArtifactPath) error {
	parts := strings.Split(p.String(), "/")
	if len(parts) != 3 || parts[1] == "" || parts[2] != "SKILL.md" {
		return source.ErrInvalidArtifactPath
	}
	return nil
}

// validateAgentTomlPath requires `agents/<name>.toml` (no subdirs).
func validateAgentTomlPath(p source.ArtifactPath) error {
	parts := strings.Split(p.String(), "/")
	if len(parts) != 2 || !strings.HasSuffix(parts[1], ".toml") || parts[1] == ".toml" {
		return source.ErrInvalidArtifactPath
	}
	return nil
}

// validatePromptPath requires `prompts/<name>.md` (no subdirs; Codex
// does not allow subdirectories under prompts/).
func validatePromptPath(p source.ArtifactPath) error {
	parts := strings.Split(p.String(), "/")
	if len(parts) != 2 || !strings.HasSuffix(parts[1], ".md") || parts[1] == ".md" {
		return source.ErrInvalidArtifactPath
	}
	return nil
}

var _ inventory.PathPolicy = pathPolicy{}

// buildResolver wires Codex's PathPolicy with the user / project Inventory
// roots and returns a PathResolver. userRoot must be a non-empty absolute
// path; empty projectRoot keeps ScopeProject operations returning
// ErrProjectRootNotConfigured at call time.
func buildResolver(userRoot, projectRoot string) (*inventory.PathResolver, error) {
	uRoot, err := inventory.NewInventoryRoot(userRoot)
	if err != nil {
		return nil, fmt.Errorf("codex: invalid user root %q: %w", userRoot, err)
	}
	var pRoot inventory.InventoryRoot
	if projectRoot != "" {
		pRoot, err = inventory.NewInventoryRoot(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("codex: invalid project root %q: %w", projectRoot, err)
		}
	}
	roots, err := inventory.NewInventoryRoots(uRoot, pRoot)
	if err != nil {
		return nil, fmt.Errorf("codex: invalid inventory roots: %w", err)
	}
	resolver, err := inventory.NewPathResolver(newPathPolicy(), roots)
	if err != nil {
		return nil, fmt.Errorf("codex: build path resolver: %w", err)
	}
	return resolver, nil
}
