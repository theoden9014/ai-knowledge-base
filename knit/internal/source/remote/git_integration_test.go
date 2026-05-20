//go:build integration

package remote

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests for defaultGitClient.CloneShallow. These tests invoke
// the real "git" binary and are gated behind the "integration" build tag
// because they (a) require git to be installed and (b) for the success
// case need to clone from another local git repository on disk.
//
// Run with:
//
//	go test -tags=integration ./internal/source/remote/...

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

// initLocalRepo creates a minimal bare-equivalent git repo on disk that
// CloneShallow can clone from via a local file path. It seeds a single
// commit so that --depth=1 has something to fetch.
func initLocalRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "init")
	return dir
}

func TestDefaultGitClient_CloneShallow_success(t *testing.T) {
	requireGit(t)
	src := initLocalRepo(t)
	dst := filepath.Join(t.TempDir(), "clone")

	c := NewDefaultGitClient()
	if err := c.CloneShallow(context.Background(), src, dst); err != nil {
		t.Fatalf("CloneShallow() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "README.md")); err != nil {
		t.Errorf("cloned file missing: %v", err)
	}
}

func TestDefaultGitClient_CloneShallow_failureWrapsErrCloneFailed(t *testing.T) {
	requireGit(t)
	c := NewDefaultGitClient()
	dst := filepath.Join(t.TempDir(), "clone")
	// Use a syntactically valid but unresolvable local path so git
	// returns a non-zero exit code without depending on the network.
	bogus := filepath.Join(t.TempDir(), "does-not-exist")
	err := c.CloneShallow(context.Background(), bogus, dst)
	if err == nil {
		t.Fatal("CloneShallow() expected error from bogus source")
	}
	if !errors.Is(err, ErrCloneFailed) {
		t.Errorf("CloneShallow() error = %v, want errors.Is ErrCloneFailed", err)
	}
}

func TestDefaultGitClient_CloneShallow_ctxCancelPreferred(t *testing.T) {
	requireGit(t)
	c := NewDefaultGitClient()
	dst := filepath.Join(t.TempDir(), "clone")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Give git some time slot in case it polls; the test only asserts
	// that the returned error reflects ctx, not ErrCloneFailed.
	time.Sleep(1 * time.Millisecond)
	bogus := filepath.Join(t.TempDir(), "does-not-exist")
	err := c.CloneShallow(ctx, bogus, dst)
	if err == nil {
		t.Fatal("CloneShallow() expected error with cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("CloneShallow() error = %v, want errors.Is context.Canceled", err)
	}
}
