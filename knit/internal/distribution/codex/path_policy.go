package codex

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// pathPolicy is the inventory.PathPolicy implementation for the Codex CLI
// distribution target. It encodes the per-kind directory and file-name
// conventions documented in docs/codex-target.md.
//
// Accepted artifact paths:
//   - skills/<name>/SKILL.md and any sibling under skills/<name>/...
//   - agents/<name>.toml
type pathPolicy struct{}

func newPathPolicy() pathPolicy { return pathPolicy{} }

// Target returns the Codex target identifier.
func (pathPolicy) Target() source.Target { return Target }

// codexPathRules indexes Codex's per-top-segment path validators.
// Adding a new accepted top segment is one entry.
var codexPathRules = map[string]func(source.ArtifactPath) error{
	"skills": validateSkillPath,
	"agents": validateAgentTomlPath,
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

// validateSkillPath accepts `skills/<name>/<...>`. The name segment must
// be non-empty and there must be at least one more segment beneath it so
// the bare directory itself is rejected. Sub-paths beyond SKILL.md are
// allowed so a skill can carry sibling assets (scripts/, references/, ...).
func validateSkillPath(p source.ArtifactPath) error {
	parts := strings.Split(p.String(), "/")
	if len(parts) < 3 || parts[1] == "" {
		return source.ErrInvalidArtifactPath
	}
	if slices.Contains(parts[2:], "") {
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

var _ inventory.PathPolicy = pathPolicy{}

// Roots identifies Codex's physical roots. Skills follow the shared
// Agent Skills convention under .agents; custom agents remain under .codex.
type Roots struct {
	UserSkills    string
	ProjectSkills string
	UserAgents    string
	ProjectAgents string
}

// DefaultRoots derives Codex roots from the user's home, project root, and
// optional CODEX_HOME override.
func DefaultRoots(userBase, projectRoot, codexHome string) Roots {
	var roots Roots
	if userBase != "" {
		roots.UserSkills = filepath.Join(userBase, ".agents")
		roots.UserAgents = filepath.Join(userBase, ".codex")
	}
	if codexHome != "" {
		roots.UserAgents = codexHome
	}
	if projectRoot != "" {
		roots.ProjectSkills = filepath.Join(projectRoot, ".agents")
		roots.ProjectAgents = filepath.Join(projectRoot, ".codex")
	}
	return roots
}

type artifactResolver struct {
	skills *inventory.PathResolver
	agents *inventory.PathResolver
}

func (r *artifactResolver) Target() source.Target { return Target }

func (r *artifactResolver) ValidateScope(scope inventory.Scope) error {
	if err := r.skills.ValidateScope(scope); err != nil {
		return err
	}
	return r.agents.ValidateScope(scope)
}

func (r *artifactResolver) Resolve(scope inventory.Scope, p source.ArtifactPath) (inventory.AbsoluteArtifactPath, error) {
	switch p.TopSegment() {
	case "skills":
		return r.skills.Resolve(scope, p)
	case "agents":
		return r.agents.Resolve(scope, p)
	default:
		return inventory.AbsoluteArtifactPath{}, source.ErrInvalidArtifactPath
	}
}

func buildResolver(roots Roots) (inventory.ArtifactResolver, error) {
	if (roots.ProjectSkills == "") != (roots.ProjectAgents == "") {
		return nil, fmt.Errorf("codex: project skill and agent roots must be configured together")
	}
	skills, err := buildPathResolver(roots.UserSkills, roots.ProjectSkills)
	if err != nil {
		return nil, fmt.Errorf("codex: build skills resolver: %w", err)
	}
	agents, err := buildPathResolver(roots.UserAgents, roots.ProjectAgents)
	if err != nil {
		return nil, fmt.Errorf("codex: build agents resolver: %w", err)
	}
	return &artifactResolver{skills: skills, agents: agents}, nil
}

func buildPathResolver(userRoot, projectRoot string) (*inventory.PathResolver, error) {
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

var _ inventory.ArtifactResolver = (*artifactResolver)(nil)
