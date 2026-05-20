package codex

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

// pathResolver owns the Codex CLI inventory layout rules.
//
// It centralizes two mappings: `(Scope) -> inventory root` and
// `(Scope, Artifact.Path) -> absolute path`. Label persistence is delegated to
// inventory.LabelStore configured at the cli layer.
//
// This type is an internal implementation detail. Installer, Uninstaller, and
// Lister share the same *pathResolver. The target is fixed to codex.Target.
type pathResolver struct {
	// userRoot is the absolute inventory root for ScopeUser, for example
	// "/Users/<name>/.codex" (or $CODEX_HOME).
	userRoot string

	// projectRoot is the absolute inventory root for ScopeProject, for example
	// "<project>/.codex". When it is empty, ScopeProject operations return
	// ErrProjectRootNotConfigured.
	projectRoot string
}

// newPathResolver constructs a pathResolver from the user / project inventory
// roots.
func newPathResolver(userRoot, projectRoot string) *pathResolver {
	return &pathResolver{
		userRoot:    userRoot,
		projectRoot: projectRoot,
	}
}

// root returns the absolute inventory root for the given Scope.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (scope is neither ScopeUser nor ScopeProject)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
func (r *pathResolver) root(scope inventory.Scope) (string, error) {
	switch scope {
	case inventory.ScopeUser:
		return r.userRoot, nil
	case inventory.ScopeProject:
		if r.projectRoot == "" {
			return "", ErrProjectRootNotConfigured
		}
		return r.projectRoot, nil
	default:
		return "", inventory.ErrInvalidScope
	}
}

// validateArtifactPath validates Artifact.Path against the Codex CLI directory
// conventions.
func validateArtifactPath(p string) error {
	if p == "" {
		return ErrInvalidArtifactPath
	}
	if path.IsAbs(p) {
		return ErrInvalidArtifactPath
	}
	cleaned := path.Clean(p)
	if cleaned != p {
		return ErrInvalidArtifactPath
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ErrInvalidArtifactPath
	}
	switch {
	case cleaned == "AGENTS.md":
		return nil
	case strings.HasPrefix(cleaned, "skills/"):
		parts := strings.Split(cleaned, "/")
		if len(parts) != 3 || parts[2] != "SKILL.md" || parts[1] == "" {
			return ErrInvalidArtifactPath
		}
		return nil
	case strings.HasPrefix(cleaned, "agents/"):
		parts := strings.Split(cleaned, "/")
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ".toml") || parts[1] == ".toml" {
			return ErrInvalidArtifactPath
		}
		return nil
	case strings.HasPrefix(cleaned, "prompts/"):
		parts := strings.Split(cleaned, "/")
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ".md") || parts[1] == ".md" {
			return ErrInvalidArtifactPath
		}
		return nil
	default:
		return ErrInvalidArtifactPath
	}
}

// resolveArtifactPath converts artifactPath (relative to the inventory root)
// into an absolute path for the given Scope.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (invalid scope)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
//  3. ErrInvalidArtifactPath       (path violates the conventions)
func (r *pathResolver) resolveArtifactPath(scope inventory.Scope, artifactPath string) (string, error) {
	rootPath, err := r.root(scope)
	if err != nil {
		return "", err
	}
	if err := validateArtifactPath(artifactPath); err != nil {
		return "", err
	}
	return filepath.Join(rootPath, filepath.FromSlash(artifactPath)), nil
}

// installationID builds the unique identifier for an installation within the
// inventory by reusing Artifact.Path verbatim.
func (r *pathResolver) installationID(artifactPath string) inventory.InstallationID {
	return inventory.InstallationID(artifactPath)
}

// resolved is a value object representing a successfully resolved scope.
// Scope validation and inventory-root selection happen once in resolve(scope).
type resolved struct {
	scope inventory.Scope
	root  string
}

// resolve validates scope once and returns a resolved.
func (r *pathResolver) resolve(scope inventory.Scope) (*resolved, error) {
	rootPath, err := r.root(scope)
	if err != nil {
		return nil, err
	}
	return &resolved{scope: scope, root: rootPath}, nil
}

// Scope returns the Scope used to build this resolved.
func (s *resolved) Scope() inventory.Scope { return s.scope }

// Root returns the absolute inventory root path.
func (s *resolved) Root() string { return s.root }

// artifactPath converts artifactPath into an absolute path rooted at
// resolved.root.
func (s *resolved) artifactPath(artifactPath string) (string, error) {
	if err := validateArtifactPath(artifactPath); err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(artifactPath)), nil
}
