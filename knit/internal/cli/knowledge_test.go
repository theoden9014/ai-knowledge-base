package cli

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func Test_newKnowledgeResolver(t *testing.T) {
	rt := newTestRuntime(nil, "/", fstest.MapFS{})
	got := newKnowledgeResolver(rt)
	if got == nil {
		t.Fatalf("newKnowledgeResolver returned nil")
	}
	if got.rt != rt {
		t.Errorf("rt mismatch: got %p, want %p", got.rt, rt)
	}
}

func Test_knowledgeResolver_resolve(t *testing.T) {
	tests := []struct {
		name    string
		wd      string
		fsys    fstest.MapFS
		want    string
		wantErr error
	}{
		{
			name: "no explicit + knowledge at cwd",
			wd:   "/repo/proj",
			fsys: fstest.MapFS{
				"repo/proj/knowledge": {Mode: fs.ModeDir | 0o755},
			},
			want: "/repo/proj/knowledge",
		},
		{
			name: "no explicit + knowledge at parent",
			wd:   "/repo/proj/inner",
			fsys: fstest.MapFS{
				"repo/proj/knowledge": {Mode: fs.ModeDir | 0o755},
			},
			want: "/repo/proj/knowledge",
		},
		{
			name:    "not found returns ErrKnowledgeDirNotFound",
			wd:      "/repo/proj",
			fsys:    fstest.MapFS{"repo/proj/main.go": {}},
			wantErr: ErrKnowledgeDirNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newKnowledgeResolver(newTestRuntime(nil, tt.wd, tt.fsys))
			got, err := r.resolve()
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
