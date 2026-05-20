package remote

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

// fakeFetchedPack is a minimal FetchedPack used to verify Dispatch returns
// the FetchedPack produced by the matching fetcher.
type fakeFetchedPack struct {
	fs      fs.FS
	packDir string
}

func (f *fakeFetchedPack) FS() fs.FS       { return f.fs }
func (f *fakeFetchedPack) PackDir() string { return f.packDir }
func (f *fakeFetchedPack) Close() error    { return nil }

// fakeFetcher is a Fetcher whose Supports check matches a fixed host and
// whose Fetch returns the pre-configured FetchedPack or error.
type fakeFetcher struct {
	host string
	pack FetchedPack
	err  error
}

func (f *fakeFetcher) Supports(host string) bool {
	return host == f.host
}

func (f *fakeFetcher) Fetch(ctx context.Context, loc *Locator) (FetchedPack, error) {
	return f.pack, f.err
}

func TestDispatch(t *testing.T) {
	type args struct {
		ctx      context.Context
		loc      *Locator
		fetchers []Fetcher
	}
	tests := []struct {
		name        string
		args        args
		wantSameAs  FetchedPack
		wantErr     bool
		wantErrKind error
	}{
		{
			name: "single matching fetcher returns its pack",
			args: args{
				ctx: context.Background(),
				loc: &Locator{Host: "github.com", Owner: "o", Repo: "r"},
				fetchers: []Fetcher{
					&fakeFetcher{host: "github.com", pack: &fakeFetchedPack{fs: fstest.MapFS{}, packDir: "."}},
				},
			},
		},
		{
			name: "first matching fetcher wins",
			args: args{
				ctx: context.Background(),
				loc: &Locator{Host: "github.com", Owner: "o", Repo: "r"},
				fetchers: []Fetcher{
					&fakeFetcher{host: "github.com", pack: &fakeFetchedPack{fs: fstest.MapFS{"a": &fstest.MapFile{}}, packDir: "."}},
					&fakeFetcher{host: "github.com", pack: &fakeFetchedPack{fs: fstest.MapFS{"b": &fstest.MapFile{}}, packDir: "."}},
				},
			},
		},
		{
			name: "non-matching fetcher is skipped",
			args: args{
				ctx: context.Background(),
				loc: &Locator{Host: "gitlab.com", Owner: "o", Repo: "r"},
				fetchers: []Fetcher{
					&fakeFetcher{host: "github.com", pack: &fakeFetchedPack{packDir: "skip"}},
					&fakeFetcher{host: "gitlab.com", pack: &fakeFetchedPack{packDir: "."}},
				},
			},
		},
		{
			name: "no matching fetcher returns ErrUnsupportedHost",
			args: args{
				ctx: context.Background(),
				loc: &Locator{Host: "bitbucket.org", Owner: "o", Repo: "r"},
				fetchers: []Fetcher{
					&fakeFetcher{host: "github.com"},
				},
			},
			wantErr:     true,
			wantErrKind: ErrUnsupportedHost,
		},
		{
			name: "empty fetchers returns ErrUnsupportedHost",
			args: args{
				ctx:      context.Background(),
				loc:      &Locator{Host: "github.com", Owner: "o", Repo: "r"},
				fetchers: nil,
			},
			wantErr:     true,
			wantErrKind: ErrUnsupportedHost,
		},
		{
			name: "fetch failure propagates without ErrUnsupportedHost",
			args: args{
				ctx: context.Background(),
				loc: &Locator{Host: "github.com", Owner: "o", Repo: "r"},
				fetchers: []Fetcher{
					&fakeFetcher{host: "github.com", err: ErrCloneFailed},
				},
			},
			wantErr:     true,
			wantErrKind: ErrCloneFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-compute the expected FetchedPack pointer when success
			// is expected, by finding the first matching fetcher in
			// tt.args.fetchers. This keeps the want value pointer-equal
			// to the underlying pack stored on the fakeFetcher, which
			// is exactly what Dispatch is contracted to return.
			var wantPack FetchedPack
			if !tt.wantErr {
				for _, f := range tt.args.fetchers {
					if ff, ok := f.(*fakeFetcher); ok && ff.Supports(tt.args.loc.Host) {
						wantPack = ff.pack
						break
					}
				}
				if wantPack == nil {
					t.Fatal("test setup: no matching fetcher found for success case")
				}
			}

			got, err := Dispatch(tt.args.ctx, tt.args.loc, tt.args.fetchers)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Dispatch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, tt.wantErrKind) {
					t.Errorf("Dispatch() error = %v, want errors.Is %v", err, tt.wantErrKind)
				}
				return
			}
			if got != wantPack {
				t.Errorf("Dispatch() = %v, want pointer-equal to %v", got, wantPack)
			}
		})
	}
}
