package remote

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
)

const githubHost = "github.com"

func newGitHubFetcher() *GitHubFetcher {
	return &GitHubFetcher{
		Git:            NewDefaultGitClient(),
		TempDirFactory: defaultTempDir,
	}
}

func defaultTempDir() (string, error) {
	return os.MkdirTemp("", "knit-remote-*")
}

func (f *GitHubFetcher) supports(host string) bool {
	return host == githubHost
}

func (f *GitHubFetcher) fetch(ctx context.Context, loc *Locator) (FetchedPack, error) {
	factory := f.TempDirFactory
	if factory == nil {
		factory = defaultTempDir
	}
	dir, err := factory()
	if err != nil {
		return nil, fmt.Errorf("remote: create temp dir: %w", err)
	}

	if err := f.Git.CloneShallow(ctx, loc.CloneURL(), dir); err != nil {
		_ = os.RemoveAll(dir)
		// Preserve the underlying error when it already wraps the
		// sentinel (the default GitClient wraps ErrCloneFailed itself).
		// For other implementations the error is wrapped here so callers
		// can always branch with errors.Is(err, ErrCloneFailed).
		if !errors.Is(err, ErrCloneFailed) {
			return nil, fmt.Errorf("%w: %v", ErrCloneFailed, err)
		}
		return nil, err
	}

	pack := &gitFetchedPack{
		root:    dir,
		packDir: ".",
	}
	if loc.Subpath != "" {
		pack.packDir = loc.Subpath
	}
	return pack, nil
}

// gitFetchedPack is the FetchedPack implementation returned by
// GitHubFetcher. It owns the cloned temporary directory and releases it on
// Close.
type gitFetchedPack struct {
	root    string
	packDir string

	mu     sync.Mutex
	closed bool
}

func (p *gitFetchedPack) FS() fs.FS {
	return os.DirFS(p.root)
}

func (p *gitFetchedPack) PackDir() string {
	return p.packDir
}

func (p *gitFetchedPack) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if err := os.RemoveAll(p.root); err != nil {
		return fmt.Errorf("%w: %v", ErrCleanupFailed, err)
	}
	return nil
}
