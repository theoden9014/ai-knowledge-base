package cli

import (
	"bytes"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

func TestNewRuntime(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	args := []string{"install", "pack"}
	getenv := func(string) string { return "v" }
	getwd := func() (string, error) { return "/cwd", nil }
	fsys := fstest.MapFS{}

	rt := NewRuntime(stdout, stderr, args, getenv, getwd, fsys)
	if rt == nil {
		t.Fatalf("NewRuntime returned nil")
	}
	if rt.Stdout != stdout {
		t.Errorf("Stdout not wired")
	}
	if rt.Stderr != stderr {
		t.Errorf("Stderr not wired")
	}
	if got, want := len(rt.Args), len(args); got != want {
		t.Errorf("Args length = %d, want %d", got, want)
	}
	if rt.Getenv == nil || rt.Getenv("X") != "v" {
		t.Errorf("Getenv not wired")
	}
	wd, _ := rt.Getwd()
	if wd != "/cwd" {
		t.Errorf("Getwd not wired: got %q", wd)
	}
	if rt.Fsys == nil {
		t.Errorf("Fsys not wired")
	}
}

func TestDefaultRuntime(t *testing.T) {
	rt := DefaultRuntime()
	if rt == nil {
		t.Fatalf("DefaultRuntime returned nil")
	}
	if rt.Stdout == nil {
		t.Errorf("Stdout is nil")
	}
	if rt.Stderr == nil {
		t.Errorf("Stderr is nil")
	}
	if rt.Getenv == nil {
		t.Errorf("Getenv is nil")
	}
	if rt.Getwd == nil {
		t.Errorf("Getwd is nil")
	}
	if rt.Fsys == nil {
		t.Errorf("Fsys is nil")
	}
}

func TestRuntime_WithFetchers(t *testing.T) {
	gh := &stubFetcher{supportsHost: "github.com"}
	gl := &stubFetcher{supportsHost: "gitlab.com"}

	type fields struct {
		Stdout   io.Writer
		Stderr   io.Writer
		Args     []string
		Getenv   func(string) string
		Getwd    func() (string, error)
		Fsys     fs.FS
		Fetchers []remote.Fetcher
	}
	type args struct {
		fetchers []remote.Fetcher
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Runtime
	}{
		{
			name:   "nil arg clears existing fetchers",
			fields: fields{Fetchers: []remote.Fetcher{gh}},
			args:   args{fetchers: nil},
			want:   &Runtime{Fetchers: nil},
		},
		{
			name:   "empty variadic clears existing fetchers",
			fields: fields{Fetchers: []remote.Fetcher{gh}},
			args:   args{fetchers: []remote.Fetcher{}},
			want:   &Runtime{Fetchers: nil},
		},
		{
			name:   "single fetcher injected",
			fields: fields{Fetchers: nil},
			args:   args{fetchers: []remote.Fetcher{gh}},
			want:   &Runtime{Fetchers: []remote.Fetcher{gh}},
		},
		{
			name:   "multiple fetchers preserve injection order",
			fields: fields{Fetchers: nil},
			args:   args{fetchers: []remote.Fetcher{gh, gl}},
			want:   &Runtime{Fetchers: []remote.Fetcher{gh, gl}},
		},
		{
			name:   "existing fetchers replaced (not appended)",
			fields: fields{Fetchers: []remote.Fetcher{gl}},
			args:   args{fetchers: []remote.Fetcher{gh}},
			want:   &Runtime{Fetchers: []remote.Fetcher{gh}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &Runtime{
				Stdout:   tt.fields.Stdout,
				Stderr:   tt.fields.Stderr,
				Args:     tt.fields.Args,
				Getenv:   tt.fields.Getenv,
				Getwd:    tt.fields.Getwd,
				Fsys:     tt.fields.Fsys,
				Fetchers: tt.fields.Fetchers,
			}
			got := rt.WithFetchers(tt.args.fetchers...)
			if got != rt {
				t.Errorf("WithFetchers should return the same receiver pointer; got %p, want %p", got, rt)
			}
			if len(got.Fetchers) != len(tt.want.Fetchers) {
				t.Fatalf("len(Fetchers) = %d, want %d", len(got.Fetchers), len(tt.want.Fetchers))
			}
			for i := range tt.want.Fetchers {
				if got.Fetchers[i] != tt.want.Fetchers[i] {
					t.Errorf("Fetchers[%d] = %p, want %p", i, got.Fetchers[i], tt.want.Fetchers[i])
				}
			}
		})
	}
}

// WithFetchers must defensively copy the input slice so the caller can
// mutate or reuse it without affecting the Runtime's state.
func TestRuntime_WithFetchers_defensiveCopy(t *testing.T) {
	gh := &stubFetcher{supportsHost: "github.com"}
	gl := &stubFetcher{supportsHost: "gitlab.com"}
	src := []remote.Fetcher{gh}
	rt := &Runtime{}
	rt.WithFetchers(src...)
	src[0] = gl // mutate caller's slice
	if got := rt.Fetchers[0]; got != gh {
		t.Errorf("Fetchers[0] = %v after caller mutation, want %v (no defensive copy)", got, gh)
	}
}
