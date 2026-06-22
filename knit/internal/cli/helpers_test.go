package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// buildFactory must NOT touch HOME when scope=project. This is the
// reviewer-requested UX fix: CI environments often run without HOME but
// with a writable project tree.
func Test_buildFactory_scopeProject_doesNotRequireHome(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Mark projectDir as a project root via `.knit`.
	if err := os.WriteFile(filepath.Join(projectDir, ".knit"), nil, 0o644); err != nil {
		t.Fatalf("write .knit: %v", err)
	}
	rt := &Runtime{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Getenv: func(string) string { return "" }, // HOME unset
		Getwd:  func() (string, error) { return projectDir, nil },
		Fsys:   os.DirFS("/"),
	}
	scope, targets, factory, err := buildFactory(rt, "project", "claude")
	if err != nil {
		t.Fatalf("buildFactory err: %v", err)
	}
	if scope != inventory.ScopeProject {
		t.Errorf("scope = %v, want project", scope)
	}
	if len(targets) != 1 {
		t.Errorf("len(targets) = %d, want 1", len(targets))
	}
	if factory.userBase != "" {
		t.Errorf("userBase = %q, want empty when scope=project", factory.userBase)
	}
	if factory.projectRoot != projectDir {
		t.Errorf("projectRoot = %q, want %q", factory.projectRoot, projectDir)
	}
}

// buildFactory must continue to require HOME when scope=user.
func Test_buildFactory_scopeUser_requiresHome(t *testing.T) {
	rt := &Runtime{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Getenv: func(string) string { return "" }, // HOME unset
		Getwd:  func() (string, error) { return "/", nil },
		Fsys:   fstest.MapFS{},
	}
	_, _, _, err := buildFactory(rt, "user", "claude")
	if !errors.Is(err, ErrHomeNotSet) {
		t.Errorf("err = %v, want ErrHomeNotSet", err)
	}
}

