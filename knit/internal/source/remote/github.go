package remote

import "context"

// GitHubFetcher is the Fetcher implementation for Locators whose Host is
// "github.com". It clones the referenced repository with depth=1 into a
// fresh temporary directory and returns a FetchedPack scoped to that
// directory plus the Locator's Subpath.
//
// GitHubFetcher composes a GitClient (the subprocess boundary) and a
// tempDirFactory (the os.MkdirTemp boundary) so both can be replaced in
// tests without spinning up a real git or touching the user's temp
// directory.
type GitHubFetcher struct {
	// Git is the GitClient used to perform the clone. Must be non-nil.
	Git GitClient

	// TempDirFactory creates a fresh empty directory that the resulting
	// FetchedPack will own. When nil, the default behavior is to call
	// os.MkdirTemp("", "knit-remote-*").
	TempDirFactory func() (string, error)
}

// NewGitHubFetcher returns a GitHubFetcher backed by the default GitClient
// and the default temp-directory factory. The returned fetcher is safe for
// concurrent use.
func NewGitHubFetcher() *GitHubFetcher {
	return newGitHubFetcher()
}

// Supports reports whether host equals "github.com". Parse guarantees that
// Locator.Host is already lower-cased, so the check is a plain string
// equality with the literal "github.com"; no additional case folding is
// performed here.
func (f *GitHubFetcher) Supports(host string) bool {
	return f.supports(host)
}

// Fetch clones the repository referenced by loc into a fresh temporary
// directory and returns a FetchedPack whose FS() is rooted at the clone
// directory and whose PackDir() reflects loc.Subpath.
//
// Fetch is responsible for cleanup on failure: if the clone fails after a
// temp directory has been created, the directory is removed before Fetch
// returns so callers never see a leaked partial clone.
func (f *GitHubFetcher) Fetch(ctx context.Context, loc *Locator) (FetchedPack, error) {
	return f.fetch(ctx, loc)
}
