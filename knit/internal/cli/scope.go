package cli

import (
	"fmt"
	"io/fs"

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
	found, ok, err := r.findUpwards(FsPathFromAbs(wd), projectRootMarkers())
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrProjectRootNotFound
	}
	return found.Abs(), nil
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
// It returns the matching FsPath when one is found; otherwise the
// second return value is false (treated as "not found" rather than an
// error).
//
// Path unification: callers (scopeResolver / knowledgeResolver) pass
// Runtime.Fsys as the filesystem under search, so the real filesystem
// from os.DirFS("/") and fstest.MapFS are treated equivalently. startDir
// is the FsPath equivalent of the caller's absolute working directory;
// the FsPath wrapper handles the "/-vs-." conventions.
//
// Marker-check order: markers are evaluated in slice order at each
// directory, and the first existing one determines the match.
func (r *scopeResolver) findUpwards(startDir FsPath, markers []string) (FsPath, bool, error) {
	dir := startDir
	for {
		for _, m := range markers {
			candidate := dir.Join(m)
			if _, err := fs.Stat(r.rt.Fsys, candidate.String()); err == nil {
				return dir, true, nil
			}
		}
		parent, ok := dir.Parent()
		if !ok {
			return FsPath{}, false, nil
		}
		dir = parent
	}
}
