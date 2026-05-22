package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// requirePackArg is a helper that extracts a single <pack> positional
// argument from fs. If the argument is missing it returns
// ErrMissingArgument; if extra arguments are present it returns ErrUsage.
func requirePackArg(fs interface {
	NArg() int
	Arg(int) string
}) (string, error) {
	if fs.NArg() < 1 {
		return "", fmt.Errorf("%w: missing <pack>", ErrMissingArgument)
	}
	if fs.NArg() > 1 {
		return "", fmt.Errorf("%w: too many positional arguments (want 1, got %d)", ErrUsage, fs.NArg())
	}
	return fs.Arg(0), nil
}

// buildFactory combines scope=user/project resolution and
// DistributionFactory construction from the scope / target flag values.
//
// userBase / projectRoot are resolved lazily only for the scope that is
// actually requested:
//
//   - scope == user    -> resolve only userBase. Pass an empty
//     projectRoot and delegate to the distribution layer's
//     ErrProjectRootNotConfigured path.
//   - scope == project -> resolve only projectRoot. Pass an empty
//     userBase. This separation preserves the UX of "use only
//     scope=project in a CI environment that has no HOME".
//
// Even when knowledgeFlag is empty, this function does not auto-detect a
// knowledge dir. Callers resolve it separately through
// [resolveKnowledgeDir] only when needed, because list / uninstall do
// not require knowledge.
func buildFactory(rt *Runtime, scopeFlag, targetFlag string) (inventory.Scope, []source.Target, *DistributionFactory, error) {
	scope, err := validateScope(scopeFlag)
	if err != nil {
		return "", nil, nil, err
	}
	sr := newScopeResolver(rt)
	var userBase, projectRoot string
	switch scope {
	case inventory.ScopeUser:
		userBase, err = sr.userBase()
		if err != nil {
			return "", nil, nil, err
		}
	case inventory.ScopeProject:
		projectRoot, err = sr.projectRoot()
		if err != nil {
			return "", nil, nil, err
		}
	}
	codexHome := rt.Getenv("CODEX_HOME")
	factory := NewDistributionFactory(userBase, projectRoot, codexHome)
	targets, err := factory.ResolveTargets(targetFlag)
	if err != nil {
		return "", nil, nil, err
	}
	return scope, targets, factory, nil
}

// resolveKnowledgeDir auto-detects the absolute path to the knowledge/
// directory using knowledgeResolver. With --knowledge-dir gone, only the
// upward-search path remains.
func resolveKnowledgeDir(rt *Runtime) (string, error) {
	return newKnowledgeResolver(rt).resolve("")
}

// loadPack loads a single pack from knowledgeDir/<packName>/ using the
// canonical Loader + Validator stack. This is the pack-name entry point;
// local-path and remote-URL arguments must be routed through
// [loadPackFromArg] instead.
func loadPack(ctx context.Context, knowledgeDir, packName string) (*source.Pack, error) {
	return loadPackFromFS(ctx, os.DirFS(knowledgeDir), packName)
}

// loadPackFromLocalDir loads a single pack whose directory was given as a
// literal filesystem path (absolute or relative). The path is normalized
// to an absolute path via [filepath.Abs] (resolved against rt.Getwd for
// relative paths) and then split into (parent, base) so the Loader can be
// invoked with os.DirFS(parent) and packDir=base.
//
// Errors:
//   - If the path does not exist, returns [source.ErrPackDirNotFound]
//     wrapped with the original input.
//   - All other errors propagate from [loadPackFromFS] (most notably
//     [source.ErrManifestNotFound] when the directory exists but has no
//     manifest.yaml).
func loadPackFromLocalDir(ctx context.Context, rt *Runtime, arg string) (*source.Pack, error) {
	abs, err := canonicalLocalDirPath(rt, arg)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", source.ErrPackDirNotFound, arg)
		}
		return nil, fmt.Errorf("cli: stat pack dir %q: %w", arg, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s (not a directory)", source.ErrPackDirNotFound, arg)
	}
	parent := filepath.Dir(abs)
	base := filepath.Base(abs)
	return loadPackFromFS(ctx, os.DirFS(parent), base)
}

func canonicalLocalDirPath(rt *Runtime, arg string) (string, error) {
	abs := arg
	if !filepath.IsAbs(abs) {
		wd, err := rt.Getwd()
		if err != nil {
			return "", fmt.Errorf("cli: resolve working directory for %q: %w", arg, err)
		}
		abs = filepath.Join(wd, abs)
	}
	return filepath.Clean(abs), nil
}

// loadPackFromFS is the shared inner helper that both [loadPack] and the
// remote-fetch path use to actually invoke the Loader. Splitting it out
// keeps the validator construction in a single place.
func loadPackFromFS(ctx context.Context, fsys fs.FS, packDir string) (*source.Pack, error) {
	v, err := source.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("cli: build validator: %w", err)
	}
	loader := source.NewLoader(v)
	pack, _, err := loader.LoadPack(ctx, fsys, packDir)
	if err != nil {
		return nil, err
	}
	return pack, nil
}

