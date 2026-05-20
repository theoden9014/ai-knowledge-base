package cli

import (
	"errors"
	"io"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

// newTestRuntime constructs a minimal Runtime suitable for resolver tests.
// Each call returns a fresh value so tests do not share state.
func newTestRuntime(env map[string]string, wd string, fsys fstest.MapFS) *Runtime {
	getenv := func(k string) string { return env[k] }
	getwd := func() (string, error) { return wd, nil }
	return &Runtime{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Args:   nil,
		Getenv: getenv,
		Getwd:  getwd,
		Fsys:   fsys,
	}
}

func Test_newScopeResolver(t *testing.T) {
	rt := newTestRuntime(nil, "/", fstest.MapFS{})
	got := newScopeResolver(rt)
	if got == nil {
		t.Fatalf("newScopeResolver() returned nil")
	}
	if got.rt != rt {
		t.Errorf("scopeResolver.rt mismatch: got %p, want %p", got.rt, rt)
	}
}

func Test_scopeResolver_userBase(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr error
	}{
		{
			name: "HOME set",
			env:  map[string]string{"HOME": "/Users/foo"},
			want: "/Users/foo",
		},
		{
			name:    "HOME unset → ErrHomeNotSet",
			env:     map[string]string{},
			wantErr: ErrHomeNotSet,
		},
		{
			name:    "HOME empty string treated as unset",
			env:     map[string]string{"HOME": ""},
			wantErr: ErrHomeNotSet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newScopeResolver(newTestRuntime(tt.env, "/", fstest.MapFS{}))
			got, err := r.userBase()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_scopeResolver_projectRoot(t *testing.T) {
	tests := []struct {
		name    string
		wd      string
		fsys    fstest.MapFS
		want    string
		wantErr error
	}{
		{
			name: "marker at cwd",
			wd:   "/repo/proj",
			fsys: fstest.MapFS{
				"repo/proj/.knit": {},
			},
			want: "/repo/proj",
		},
		{
			name: "marker at parent",
			wd:   "/repo/proj/subdir",
			fsys: fstest.MapFS{
				"repo/proj/.git": {},
			},
			want: "/repo/proj",
		},
		{
			name: ".knit takes precedence over .git in same dir",
			wd:   "/repo/proj",
			fsys: fstest.MapFS{
				"repo/proj/.knit": {},
				"repo/proj/.git":  {},
			},
			want: "/repo/proj",
		},
		{
			name: "nearest .knit overrides farther .git",
			wd:   "/repo/proj/inner",
			fsys: fstest.MapFS{
				"repo/.git":             {},
				"repo/proj/inner/.knit": {},
			},
			want: "/repo/proj/inner",
		},
		{
			name: "no marker found → ErrProjectRootNotFound",
			wd:   "/repo/proj",
			fsys: fstest.MapFS{
				"repo/proj/main.go": {},
			},
			wantErr: ErrProjectRootNotFound,
		},
		{
			name: "package.json must NOT be a marker (per reviewer feedback)",
			wd:   "/repo/proj",
			fsys: fstest.MapFS{
				"repo/proj/package.json": {},
			},
			wantErr: ErrProjectRootNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newScopeResolver(newTestRuntime(nil, tt.wd, tt.fsys))
			got, err := r.projectRoot()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_projectRootMarkers(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "fixed priority: .knit > .git > go.mod (no package.json)",
			want: []string{".knit", ".git", "go.mod"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectRootMarkers()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("projectRootMarkers() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_validateScope(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    inventory.Scope
		wantErr error
	}{
		{name: "user", s: "user", want: inventory.ScopeUser},
		{name: "project", s: "project", want: inventory.ScopeProject},
		{name: "empty → ErrInvalidFlagValue", s: "", wantErr: ErrInvalidFlagValue},
		{name: "unknown → ErrInvalidFlagValue", s: "global", wantErr: ErrInvalidFlagValue},
		{name: "uppercase rejected", s: "USER", wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateScope(tt.s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("scope mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_scopeResolver_findUpwards(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fstest.MapFS
		start   string
		markers []string
		want    string
	}{
		{
			name: "find at startDir",
			fsys: fstest.MapFS{
				"a/b/.knit": {},
			},
			start:   "a/b",
			markers: []string{".knit"},
			want:    "a/b",
		},
		{
			name: "ascend to ancestor",
			fsys: fstest.MapFS{
				"a/.knit":          {},
				"a/b/c/other.file": {},
			},
			start:   "a/b/c",
			markers: []string{".knit"},
			want:    "a",
		},
		{
			name: "marker priority: first match in list wins per dir",
			fsys: fstest.MapFS{
				"root/.knit": {},
				"root/.git":  {},
			},
			start:   "root",
			markers: []string{".git", ".knit"},
			want:    "root",
		},
		{
			name: "nearest dir wins regardless of marker order",
			fsys: fstest.MapFS{
				"a/.git":     {},
				"a/b/c/.git": {},
			},
			start:   "a/b/c",
			markers: []string{".git"},
			want:    "a/b/c",
		},
		{
			name: "not found returns empty string",
			fsys: fstest.MapFS{
				"a/b/file": {},
			},
			start:   "a/b",
			markers: []string{".knit"},
			want:    "",
		},
		{
			name: "root directory (.) searched as final ancestor",
			fsys: fstest.MapFS{
				".knit": {},
			},
			start:   "a/b/c",
			markers: []string{".knit"},
			want:    ".",
		},
		{
			name:    "empty start treated as root",
			fsys:    fstest.MapFS{".knit": {}},
			start:   "",
			markers: []string{".knit"},
			want:    ".",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newScopeResolver(newTestRuntime(nil, "/", tt.fsys))
			got, err := r.findUpwards(tt.start, tt.markers)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
