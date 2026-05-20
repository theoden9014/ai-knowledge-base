// Package remote fetches a knowledge pack from a remote git host (initially
// github.com) into a temporary directory and exposes it as an fs.FS so the
// existing source.Loader can read it without knowing the pack came from the
// network.
//
// The package is split along SRP boundaries:
//
//   - Locator + Parse: pure data + parsing of the user-supplied argument
//     string (no I/O).
//   - Fetcher / FetchedPack: pluggable transport. The default implementation
//     for github.com is GitHubFetcher; future hosts plug in as additional
//     Fetcher implementations without touching callers.
//   - GitClient: the subprocess boundary. Replacing the GitClient lets tests
//     run without invoking the real git binary.
//
// Lifecycle: every FetchedPack owns a temporary directory that must be
// released with Close(). Callers are expected to defer Close() immediately
// after a successful Fetch so the temp directory is never leaked.
package remote