// cleanupResolvedPack runs rp.Cleanup() and, if it returns an error,
// writes a one-line warning to rt.Stderr. The error is intentionally not
// propagated back to the caller because cleanup failure (typically a
// remote temp-dir removal error) is a leak warning, not a command-level
// failure: install / update / build either succeeded or returned their
// own error before Cleanup ran, and overwriting that signal with a
// post-cleanup error would mislead the user about what happened.
//
// Use as the body of `defer cleanupResolvedPack(rt, rp)` immediately
// after a successful loadPackFromArg.
func cleanupResolvedPack(rt *Runtime, rp *resolvedPack) {
	if err := rp.Cleanup(); err != nil {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: cleanup failed: %v\n", err)
	}
}

// stripURLScheme accepts "http://" or "https://" prefixes commonly pasted
// by users (e.g. "https://github.com/owner/repo") and returns the
// remainder. Any other input is returned unchanged. The behavior is
// host-only triage: this helper does not validate that what remains is a
// well-formed locator (that is [remote.Parse]'s job).
//
// Git SSH form ("git@github.com:owner/repo") is intentionally NOT handled
// here. Supporting SSH-style references is future work and would require
// extending [remote.Parse] with a separate shape.
func stripURLScheme(arg string) string {
	switch {
	case strings.HasPrefix(arg, "https://"):
		return arg[len("https://"):]
	case strings.HasPrefix(arg, "http://"):
		return arg[len("http://"):]
	default:
		return arg
	}
}

// resolvedPack bundles a loaded [source.Pack] with a cleanup function and
// the canonical pack name as it should appear in user-visible messages.
//
//   - Pack: the loaded pack. Always non-nil on success.
//   - Name: the pack name suitable for logging and for the SourceEntryID
//     <pack>.<kind>.<entry> convention. Always equals Pack.Name, which
//     [source.Loader] derives from the manifest's `pack:` field (which
//     by convention also equals the pack directory name). Using
//     Pack.Name uniformly — for both local and remote args — guarantees
//     update / uninstall's installationBelongsToPack lookup matches the
//     same key the Installer recorded at install time.
//   - Cleanup: idempotent cleanup callback. For local args this is a
//     no-op. For remote args this calls [remote.FetchedPack.Close] which
//     removes the temp directory. Callers MUST defer this immediately
//     after a successful loadPackFromArg.
//
// Cleanup error policy: the returned function may surface
// [remote.ErrCleanupFailed] (typically temp-dir removal failed). The
// caller's defer is expected to write that error to rt.Stderr as a
// warning but NOT to overwrite the Run-level error; cleanup failure is
// a leak warning, not a command failure. This keeps the Run error
// signal consistent with the user's intent (the install/build itself
// either succeeded or did not). See cmd_install.go / cmd_update.go /
// cmd_build.go for the canonical defer pattern.
type resolvedPack struct {
	Pack    *source.Pack
	Name    string
	Source  source.SourceRef
	Cleanup func() error
}

// argResolver loads a single resolvedPack for one ArgKind. The shared
// loadPackFromArg dispatcher chooses the right resolver from packResolvers.
type argResolver func(ctx context.Context, rt *Runtime, t TriagedArg) (*resolvedPack, error)

// packResolvers maps ArgKind to its resolver. Adding a new ArgKind is
// one entry plus a resolver function below.
var packResolvers = map[ArgKind]argResolver{
	ArgKindRemoteURL: resolveRemoteURLArg,
	ArgKindLocalPath: resolveLocalPathArg,
	ArgKindPackName:  resolvePackNameArg,
}

// loadPackFromArg is the unified entry point used by install / update /
// build for the <pack-or-path-or-url> positional argument. Triage is
// delegated to [TriageArg]; this function applies an ambiguity guard and
// then forwards to the [argResolver] registered for the resulting kind.
//
// Error handling per Kind:
//   - ArgKindRemoteURL : remote.Parse / remote.Dispatch errors surface
//     verbatim; error_map.go maps each remote.Err* to an ExitCode.
//   - ArgKindLocalPath : missing directory -> [source.ErrPackDirNotFound];
//     missing manifest -> the loader's [source.ErrManifestNotFound].
//   - ArgKindPackName  : when knowledge/ cannot be auto-detected,
//     returns [ErrKnowledgeDirNotFound].
//
// The cleanup function in the returned resolvedPack is always non-nil
// (local paths return a no-op closure) so callers can defer it
// unconditionally.
func loadPackFromArg(ctx context.Context, rt *Runtime, arg string) (*resolvedPack, error) {
	t := TriageArg(arg)

	// Ambiguity guard: a single token containing "." but no "/" is
	// most likely a half-typed host name. Reject with an actionable
	// hint instead of letting the pack-name loader fail with
	// ErrManifestNotFound.
	if t.Kind == ArgKindPackName && strings.ContainsRune(t.Cleaned, '.') {
		return nil, fmt.Errorf(
			"%w: ambiguous argument %q: use ./%s for a local path or https://%s/<owner>/<repo> for a remote URL",
			ErrUsage, arg, arg, arg,
		)
	}

	resolver, ok := packResolvers[t.Kind]
	if !ok {
		// Unreachable while TriageArg only returns the registered kinds;
		// kept so that adding a new ArgKind without updating
		// packResolvers fails loudly at runtime rather than silently.
		return nil, fmt.Errorf("%w: unknown ArgKind=%d", ErrUsage, t.Kind)
	}
	return resolver(ctx, rt, t)
}

