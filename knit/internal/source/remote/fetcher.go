package remote

import (
	"context"
	"fmt"
	"io/fs"
)

// Fetcher retrieves a knowledge pack referenced by a Locator and returns a
// FetchedPack that exposes the pack's containing directory as an fs.FS.
//
// Fetcher is an interface so that additional git providers (gitlab.com,
// bitbucket.org, etc.) can be introduced as new implementations without
// touching the call site. Selection across multiple Fetchers is handled by
// Dispatch in this package; callers should not write their own selection
// loop.
type Fetcher interface {
	// Supports reports whether this fetcher can handle Locators whose
	// Host equals the argument. The argument is expected to already be
	// lower-case (Parse guarantees this on Locator.Host); implementations
	// should compare against literal lower-case host names.
	Supports(host string) bool

	// Fetch downloads the repository referenced by loc into a temporary
	// directory and returns a FetchedPack rooted at that directory. The
	// returned FetchedPack owns the temp directory and must be released
	// with FetchedPack.Close.
	//
	// ctx is honored for cancellation: a long-running clone must abort
	// promptly when ctx is done, and any partially-created temp
	// directory must be removed before Fetch returns.
	Fetch(ctx context.Context, loc *Locator) (FetchedPack, error)
}

// Dispatch selects the first Fetcher in fetchers whose Supports returns
// true for loc.Host and delegates to its Fetch. If no fetcher claims
// support, ErrUnsupportedHost is returned (no fetcher is invoked).
//
// Dispatch exists so that the responsibility for "no fetcher matched" lives
// in this package rather than being re-implemented by every caller. The CLI
// layer is therefore free of host-selection logic; it just constructs the
// list of available Fetchers and calls Dispatch.
//
// The fetchers slice is consulted in order; a registered earlier wins over
// a later one for the same host.
func Dispatch(ctx context.Context, loc *Locator, fetchers []Fetcher) (FetchedPack, error) {
	for _, f := range fetchers {
		if f.Supports(loc.Host) {
			return f.Fetch(ctx, loc)
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedHost, loc.Host)
}

// FetchedPack is the handle returned by Fetcher.Fetch. It exposes the
// minimal surface that source.Loader.LoadPack needs (an fs.FS plus the pack
// directory name within that FS) and an explicit Close so the caller can
// release the underlying temporary directory.
//
// The interface is deliberately narrow:
//   - It does not expose the absolute host path of the temp directory;
//     callers should never need it.
//   - It does not return *source.Pack itself; the bridging to Loader is
//     left to the CLI layer to keep this package free of a circular
//     dependency on source.
type FetchedPack interface {
	// FS returns the filesystem rooted at the cloned repository root.
	// The fs.FS is valid until Close is called. Calling FS after Close
	// still returns a non-nil fs.FS, but any Open / Stat / ReadFile on
	// it will fail with an ENOENT-equivalent error because the backing
	// directory has been removed; callers must not retain the fs.FS
	// past Close.
	FS() fs.FS

	// PackDir returns the path of the pack directory within FS(), in
	// the form expected by source.Loader.LoadPack (forward slashes, no
	// leading or trailing slash). It corresponds to Locator.Subpath
	// when set, otherwise to the repository root ".".
	PackDir() string

	// Close releases the temporary directory backing FS(). It is safe
	// to call from multiple goroutines but the semantics are sequential:
	//
	//   - The first call performs the cleanup and either returns nil
	//     (success) or an error wrapping ErrCleanupFailed (failure).
	//   - All subsequent calls return nil regardless of the first
	//     call's outcome. The failure is reported exactly once; this
	//     package never retries automatically because the failure
	//     cause (locked file, permission denied) is typically not
	//     transient. Callers who need to retry cleanup must track the
	//     directory path themselves and re-run RemoveAll externally.
	Close() error
}
