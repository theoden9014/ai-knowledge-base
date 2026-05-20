package cli

import (
	"context"
	"errors"
	"io/fs"
	"testing/fstest"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// stubFetcher implements remote.Fetcher for tests. It claims a single
// host via supportsHost and returns either a pre-built stubFetchedPack
// or an injected error. closeErr lets tests force ErrCleanupFailed.
type stubFetcher struct {
	supportsHost string
	fetchErr     error
	closeErr     error
	files        fstest.MapFS
	packDir      string
}

func (s *stubFetcher) Supports(host string) bool { return host == s.supportsHost }

func (s *stubFetcher) Fetch(_ context.Context, _ *remote.Locator) (remote.FetchedPack, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return &stubFetchedPack{files: s.files, packDir: s.packDir, closeErr: s.closeErr}, nil
}

// stubFetchedPack is the in-memory FetchedPack used by stubFetcher. It
// can simulate cleanup failures via closeErr.
type stubFetchedPack struct {
	files    fstest.MapFS
	packDir  string
	closeErr error
	closed   bool
}

func (s *stubFetchedPack) FS() fs.FS       { return s.files }
func (s *stubFetchedPack) PackDir() string { return s.packDir }
func (s *stubFetchedPack) Close() error {
	if s.closed {
		return nil // idempotent
	}
	s.closed = true
	return s.closeErr
}

// errStubFetch is a stable error consumers can errors.Is against without
// the test depending on the exact remote.Err* value.
var errStubFetch = errors.New("stub fetch failure")