// And buildFactory must not silently consult projectRoot when scope=user,
// even if the project root is discoverable; that would leak project state
// into a user-scope install path.
func Test_buildFactory_scopeUser_doesNotResolveProjectRoot(t *testing.T) {
	tmp := t.TempDir()
	homeDir := filepath.Join(tmp, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// no .knit / .git anywhere; scopeResolver.projectRoot would fail. If
	// buildFactory wrongly tried to resolve it under scope=user, we'd see
	// ErrProjectRootNotFound here.
	rt := &Runtime{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Getenv: func(k string) string {
			if k == "HOME" {
				return homeDir
			}
			return ""
		},
		Getwd: func() (string, error) { return tmp, nil },
		Fsys:  os.DirFS("/"),
	}
	scope, _, factory, err := buildFactory(rt, "user", "claude")
	if err != nil {
		t.Fatalf("buildFactory err: %v", err)
	}
	if scope != inventory.ScopeUser {
		t.Errorf("scope = %v, want user", scope)
	}
	if factory.userBase != homeDir {
		t.Errorf("userBase = %q, want %q", factory.userBase, homeDir)
	}
	if factory.projectRoot != "" {
		t.Errorf("projectRoot = %q, want empty when scope=user", factory.projectRoot)
	}
}

// remotePackFS builds a fstest.MapFS suitable for a stubFetchedPack
// returning a minimal but loader-valid knowledge pack named "remote-p".
// PackDir() in the stub should point at the pack subdirectory ("remote-p").
func remotePackFS(packDir, packName string) fstest.MapFS {
	return remotePackFSWithBody(packDir, packName, "body of remote skill a\n")
}

func remotePackFSWithBody(packDir, packName, body string) fstest.MapFS {
	manifest := "pack: " + packName + "\n" +
		"version: 0.1.0\n" +
		"description: remote test pack\n" +
		"default_tools: [claude]\n" +
		"entries:\n" +
		"  - id: " + packName + ".skill.a\n" +
		"    path: skills/a\n"
	skill := "---\n" +
		"id: " + packName + ".skill.a\n" +
		"kind: skill\n" +
		"name: " + packName + "-a\n" +
		"description: skill a\n" +
		"---\n" +
		body
	return fstest.MapFS{
		packDir + "/manifest.yaml":     {Data: []byte(manifest)},
		packDir + "/skills/a/SKILL.md": {Data: []byte(skill)},
	}
}

func Test_loadPackFromArg(t *testing.T) {
	t.Run("ambiguous arg with dot but no slash returns ErrUsage", func(t *testing.T) {
		f := newCmdFixture(t)
		rt, _, _ := f.runtime(t)
		_, err := loadPackFromArg(context.Background(), rt, "foo.bar")
		if !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})

	t.Run("local path pointing at a file returns ErrPackDirNotFound", func(t *testing.T) {
		f := newCmdFixture(t)
		// Create a regular file (not a directory) inside f.tmp.
		filePath := filepath.Join(f.tmp, "not-a-dir.txt")
		if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		rt, _, _ := f.runtime(t)
		_, err := loadPackFromArg(context.Background(), rt, "./not-a-dir.txt")
		if !errors.Is(err, source.ErrPackDirNotFound) {
			t.Errorf("err = %v, want source.ErrPackDirNotFound", err)
		}
	})

	t.Run("pack name arg auto-detects knowledge/", func(t *testing.T) {
		f := newCmdFixture(t)
		rt, _, _ := f.runtime(t)
		// f.tmp contains knowledge/, runtime.Getwd returns f.tmp, so
		// auto-detection should find it.
		rp, err := loadPackFromArg(context.Background(), rt, f.pack)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rp.Pack == nil {
			t.Fatalf("Pack is nil")
		}
		if rp.Name != "p" {
			t.Errorf("Name = %q, want %q", rp.Name, "p")
		}
		if err := rp.Cleanup(); err != nil {
			t.Errorf("local Cleanup should be a no-op, got %v", err)
		}
	})

	t.Run("remote URL arg dispatches to fetcher and loads pack", func(t *testing.T) {
		// stub fetcher returns an in-memory FS whose pack directory is
		// "remote-p", mirroring how a clone of github.com/owner/remote-p
		// would look.
		stub := &stubFetcher{
			supportsHost: "github.com",
			files:        remotePackFS("remote-p", "remote-p"),
			packDir:      "remote-p",
		}
		rt := &Runtime{
			Stdout:   io.Discard,
			Stderr:   io.Discard,
			Getenv:   func(string) string { return "" },
			Getwd:    func() (string, error) { return "/", nil },
			Fsys:     fstest.MapFS{},
			Fetchers: []remote.Fetcher{stub},
		}
		rp, err := loadPackFromArg(context.Background(), rt, "github.com/owner/remote-p")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rp.Pack == nil {
			t.Fatal("Pack is nil")
		}
		if rp.Name != "remote-p" {
			t.Errorf("Name = %q, want %q (should equal Pack.Name)", rp.Name, "remote-p")
		}
		// Cleanup must delegate to FetchedPack.Close (not a no-op).
		if err := rp.Cleanup(); err != nil {
			t.Errorf("Cleanup err: %v", err)
		}
	})

	t.Run("https:// scheme is stripped before triage", func(t *testing.T) {
		stub := &stubFetcher{
			supportsHost: "github.com",
			files:        remotePackFS("remote-p", "remote-p"),
			packDir:      "remote-p",
		}
		rt := &Runtime{
			Stdout: io.Discard, Stderr: io.Discard,
			Getenv:   func(string) string { return "" },
			Getwd:    func() (string, error) { return "/", nil },
			Fsys:     fstest.MapFS{},
			Fetchers: []remote.Fetcher{stub},
		}
		rp, err := loadPackFromArg(context.Background(), rt, "https://github.com/owner/remote-p")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if rp.Name != "remote-p" {
			t.Errorf("Name = %q, want %q", rp.Name, "remote-p")
		}
	})

	t.Run("URL with no supporting fetcher returns ErrUnsupportedHost", func(t *testing.T) {
		// stubFetcher claims gitlab.com; arg points at github.com.
		stub := &stubFetcher{supportsHost: "gitlab.com"}
		rt := &Runtime{
			Stdout: io.Discard, Stderr: io.Discard,
			Getenv:   func(string) string { return "" },
			Getwd:    func() (string, error) { return "/", nil },
			Fsys:     fstest.MapFS{},
			Fetchers: []remote.Fetcher{stub},
		}
		_, err := loadPackFromArg(context.Background(), rt, "github.com/owner/repo")
		if !errors.Is(err, remote.ErrUnsupportedHost) {
			t.Errorf("err = %v, want remote.ErrUnsupportedHost", err)
		}
	})

	t.Run("fetcher returning error surfaces it without leaking a tempdir", func(t *testing.T) {
		stub := &stubFetcher{
			supportsHost: "github.com",
			fetchErr:     errStubFetch,
		}
		rt := &Runtime{
			Stdout: io.Discard, Stderr: io.Discard,
			Getenv:   func(string) string { return "" },
			Getwd:    func() (string, error) { return "/", nil },
			Fsys:     fstest.MapFS{},
			Fetchers: []remote.Fetcher{stub},
		}
		_, err := loadPackFromArg(context.Background(), rt, "github.com/owner/repo")
		if !errors.Is(err, errStubFetch) {
			t.Errorf("err = %v, want errStubFetch", err)
		}
	})

	t.Run("loader failure after fetch triggers best-effort Close", func(t *testing.T) {
		// Empty pack FS: loader will fail with ErrManifestNotFound.
		// We want to confirm the stub's FetchedPack.Close() was invoked
		// (its `closed` flag flips). We do that by holding a reference
		// to the inner FetchedPack via a custom Fetcher.
		var fetched *stubFetchedPack
		fetcher := &fetcherHook{
			supportsHost: "github.com",
			onFetch: func() remote.FetchedPack {
				fetched = &stubFetchedPack{
					files:   fstest.MapFS{}, // no manifest
					packDir: ".",
				}
				return fetched
			},
		}
		rt := &Runtime{
			Stdout: io.Discard, Stderr: io.Discard,
			Getenv:   func(string) string { return "" },
			Getwd:    func() (string, error) { return "/", nil },
			Fsys:     fstest.MapFS{},
			Fetchers: []remote.Fetcher{fetcher},
		}
		_, err := loadPackFromArg(context.Background(), rt, "github.com/owner/repo")
		if err == nil {
			t.Fatal("expected error from loader on empty pack")
		}
		if fetched == nil {
			t.Fatal("fetcher hook never invoked")
		}
		if !fetched.closed {
			t.Errorf("FetchedPack.Close should have been called after loader failure")
		}
	})
}

// fetcherHook is a small Fetcher whose Fetch() returns whatever onFetch
// constructs. It is only used by Test_loadPackFromArg to observe
// FetchedPack.Close invocation on the loader-failure path.
type fetcherHook struct {
	supportsHost string
	onFetch      func() remote.FetchedPack
}

func (f *fetcherHook) Supports(host string) bool { return host == f.supportsHost }
func (f *fetcherHook) Fetch(_ context.Context, _ *remote.Locator) (remote.FetchedPack, error) {
	return f.onFetch(), nil
}

func Test_cleanupResolvedPack(t *testing.T) {
	tests := []struct {
		name              string
		cleanupErr        error
		wantStderrContent string
	}{
		{
			name:              "success leaves stderr empty",
			cleanupErr:        nil,
			wantStderrContent: "",
		},
		{
			name:              "failure surfaces warning to stderr",
			cleanupErr:        errStubFetch,
			wantStderrContent: "warning: cleanup failed: stub fetch failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			rt := &Runtime{Stdout: io.Discard, Stderr: stderr}
			err := tt.cleanupErr // capture for closure
			rp := &resolvedPack{
				Cleanup: func() error { return err },
			}
			cleanupResolvedPack(rt, rp)
			got := stderr.String()
			if tt.wantStderrContent == "" {
				if got != "" {
					t.Errorf("stderr should be empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantStderrContent) {
				t.Errorf("stderr missing %q\ngot: %q", tt.wantStderrContent, got)
			}
		})
	}
}

func Test_stripURLScheme(t *testing.T) {
	type args struct {
		arg string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "https:// stripped",
			args: args{arg: "https://github.com/owner/repo"},
			want: "github.com/owner/repo",
		},
		{
			name: "http:// stripped",
			args: args{arg: "http://github.com/owner/repo"},
			want: "github.com/owner/repo",
		},
		{
			name: "bare host passthrough",
			args: args{arg: "github.com/owner/repo"},
			want: "github.com/owner/repo",
		},
		{
			name: "local pack name passthrough",
			args: args{arg: "structure-behavior-design"},
			want: "structure-behavior-design",
		},
		{
			name: "https with subpath",
			args: args{arg: "https://github.com/owner/repo/sub/path"},
			want: "github.com/owner/repo/sub/path",
		},
		{
			name: "empty string passthrough",
			args: args{arg: ""},
			want: "",
		},
		{
			name: "non-strippable scheme passthrough (git@ ssh form)",
			args: args{arg: "git@github.com:owner/repo"},
			want: "git@github.com:owner/repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripURLScheme(tt.args.arg); got != tt.want {
				t.Errorf("stripURLScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}
