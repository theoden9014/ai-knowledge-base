package gemini

import (
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

// pathResolver owns Gemini CLI Inventory placement rules.
//
// It centralizes two mappings: "(Scope) -> Inventory root" and
// "(Scope, Artifact.Path) -> absolute path". Label persistence is delegated
// to inventory.LabelStore configured at the cli layer.
//
// This type is internal to the package. Installer / Uninstaller / Lister share
// the same *pathResolver. target is always fixed to gemini.Target, so it is
// intentionally omitted from method parameters.
type pathResolver struct {
	// userRoot is the absolute Inventory root path for ScopeUser.
	// Example: "/Users/<name>/.gemini"
	userRoot string

	// projectRoot is the absolute Inventory root path for ScopeProject.
	// Example: "<project>/.gemini"
	// When empty, ScopeProject-related operations return
	// ErrProjectRootNotConfigured.
	projectRoot string
}

// resolved is a path-calculation handle with root pre-resolved for a Scope.
type resolved struct {
	scope inventory.Scope
	root  string
}

// newPathResolver constructs a pathResolver from the user / project Inventory
// roots.
func newPathResolver(userRoot, projectRoot string) *pathResolver {
	return &pathResolver{
		userRoot:    userRoot,
		projectRoot: projectRoot,
	}
}

// resolve pre-resolves root for scope and returns the [resolved] handle.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (scope is neither ScopeUser nor ScopeProject)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
func (r *pathResolver) resolve(scope inventory.Scope) (*resolved, error) {
	rootPath, err := r.root(scope)
	if err != nil {
		return nil, err
	}
	return &resolved{scope: scope, root: rootPath}, nil
}

// root returns the absolute Inventory root path for Scope.
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

// resolveArtifactPath converts Artifact.Path (relative to the Inventory root)
// into an absolute path for Scope.
//
// Error precedence:
//  1. inventory.ErrInvalidScope    (invalid scope)
//  2. ErrProjectRootNotConfigured  (scope is ScopeProject but projectRoot is unset)
//  3. ErrInvalidArtifactPath       (artifactPath violates conventions)
func (r *pathResolver) resolveArtifactPath(scope inventory.Scope, artifactPath string) (string, error) {
	rs, err := r.resolve(scope)
	if err != nil {
		return "", err
	}
	return rs.artifactPath(artifactPath)
}

// installationID builds the unique identifier for a destination within the
// Inventory by reusing Artifact.Path verbatim.
func (r *pathResolver) installationID(artifactPath string) inventory.InstallationID {
	return inventory.InstallationID(artifactPath)
}

// Scope returns the Scope captured by this resolved value.
func (rs *resolved) Scope() inventory.Scope { return rs.scope }

// Root returns the absolute Inventory root path.
func (rs *resolved) Root() string { return rs.root }

// artifactPath expands artifactPath (relative to root) into an absolute path.
//
// ErrInvalidArtifactPath is returned when artifactPath is empty / absolute /
// escapes via ".." / or violates Gemini CLI directory conventions.
func (rs *resolved) artifactPath(artifactPath string) (string, error) {
	if !isValidArtifactPath(artifactPath) {
		return "", ErrInvalidArtifactPath
	}
	return filepath.Join(rs.root, filepath.FromSlash(artifactPath)), nil
}

// isValidArtifactPath reports whether artifactPath follows Gemini CLI
// directory conventions:
//   - "GEMINI.md"
//   - "skills/<name>/SKILL.md"
//   - "agents/<name>.md"
//   - "commands/<name>.toml"
func isValidArtifactPath(p string) bool {
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return false
		}
	}
	if p == "GEMINI.md" {
		return true
	}
	switch {
	case strings.HasPrefix(p, "skills/"):
		return true
	case strings.HasPrefix(p, "agents/"):
		return true
	case strings.HasPrefix(p, "commands/"):
		return true
	}
	return false
}
