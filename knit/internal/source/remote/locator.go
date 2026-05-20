package remote

// Locator identifies a knowledge pack hosted on a remote git provider. It is
// a pure data type: parsing produces a Locator, fetching consumes one. No
// I/O happens during Locator construction or inspection.
//
// The initial version targets the repository HEAD only. Ref/branch/commit
// pinning is intentionally not supported. The forward-compatible extension
// path is an "@<ref>" suffix on the input string (for example
// "github.com/<owner>/<repo>/<subpath>@v1.2.0") that would be parsed into a
// future Ref string field on Locator. GitHub-style "tree/<branch>" segments
// and URL "?ref=<branch>" query strings are explicitly out of scope so the
// input form stays canonical.
type Locator struct {
	// Host is the lower-cased hostname of the git provider. Parse
	// guarantees this is already lower-case, so Fetcher implementations
	// can compare against literal lower-case hosts (e.g. "github.com")
	// directly. Only "github.com" is supported initially; new providers
	// are added by registering a Fetcher whose Supports method returns
	// true for their host.
	Host string

	// Owner is the path segment immediately after the host (for github
	// this is the account or organization name).
	Owner string

	// Repo is the repository name (the segment after Owner, with any
	// trailing ".git" stripped).
	Repo string

	// Subpath is the in-repository path that points at the pack
	// directory, relative to the repository root.
	//
	// Normalization guaranteed by Parse:
	//   - Uses forward slashes only. Backslashes in the input cause
	//     ErrInvalidLocator (no implicit conversion).
	//   - Never starts or ends with "/".
	//   - Internal "//" runs are rejected (ErrInvalidLocator).
	//   - May be empty when the pack lives at the repository root.
	Subpath string
}

// CloneURL returns the URL that should be passed to "git clone" for this
// Locator's Host/Owner/Repo. The Subpath is intentionally not embedded in
// the URL because clone operates on whole repositories.
func (l *Locator) CloneURL() string {
	return l.cloneURL()
}

// Parse decodes a user-supplied pack argument of the form
// "<host>/<owner>/<repo>[/<subpath>...]" into a Locator. The argument must
// already have been classified as a remote reference (see IsRemoteArg);
// passing a local pack name yields ErrInvalidLocator.
//
// Accepted shapes (canonical):
//
//	github.com/<owner>/<repo>
//	github.com/<owner>/<repo>/<subpath>
//
// Edge-case contract:
//
//   - URL scheme prefix: "http://" and "https://" prefixes are rejected
//     with ErrInvalidLocator. The CLI layer must strip them before calling
//     Parse (we want a single canonical input form rather than parsing
//     URL syntax here).
//
//   - Query strings and fragments: any '?' or '#' in the input is rejected
//     with ErrInvalidLocator. Ref/branch selection is intentionally out of
//     scope (see Locator's doc comment).
//
//   - Repeated slashes: consecutive '/' characters (e.g. "github.com//x")
//     are rejected with ErrInvalidLocator. We do not silently coalesce
//     them.
//
//   - Backslashes: any '\' character is rejected with ErrInvalidLocator;
//     callers must use forward slashes (this also avoids ambiguity on
//     Windows).
//
//   - URL-encoded characters: percent-encoded sequences (%2F etc.) are
//     not decoded. They are treated as literal segment characters, which
//     for typical owner/repo names will fail downstream during the clone
//     itself. Future versions may add decoding; this is a deliberate
//     "no surprise normalization" choice in the initial release.
//
//   - Host case: Host is lower-cased in the resulting Locator. Owner,
//     Repo, and Subpath are preserved verbatim because git hosts vary
//     in their case sensitivity rules.
//
//   - Trailing ".git": stripped from the repo segment only.
//
//   - Trailing slash on the argument as a whole: rejected (would yield
//     an empty trailing Subpath segment).
func Parse(arg string) (*Locator, error) {
	return parse(arg)
}

// IsRemoteArg reports whether the user-supplied pack argument should be
// interpreted as a remote reference rather than a local pack directory
// name or a local filesystem path.
//
// Triage rule (host-like first segment):
//   - The argument must not begin with "/" (absolute path), "./", or "../"
//     (relative path). Those shapes are reserved for local filesystem
//     paths by the CLI layer.
//   - The first segment (the substring up to the first "/" or the entire
//     string if there is no "/") must match a host-like pattern: at least
//     one inner "." with non-empty labels on both sides, i.e.
//     "<word>.<word>(.<word>)*". Pure-dot fragments like "." and ".." are
//     therefore not host-like and yield false.
//   - The argument must contain at least one "/" so the downstream
//     "<host>/<owner>/<repo>" shape can be reached. A bare "github.com"
//     yields false here (it is neither a valid local pack name nor a
//     complete remote ref); Parse still rejects it explicitly.
//
// IsRemoteArg only triages between code paths; full structural validation
// happens in Parse.
func IsRemoteArg(arg string) bool {
	return isRemoteArg(arg)
}
