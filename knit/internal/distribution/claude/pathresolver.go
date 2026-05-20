package claude

import (
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

// pathResolver owns Claude Code Inventory placement rules.
//
// It centralizes "(Scope) -> Inventory root" and
// "(Scope, Artifact.Path) -> absolute artifact path" so that Builder /
// Installer / Uninstaller / Lister all agree on the same convention. Label
// placement lives outside this resolver: it is delegated to
// inventory.LabelStore via the SidecarLabelStore root configured at the cli
// layer.
//
// This type is internal to the package and is not exported. Installer /
// Uninstaller / Lister share the same *pathResolver. target is always fixed
// to claude.Target, so it is intentionally omitted from method parameters
// (for ISP and to prevent accidental mismatches).
type pathResolver struct {
	// userRoot is the absolute Inventory root path for ScopeUser.
	// Example: "/Users/<name>/.claude"
	userRoot string

	// projectRoot is the absolute Inventory root path for ScopeProject.
	// Example: "<project>/.claude"
	// When empty, ScopeProject-related operations return
	// ErrProjectRootNotConfigured so the case where the CLI layer could not
	// discover the project root is made explicit.
	projectRoot string
}

// validTopLevels contains the allowed first path segments for Artifact.Path.
// It follows Claude Code directory conventions.
var validTopLevels = map[string]struct{}{
	"skills":   {},
	"agents":   {},
	"commands": {},
}

// newPathResolver constructs a pathResolver from the user / project Inventory
// roots.
//
// Typical values:
//   - userRoot   : "$HOME/.claude"
//   - projectRoot: "<project>/.claude"
//
// Empty-string behavior:
//   - Passing an empty string for userRoot is undefined; the caller is
//     responsible for validating it beforehand.
//   - projectRoot may be empty. In that case, ScopeProject operations return
//     ErrProjectRootNotConfigured.
func newPathResolver(userRoot, projectRoot string) *pathResolver {
	return &pathResolver{
		userRoot:    userRoot,
		projectRoot: projectRoot,
	}
}

// root returns the absolute Inventory root path corresponding to Scope.
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

// resolved is a value object representing the result of evaluating pathResolver
// for a specific Scope. It holds the Scope evaluation result and guarantees
// that Scope is not re-evaluated during artifact path resolution.
type resolved struct {
	pr    *pathResolver
	scope inventory.Scope
	// root is the absolute Inventory root path (ScopeUser -> userRoot, ScopeProject -> projectRoot).
	root string
}

// resolve evaluates Scope once and returns the *resolved object used as the
// starting point for path resolution in Installer / Uninstaller / Lister.
//
// Error precedence is the same as root.
func (r *pathResolver) resolve(scope inventory.Scope) (*resolved, error) {
	root, err := r.root(scope)
	if err != nil {
		return nil, err
	}
	return &resolved{pr: r, scope: scope, root: root}, nil
}

// Scope returns the Scope used to create this resolved value.
func (rv *resolved) Scope() inventory.Scope { return rv.scope }

// Root returns this resolved value's absolute Inventory root path.
func (rv *resolved) Root() string { return rv.root }

// ResolveArtifactPath converts relPath (relative to the Inventory root) into
// an absolute path for this resolved value's Scope.
func (rv *resolved) ResolveArtifactPath(relPath string) (string, error) {
	if err := validateArtifactPath(relPath); err != nil {
		return "", err
	}
	return filepath.Join(rv.root, relPath), nil
}

// resolveArtifactPath is the original entry point that re-evaluates Scope and
// validates artifactPath in one call. It is retained for tests and callers
// that only need a one-shot conversion.
func (r *pathResolver) resolveArtifactPath(scope inventory.Scope, artifactPath string) (string, error) {
	root, err := r.root(scope)
	if err != nil {
		return "", err
	}
	if err := validateArtifactPath(artifactPath); err != nil {
		return "", err
	}
	return filepath.Join(root, artifactPath), nil
}

// validateArtifactPath reports whether artifactPath conforms to Claude Code
// directory conventions. Accepted shapes: "CLAUDE.md" and entries under each
// validTopLevels segment such as "skills/<...>/SKILL.md".
//
// Rejected: empty, absolute, ".." escape, or any other first segment.
func validateArtifactPath(artifactPath string) error {
	if artifactPath == "" {
		return ErrInvalidArtifactPath
	}
	if filepath.IsAbs(artifactPath) {
		return ErrInvalidArtifactPath
	}
	cleaned := filepath.Clean(artifactPath)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ErrInvalidArtifactPath
	}
	if cleaned == "CLAUDE.md" {
		return nil
	}
	idx := strings.IndexByte(cleaned, '/')
	if idx <= 0 {
		return ErrInvalidArtifactPath
	}
	if _, ok := validTopLevels[cleaned[:idx]]; !ok {
		return ErrInvalidArtifactPath
	}
	return nil
}

// installationID builds the unique identifier within the destination Inventory.
// This implementation uses Artifact.Path directly as InstallationID because it
// is unique within the combination of Inventory root and scope.
//
// Calling-order contract:
//   - This method performs no validation. Callers must call it only after
//     resolveArtifactPath / ResolveArtifactPath has succeeded.
func (r *pathResolver) installationID(artifactPath string) inventory.InstallationID {
	return inventory.InstallationID(artifactPath)
}
