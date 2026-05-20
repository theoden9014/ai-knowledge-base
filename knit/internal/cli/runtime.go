package cli

import (
	"io"
	"io/fs"
	"os"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// Runtime is the shared I/O and environment abstraction used by all
// subcommands.
//
// Roles:
//   - Make stdout / stderr destinations swappable for testability.
//   - Abstract access to environment variables such as HOME and
//     CODEX_HOME (testability plus a single gateway for scope=user
//     resolution).
//   - Abstract current-directory lookup (the starting point for project
//     root discovery).
//   - Abstract filesystem access so upward search for the project root /
//     knowledge dir can be replaced with fstest.MapFS.
//
// From an SRP perspective, Runtime is a value object that provides only
// references to I/O and environment. Logic that changes subcommand
// behavior belongs on the Command side.
//
// # Args handling (single source of truth for the args path)
//
// Runtime.Args is the single source of truth for subcommand-routing
// input args. Neither Execute nor App.Execute accepts args as a
// parameter; the contract is that this field always contains the latest
// args. Tests can swap args by passing them to NewRuntime, while
// production uses DefaultRuntime() to populate os.Args[1:].
//
// Do not use the zero value of this struct: it panics if Stdout /
// Stderr / Fsys / Getenv / Getwd are nil. Construct it via [NewRuntime]
// or [DefaultRuntime].
type Runtime struct {
	// Stdout is the normal output destination (success messages, list
	// results, etc.).
	Stdout io.Writer

	// Stderr is the destination for errors, progress, and warnings.
	Stderr io.Writer

	// Args is the argument slice after the subcommand, equivalent to
	// os.Args[1:]. This field is the single source of truth for the args
	// path (Execute / App.Execute do not accept args as a parameter).
	Args []string

	// Getenv retrieves environment variables. It is normally os.Getenv but
	// can be replaced in tests.
	Getenv func(string) string

	// Getwd retrieves the current directory. It is normally os.Getwd but
	// can be replaced in tests.
	Getwd func() (string, error)

	// Fsys is the filesystem abstraction used by upward search
	// ([findUpwards]) for the project root / knowledge dir. In production
	// it holds an abstraction over the real OS filesystem (equivalent to
	// os.DirFS("/")); in tests it can be replaced with fstest.MapFS.
	//
	// The search logic is designed around absolute-style paths inside fsys
	// (without a leading slash). Responsibility for normalizing the
	// Runtime.Getwd result into an fs.FS path belongs to scopeResolver /
	// knowledgeResolver.
	Fsys fs.FS

	// Fetchers is the set of [remote.Fetcher]s used when the pack argument
	// is a remote URL. Selection is by host (for example, "github.com"),
	// with one Fetcher chosen when Supports(host) == true.
	//
	// Design decision:
	//   - Treat Fetchers as an extension of the I/O abstractions
	//     (Stdout/Stderr/Fsys), specifically network I/O, and keep them on
	//     Runtime. This lets subcommands avoid knowing Fetchers directly
	//     and centralizes test-time stub replacement in one place.
	//   - They do not belong on DistributionFactory because the factory's
	//     job is mapping target -> concrete Builder/Installer, while
	//     remote fetching is not target-specific.
	//
	// If this field is nil or empty, commands invoked with a remote URL
	// argument fall through to [remote.ErrUnsupportedHost]. DefaultRuntime
	// registers exactly one production GitHubFetcher.
	Fetchers []remote.Fetcher
}

// NewRuntime constructs a Runtime from the given I/O and environment
// accessors.
//
// Contract:
//   - stdout and stderr must both be non-nil.
//   - args may be nil or an empty slice (treated as no subcommand).
//   - getenv, getwd, and fsys must all be non-nil.
//
// Fetchers remain unset in this constructor. In production,
// [DefaultRuntime] registers one GitHubFetcher. Tests or special use
// cases can replace Fetchers either by assigning to the returned
// Runtime's Fetchers field directly or by using the [WithFetchers]
// helper.
func NewRuntime(stdout, stderr io.Writer, args []string, getenv func(string) string, getwd func() (string, error), fsys fs.FS) *Runtime {
	return &Runtime{
		Stdout: stdout,
		Stderr: stderr,
		Args:   args,
		Getenv: getenv,
		Getwd:  getwd,
		Fsys:   fsys,
	}
}

// WithFetchers replaces rt.Fetchers and returns the same *Runtime. It is
// provided for method chaining (`NewRuntime(...).WithFetchers(...)`).
// If fetchers is nil, rt.Fetchers is cleared to nil.
func (rt *Runtime) WithFetchers(fetchers ...remote.Fetcher) *Runtime {
	if len(fetchers) == 0 {
		rt.Fetchers = nil
		return rt
	}
	rt.Fetchers = append([]remote.Fetcher(nil), fetchers...)
	return rt
}

// DefaultRuntime constructs a Runtime using the real OS environment
// (os.Stdout / os.Stderr / os.Args[1:] / os.Getenv / os.Getwd /
// os.DirFS("/")). It registers the production GitHubFetcher through
// [defaultFetchers]. main is expected to call this and pass the result
// to Execute.
func DefaultRuntime() *Runtime {
	var args []string
	if len(os.Args) > 1 {
		args = append([]string(nil), os.Args[1:]...)
	}
	rt := NewRuntime(os.Stdout, os.Stderr, args, os.Getenv, os.Getwd, os.DirFS("/"))
	if fetchers := defaultFetchers(); len(fetchers) > 0 {
		rt.WithFetchers(fetchers...)
	}
	return rt
}

// defaultFetchers returns the production set of [remote.Fetcher]s wired
// into DefaultRuntime. It is split out so the construction can be stubbed
// in tests, and so adding a new git provider (gitlab, bitbucket, ...) is
// a one-line append here without touching DefaultRuntime or any
// sub-command.
//
// Order matters: [remote.Dispatch] returns the first Fetcher whose
// Supports matches the locator's host, so earlier entries win.
func defaultFetchers() []remote.Fetcher {
	return []remote.Fetcher{remote.NewGitHubFetcher()}
}
