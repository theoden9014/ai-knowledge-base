package inventory

import (
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// InventoryRoot is the absolute root directory of a single Inventory (a
// (Scope, Target) pair). The zero value is invalid; construction goes
// through NewInventoryRoot which enforces non-empty and absolute.
type InventoryRoot struct {
	abs string
}

// NewInventoryRoot validates abs and returns an InventoryRoot. abs must be
// non-empty and an absolute path according to filepath.IsAbs (host-dependent).
func NewInventoryRoot(abs string) (InventoryRoot, error) {
	if abs == "" {
		return InventoryRoot{}, ErrInvalidInventoryRoot
	}
	if !filepath.IsAbs(abs) {
		return InventoryRoot{}, ErrInvalidInventoryRoot
	}
	return InventoryRoot{abs: filepath.Clean(abs)}, nil
}

// String returns the absolute path. Returns "" for the zero value.
func (r InventoryRoot) String() string { return r.abs }

// IsZero reports whether r is the zero value.
func (r InventoryRoot) IsZero() bool { return r.abs == "" }

// Join combines r with rel and returns an AbsoluteArtifactPath. Returns
// ErrArtifactPathEscape if the lexically cleaned result is not contained
// within r.
func (r InventoryRoot) Join(rel source.ArtifactPath) (AbsoluteArtifactPath, error) {
	if r.IsZero() {
		return AbsoluteArtifactPath{}, ErrInvalidInventoryRoot
	}
	if rel.IsZero() {
		return AbsoluteArtifactPath{}, source.ErrInvalidArtifactPath
	}
	joined := filepath.Join(r.abs, filepath.FromSlash(rel.String()))
	cleaned := filepath.Clean(joined)
	if !isWithin(r.abs, cleaned) {
		return AbsoluteArtifactPath{}, ErrArtifactPathEscape
	}
	return AbsoluteArtifactPath{root: r, rel: rel, abs: cleaned}, nil
}

// InventoryRoots holds the (user, project) pair of InventoryRoot for a given
// Target. projectRoot is optional: leaving it as the zero InventoryRoot means
// the project scope is not configured.
type InventoryRoots struct {
	user    InventoryRoot
	project InventoryRoot
}

// NewInventoryRoots returns an InventoryRoots with userRoot required and
// projectRoot optional. Pass the zero InventoryRoot for projectRoot when no
// project scope is configured.
func NewInventoryRoots(userRoot, projectRoot InventoryRoot) (InventoryRoots, error) {
	if userRoot.IsZero() {
		return InventoryRoots{}, ErrInvalidInventoryRoot
	}
	return InventoryRoots{user: userRoot, project: projectRoot}, nil
}

// For returns the InventoryRoot for the given Scope. Returns ErrInvalidScope
// when scope is not one of the valid constants, and ErrProjectRootNotConfigured
// when scope is ScopeProject but no project root was configured.
func (rs InventoryRoots) For(scope Scope) (InventoryRoot, error) {
	if err := scope.Validate(); err != nil {
		return InventoryRoot{}, err
	}
	switch scope {
	case ScopeUser:
		return rs.user, nil
	case ScopeProject:
		if rs.project.IsZero() {
			return InventoryRoot{}, ErrProjectRootNotConfigured
		}
		return rs.project, nil
	default:
		return InventoryRoot{}, ErrInvalidScope
	}
}

// AbsoluteArtifactPath is an Artifact's absolute path on the host filesystem,
// produced by joining an InventoryRoot with an ArtifactPath. It retains both
// the root and the relative components so callers can reason about boundary
// crossings.
type AbsoluteArtifactPath struct {
	root InventoryRoot
	rel  source.ArtifactPath
	abs  string
}

// String returns the absolute path. Returns "" for the zero value.
func (p AbsoluteArtifactPath) String() string { return p.abs }

// IsZero reports whether p is the zero value.
func (p AbsoluteArtifactPath) IsZero() bool { return p.abs == "" }

// Root returns the InventoryRoot the absolute path was derived from.
func (p AbsoluteArtifactPath) Root() InventoryRoot { return p.root }

// RelativePath returns the ArtifactPath component the absolute path was
// derived from.
func (p AbsoluteArtifactPath) RelativePath() source.ArtifactPath { return p.rel }

// isWithin reports whether path lies within root after lexical cleaning. Both
// root and path must already be lexically cleaned and absolute.
func isWithin(root, path string) bool {
	if path == root {
		return true
	}
	sep := string(filepath.Separator)
	prefix := root
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(path, prefix)
}
