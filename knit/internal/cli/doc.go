// Package cli provides the CLI entry point for knit.
//
// This package defines and assembles the subcommand tree and implements
// each command by combining `source`, `inventory`, and
// `distribution/<target>`. Selection of concrete implementations, such
// as which Target to build/distribute to and which Scope to use as the
// Inventory, happens in this package.
//
// # Subcommand Set
//
// The vocabulary follows package-management tools:
//
//   - knit install <pack-or-path-or-url> [--scope=user|project] [--target=claude|codex|gemini|all]
//   - knit uninstall <pack>              [--scope=user|project] [--target=claude|codex|gemini|all]
//   - knit list                          [--scope=user|project] [--target=claude|codex|gemini|all]
//   - knit update <pack-or-path-or-url>  [--scope=user|project] [--target=claude|codex|gemini|all]
//   - knit build <pack-or-path-or-url>   --target=claude|codex|gemini [-o <dir>]
//   - knit help [<subcommand>]
//   - knit --version
//
// Note: `build` is a debug-oriented command and accepts only a single
// Target. `--target=all` is rejected. Users who want to build multiple
// Targets must invoke the command separately for each Target.
//
// # Pack Argument Interpretation (Wave6)
//
// The <pack-or-path-or-url> argument for install / build, and explicit-source
// update, is triaged into the following three forms (see [loadPackFromArg] for
// details):
//
//  1. A remote git URL (first segment is host-like, e.g. "github.com")
//     Example: `github.com/owner/repo`, `github.com/owner/repo/path/to/pack`
//     Example: `https://github.com/owner/repo`, `http://github.com/owner/repo`
//     -> the cli layer strips any http(s):// prefix, triages via
//     [remote.IsRemoteArg], then runs [remote.Parse] ->
//     [remote.Fetcher.Fetch], cloning into a temporary directory before
//     handing off to the existing Loader. FetchedPack.Close() removes the
//     temporary directory on exit.
//
//  2. A local directory path (absolute, "./" / "../" prefix, or contains "/")
//     Example: `./knowledge/structure-behavior-design`
//     Example: `/Users/alice/work/my-pack`
//     -> the cli layer resolves to an absolute path, then loads with
//     os.DirFS(parent) and packDir=base. Non-existent directories
//     return [source.ErrPackDirNotFound].
//
//  3. A pack name (kebab-case, no "/" and not host-like)
//     Example: `structure-behavior-design`
//     -> loaded from the auto-detected knowledge/ directory (upward
//     search from cwd). If knowledge/ is not found,
//     [ErrKnowledgeDirNotFound] is returned.
//
// update has one additional rule: a bare pack name refreshes only
// installations that have a recorded remote source. Packs installed from local
// sources must be updated by passing the local path explicitly.
//
// uninstall accepts local pack names and local directory paths. For local
// directory paths, the manifest is read only to recover the canonical Pack.Name
// used for matching Installation.Provenance. Remote URLs return ErrUsage so a
// network fetch can never trigger deletion.
//
// # Fetcher DI
//
// [remote.Fetcher] values are stored in [Runtime.Fetchers]. In
// production, [DefaultRuntime] registers a single GitHubFetcher. If a
// remote URL is passed to a Runtime with no Fetcher, [loadPackFromArg]
// returns [remote.ErrUnsupportedHost]. This also enables a "test-only
// cli without the remote library" by leaving Fetchers nil and thereby
// disabling remote paths entirely.
//
// # Framework Selection Policy
//
// This package does not use external CLI libraries such as cobra,
// urfave/cli, or kong. Instead it is built from the standard library's
// flag package plus a custom subcommand router. The reasoning is:
//
//   - It preserves the knit project policy of minimizing external
//     dependencies: go.mod requires only YAML / JSON Schema / TOML /
//     go-cmp, with no CLI library.
//   - The number of subcommands is small (install / uninstall / list /
//     update / build / help), and the flags are limited to `--scope`,
//     `--target`, `--output`, and `--version`, so most cobra features
//     such as dynamic completion, persistent flag inheritance, and
//     persistent hooks are unnecessary.
//   - It allows explicit SRP-oriented separation between subcommand
//     routing, flag sets, and command execution.
//   - If migration to cobra or similar becomes necessary later, each
//     command already conforms to a single interface (Command), so the
//     transition is mostly a matter of rearranging implementations.
//
// # Key Types
//
//   - [Execute] is the only entry point called from main.
//   - [Command] is the interface implemented by each subcommand.
//   - [App] is the container that aggregates and routes Commands.
//   - [Runtime] bundles the shared I/O and environment for subcommands
//     (stdout, stderr, args, getenv, getwd, fsys). It is abstracted for
//     testability. The single source of truth for args is
//     [Runtime.Args], and neither Execute nor App.Execute takes args as
//     a parameter.
//   - [DistributionFactory] builds the Builder / Installer /
//     Uninstaller / Lister for the selected Target based on the
//     `--target` flag. Interpretation of Target-specific environment
//     rules, such as preferring codex's `$CODEX_HOME` as the
//     ScopeUser Inventory root, is also centralized here. This favors a
//     clear aggregation point over strict OCP. Adding a new Target with
//     similar special handling should require only one change here.
//
// # Exit Codes
//
// This package returns the exit code as the return value of [Execute].
// main is a thin wrapper that passes that value directly to os.Exit.
// Exit-code conventions are centralized as constants in [ExitCode].
//
// `-h` / `--help` (an explicit help request) follows POSIX/GNU
// convention: write to stdout and return ExitSuccess. By contrast,
// running with no arguments is treated as incorrect usage, so it writes
// to stderr and returns ExitUsage. The distinction is intentional and
// reflected in the ExitCode values.
//
// # Supported Platforms
//
// This package, and knit as a whole, primarily targets Unix-like
// systems (macOS / Linux). Windows drive-letter paths (`C:\...`) and
// backslash-separated paths are currently out of scope. In particular,
// helpers such as absToFsPath and fsPathToAbs used by scopeResolver are
// implemented with Unix absolute paths (`/` prefix) as the assumption.
// Windows support remains future work.
package cli