// resolveRemoteURLArg fetches a remote pack and wraps the result in a
// resolvedPack whose Cleanup closes the fetched FS.
func resolveRemoteURLArg(ctx context.Context, rt *Runtime, t TriagedArg) (*resolvedPack, error) {
	loc, err := remote.Parse(t.Cleaned)
	if err != nil {
		return nil, err
	}
	fetched, err := remote.Dispatch(ctx, loc, rt.Fetchers)
	if err != nil {
		return nil, err
	}
	pack, err := loadPackFromFS(ctx, fetched.FS(), fetched.PackDir())
	if err != nil {
		_ = fetched.Close()
		return nil, err
	}
	return &resolvedPack{
		Pack:    pack,
		Name:    pack.Name,
		Source:  source.SourceRef{Kind: source.SourceRefRemoteURL, Value: t.Cleaned},
		Cleanup: fetched.Close,
	}, nil
}

// resolveLocalPathArg loads a pack from a literal filesystem path.
func resolveLocalPathArg(ctx context.Context, rt *Runtime, t TriagedArg) (*resolvedPack, error) {
	pack, err := loadPackFromLocalDir(ctx, rt, t.Cleaned)
	if err != nil {
		return nil, err
	}
	abs, err := canonicalLocalDirPath(rt, t.Cleaned)
	if err != nil {
		return nil, err
	}
	return &resolvedPack{
		Pack:    pack,
		Name:    pack.Name,
		Source:  source.SourceRef{Kind: source.SourceRefLocalPath, Value: abs},
		Cleanup: func() error { return nil },
	}, nil
}

// resolvePackNameArg loads a pack from the auto-detected knowledge/
// directory.
func resolvePackNameArg(ctx context.Context, rt *Runtime, t TriagedArg) (*resolvedPack, error) {
	knowledgeDir, err := resolveKnowledgeDir(rt)
	if err != nil {
		return nil, err
	}
	pack, err := loadPack(ctx, knowledgeDir, t.Cleaned)
	if err != nil {
		return nil, err
	}
	return &resolvedPack{
		Pack:    pack,
		Name:    pack.Name,
		Source:  source.SourceRef{Kind: source.SourceRefLocalPath, Value: filepath.Join(knowledgeDir, t.Cleaned)},
		Cleanup: func() error { return nil },
	}, nil
}

// writeArtifactToDir writes art under outDir using art.Path as a
// relative suffix. The write goes through inventory.FsArtifactStore so
// build -o follows the same filesystem-I/O abstraction as install. The
// outDir argument is normalized to an absolute path so the underlying
// InventoryRoot can accept it.
func writeArtifactToDir(outDir string, art source.Artifact) error {
	rel, err := source.NewArtifactPath(art.Path)
	if err != nil {
		return fmt.Errorf("cli: validate artifact path: %w", err)
	}
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("cli: resolve out dir %q: %w", outDir, err)
	}
	root, err := inventory.NewInventoryRoot(abs)
	if err != nil {
		return fmt.Errorf("cli: invalid out dir: %w", err)
	}
	dst, err := root.Join(rel)
	if err != nil {
		return fmt.Errorf("cli: join artifact path: %w", err)
	}
	if err := inventory.NewFsArtifactStore().Write(context.Background(), dst, art.Content, art.Mode); err != nil {
		return fmt.Errorf("cli: write artifact: %w", err)
	}
	return nil
}

// printArtifactSummary writes a one-line summary per artifact (used by
// `knit build` without -o).
func printArtifactSummary(w io.Writer, artifacts []source.Artifact) {
	for _, art := range artifacts {
		_, _ = fmt.Fprintf(w, "%-12s %s (%d bytes, entries=%s)\n",
			art.Target, art.Path, len(art.Content), strings.Join(art.SourceEntryIDs, ","))
	}
}

// aggregateOrSingle picks the right return shape for sub-commands that
// iterate multiple targets. If the user requested a single target, the
// failure (if any) is returned verbatim so callers see the precise
// sentinel. If multiple targets were requested and any failed, the
// failures are wrapped in *AggregateError. nil is returned when none
// failed.
func aggregateOrSingle(targetCount int, failures []TargetFailure) error {
	if len(failures) == 0 {
		return nil
	}
	if targetCount == 1 {
		return failures[0].Err
	}
	return &AggregateError{Failures: failures}
}
