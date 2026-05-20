package cli

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

// scopeResolver is responsible for resolving the (userRoot,
// projectRoot) pair passed to each target distribution's NewInstaller /
// NewUninstaller / NewLister from the --scope=user|project value and
// the runtime environment.
//
// SRP: keep "resolving the scope value" separate from "target-specific
// subdirectory naming" (.claude / .codex / .gemini). This type returns
// only "scope=user -> the user's base dir" and
// "scope=project -> the project's base dir". Building each Target's
// Inventory root underneath that (for example, <user base>/.claude) is
// the responsibility of DistributionFactory.
//
// Input sources:
//   - userBaseDir: the starting point for scope=user, usually $HOME.
//   - Retrieve the $HOME environment variable through Runtime.Getenv,
//     and return ErrHomeNotSet if it is unset.
//   - Codex prefers $CODEX_HOME (when set) as the ScopeUser Inventory
//     root, but this type handles only HOME. Interpretation of
//     target-specific environment variables such as CODEX_HOME is
//     delegated to DistributionFactory.
//   - projectRoot: the starting point for scope=project.
//   - Normalize the current absolute path from Runtime.Getwd into a
//     path on Runtime.Fsys, then search upward via [findUpwards] for
//     markers (`.knit` > `.git` > `go.mod`). The priority is fixed in
//     that order, with `.knit` first so that an explicit knit-aware
//     marker wins over generic ones.
//   - Return ErrProjectRootNotFound if the search result is empty.
//   - No cache is kept inside scopeResolver because a fresh resolver
//     is built per command execution.
//
// This type is an internal implementation detail and is not exported.
// Command implementations obtain Installer / Uninstaller / Lister
// through [DistributionFactory], which uses this resolver internally.
type scopeResolver struct {
	rt *Runtime
}

// newScopeResolver constructs a scopeResolver from a Runtime.
func newScopeResolver(rt *Runtime) *scopeResolver {
	return &scopeResolver{rt: rt}
}

// userBase returns the base directory for scope=user (usually $HOME).
// It returns ErrHomeNotSet when $HOME is unset.
func (r *scopeResolver) userBase() (string, error) {
	home := r.rt.Getenv("HOME")
	if home == "" {
		return "", ErrHomeNotSet
	}
	return home, nil
}

// projectRoot locates and returns the base directory for scope=project.
// It searches Runtime.Fsys for the nearest parent directory containing
// any marker from [projectRootMarkers] and returns its absolute path.
// If nothing is found, it returns ErrProjectRootNotFound.
func (r *scopeResolver) projectRoot() (string, error) {
	wd, err := r.rt.Getwd()
	if err != nil {
		return "", fmt.Errorf("cli: getwd: %w", err)
	}
	startFsPath := absToFsPath(wd)
	found, err := r.findUpwards(startFsPath, projectRootMarkers())
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", ErrProjectRootNotFound
	}
	return fsPathToAbs(found), nil
}

// projectRootMarkers returns the priority-ordered marker list used by
// projectRoot discovery. Priority is determined by slice order, with the
// first element having the highest priority:
//
//  1. ".knit"   - the strongest explicit knit-specific marker
//  2. ".git"    - a generic marker present in most repositories
//  3. "go.mod"  - the Go module root
//
// Note: package.json is intentionally excluded because it can
// accidentally match inside `node_modules/` in Node-based repositories.
// If support for that root becomes necessary, add an opt-in mechanism
// such as a future `--root-marker` flag.
//
// Expose this as a function rather than a var so tests can replace it.
// (The implementation is expected to remain a simple fixed-value
// function.)
func projectRootMarkers() []string {
	return []string{".knit", ".git", "go.mod"}
}

// validateScope normalizes a scope string into inventory.Scope.
// It returns ErrInvalidFlagValue for anything other than "user" /
// "project".
//
// This overlaps with inventory.Scope.Validate, but the CLI layer keeps
// it as a separate function so it can map to
// ErrInvalidFlagValue (= ExitUsage).
func validateScope(s string) (inventory.Scope, error) {
	switch s {
	case string(inventory.ScopeUser):
		return inventory.ScopeUser, nil
	case string(inventory.ScopeProject):
		return inventory.ScopeProject, nil
	default:
		return "", fmt.Errorf("%w: scope=%q (want user|project)", ErrInvalidFlagValue, s)
	}
}

// findUpwards starts from startDir and walks toward parent directories,
// searching rt.Fsys for a directory that contains any of the markers.
// It returns the matching path within rt.Fsys. If nothing is found, it
// returns an empty string and nil, treating the outcome as "not found"
// rather than an error.
//
// Path unification: callers (scopeResolver / knowledgeResolver) are
// expected to pass Runtime.Fsys as the filesystem under search, and this
// function accepts only the fs.FS abstraction. That lets it treat the
// real filesystem from os.DirFS("/") and fstest.MapFS equivalently.
//
// startDir must be a path within rt.Fsys (no leading slash, valid for
// fs.ValidPath). Conversion from an absolute path to an fs.FS path is
// the caller's responsibility.
//
// Marker-check order: markers are evaluated in slice order at each
// directory, and the first existing one determines the match. This
// function preserves the caller-supplied ordering so the caller controls
// priority.
func (r *scopeResolver) findUpwards(startDir string, markers []string) (string, error) {
	dir := startDir
	if dir == "" {
		dir = "."
	}
	for {
		for _, m := range markers {
			candidate := joinFsPath(dir, m)
			if _, err := fs.Stat(r.rt.Fsys, candidate); err == nil {
				return dir, nil
			}
		}
		if dir == "." {
			return "", nil
		}
		parent := path.Dir(dir)
		// path.Dir("foo") returns ".", path.Dir(".") returns ".".
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// joinFsPath joins dir and name into a fs.FS-friendly path. The root
// directory ("." in fs.FS) is treated as the empty prefix so that
// joinFsPath(".", ".knit") yields ".knit" (a valid fs.FS path) rather
// than "./.knit" (which fs.Stat rejects).
func joinFsPath(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + "/" + name
}

// absToFsPath converts an absolute filesystem path ("/foo/bar") into an
// fs.FS-style path ("foo/bar"). The filesystem root ("/") maps to ".".
// Relative inputs are returned unchanged so that tests can pass fs paths
// directly.
func absToFsPath(p string) string {
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		// already a relative fs path or empty
		if p == "" {
			return "."
		}
		return p
	}
	trimmed := strings.TrimPrefix(p, "/")
	if trimmed == "" {
		return "."
	}
	return trimmed
}

// fsPathToAbs is the inverse of absToFsPath. The fs root (".") maps to
// "/", and any other fs path receives a leading "/".
func fsPathToAbs(p string) string {
	if p == "" || p == "." {
		return "/"
	}
	return "/" + p
}
