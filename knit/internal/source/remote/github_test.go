package remote

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// fakeGitClient is a GitClient that records its CloneShallow inputs and
// returns the configured error. When cloneErr is nil it materializes a
// minimal "manifest.yaml" file in the target directory so tests can prove
// the FetchedPack's FS surfaces the cloned content.
type fakeGitClient struct {
	cloneErr   error
	wantURL    string
	gotURL     string
	gotDir     string
	gotCtxDone bool
	seed       map[string]string
}

func (g *fakeGitClient) CloneShallow(ctx context.Context, url, dir string) error {
	g.gotURL = url
	g.gotDir = dir
	if ctx.Err() != nil {
		g.gotCtxDone = true
		return ctx.Err()
	}
	if g.cloneErr != nil {
		return g.cloneErr
	}
	for relPath, content := range g.seed {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func TestNewGitHubFetcher(t *testing.T) {
	f := NewGitHubFetcher()
	if f == nil {
		t.Fatal("NewGitHubFetcher() returned nil")
	}
	if f.Git == nil {
		t.Error("NewGitHubFetcher().Git is nil, want default GitClient")
	}
	if f.TempDirFactory == nil {
		t.Error("NewGitHubFetcher().TempDirFactory is nil, want default factory")
	}
}

func TestGitHubFetcher_Supports(t *testing.T) {
	type fields struct {
		Git            GitClient
		TempDirFactory func() (string, error)
	}
	type args struct {
		host string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "github.com matches", args: args{host: "github.com"}, want: true},
		{name: "uppercase does not match (Parse normalizes input)", args: args{host: "GitHub.com"}, want: false},
		{name: "gitlab does not match", args: args{host: "gitlab.com"}, want: false},
		{name: "empty host does not match", args: args{host: ""}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &GitHubFetcher{
				Git:            tt.fields.Git,
				TempDirFactory: tt.fields.TempDirFactory,
			}
			if got := f.Supports(tt.args.host); got != tt.want {
				t.Errorf("GitHubFetcher.Supports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitHubFetcher_Fetch_success(t *testing.T) {
	tempDir := t.TempDir()
	git := &fakeGitClient{
		wantURL: "https://github.com/o/r.git",
		seed: map[string]string{
			"manifest.yaml":           "pack: r\n",
			"knowledge/pack/file.txt": "hello",
		},
	}
	f := &GitHubFetcher{
		Git:            git,
		TempDirFactory: func() (string, error) { return tempDir, nil },
	}
	loc := &Locator{Host: "github.com", Owner: "o", Repo: "r", Subpath: "knowledge/pack"}

	fp, err := f.Fetch(context.Background(), loc)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer func() {
		if cerr := fp.Close(); cerr != nil {
			t.Errorf("Close() error = %v", cerr)
		}
	}()

	if git.gotURL != "https://github.com/o/r.git" {
		t.Errorf("git client received url %q, want https clone url", git.gotURL)
	}
	if git.gotDir != tempDir {
		t.Errorf("git client received dir %q, want %q", git.gotDir, tempDir)
	}
	if got := fp.PackDir(); got != "knowledge/pack" {
		t.Errorf("PackDir() = %q, want %q", got, "knowledge/pack")
	}

	data, err := fs.ReadFile(fp.FS(), "knowledge/pack/file.txt")
	if err != nil {
		t.Fatalf("read from FS: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("FS file content = %q, want %q", data, "hello")
	}
}

func TestGitHubFetcher_Fetch_emptySubpathReturnsDot(t *testing.T) {
	tempDir := t.TempDir()
	git := &fakeGitClient{seed: map[string]string{"manifest.yaml": "pack: r\n"}}
	f := &GitHubFetcher{
		Git:            git,
		TempDirFactory: func() (string, error) { return tempDir, nil },
	}
	loc := &Locator{Host: "github.com", Owner: "o", Repo: "r"}

	fp, err := f.Fetch(context.Background(), loc)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	t.Cleanup(func() {
		if err := fp.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if got := fp.PackDir(); got != "." {
		t.Errorf("PackDir() = %q, want %q for empty Subpath", got, ".")
	}
}

func TestGitHubFetcher_Fetch_cloneFailureCleansUp(t *testing.T) {
	tempDir := t.TempDir()
	cloneDir := filepath.Join(tempDir, "clone")
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	git := &fakeGitClient{cloneErr: errors.New("boom")}
	f := &GitHubFetcher{
		Git:            git,
		TempDirFactory: func() (string, error) { return cloneDir, nil },
	}
	loc := &Locator{Host: "github.com", Owner: "o", Repo: "r"}

	fp, err := f.Fetch(context.Background(), loc)
	if err == nil {
		if cerr := fp.Close(); cerr != nil {
			t.Errorf("Close() error = %v", cerr)
		}
		t.Fatal("Fetch() expected error from clone failure")
	}
	if !errors.Is(err, ErrCloneFailed) {
		t.Errorf("Fetch() error = %v, want errors.Is ErrCloneFailed", err)
	}
	if _, statErr := os.Stat(cloneDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("clone dir %q still exists after Fetch failure", cloneDir)
	}
}

func TestGitHubFetcher_Fetch_tempDirFailureSurfaces(t *testing.T) {
	git := &fakeGitClient{}
	wantErr := errors.New("mkdir-temp failed")
	f := &GitHubFetcher{
		Git:            git,
		TempDirFactory: func() (string, error) { return "", wantErr },
	}
	loc := &Locator{Host: "github.com", Owner: "o", Repo: "r"}

	_, err := f.Fetch(context.Background(), loc)
	if err == nil {
		t.Fatal("Fetch() expected error from TempDirFactory failure")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Fetch() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestFetchedPack_Close_idempotent(t *testing.T) {
	tempDir := t.TempDir()
	cloneDir := filepath.Join(tempDir, "clone")
	git := &fakeGitClient{seed: map[string]string{"manifest.yaml": "pack: r\n"}}
	f := &GitHubFetcher{
		Git:            git,
		TempDirFactory: func() (string, error) { return cloneDir, os.MkdirAll(cloneDir, 0o755) },
	}
	loc := &Locator{Host: "github.com", Owner: "o", Repo: "r"}
	fp, err := f.Fetch(context.Background(), loc)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if err := fp.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if _, statErr := os.Stat(cloneDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("clone dir %q still exists after Close", cloneDir)
	}

	// Subsequent Close calls return nil without retrying.
	if err := fp.Close(); err != nil {
		t.Errorf("second Close() error = %v, want nil after successful close", err)
	}
	if err := fp.Close(); err != nil {
		t.Errorf("third Close() error = %v, want nil after successful close", err)
	}
}

func TestFetchedPack_Close_failureReportedOnlyOnce(t *testing.T) {
	// Simulate a cleanup failure by pointing the FetchedPack at a path
	// inside a directory whose write bit has been removed. Under POSIX
	// this prevents RemoveAll from unlinking entries, surfacing as
	// ErrCleanupFailed on the first Close.
	parent := t.TempDir()
	target := filepath.Join(parent, "locked")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "file"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(parent, 0o755)
	})

	fp := &gitFetchedPack{root: target, packDir: "."}

	first := fp.Close()
	if first == nil {
		t.Skip("RemoveAll did not fail on this platform; cannot exercise failure path")
	}
	if !errors.Is(first, ErrCleanupFailed) {
		t.Errorf("first Close() error = %v, want errors.Is ErrCleanupFailed", first)
	}
	if second := fp.Close(); second != nil {
		t.Errorf("second Close() error = %v, want nil (failure reported only once)", second)
	}
}
