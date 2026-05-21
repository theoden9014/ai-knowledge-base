package inventory

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestNewInventoryRoot(t *testing.T) {
	type args struct {
		abs string
	}
	tests := []struct {
		name    string
		args    args
		want    InventoryRoot
		wantErr error
	}{
		{name: "empty rejected", args: args{abs: ""}, want: InventoryRoot{}, wantErr: ErrInvalidInventoryRoot},
		{name: "relative rejected", args: args{abs: "relative/path"}, want: InventoryRoot{}, wantErr: ErrInvalidInventoryRoot},
		{name: "absolute accepted", args: args{abs: "/home/user/.codex"}, want: InventoryRoot{abs: "/home/user/.codex"}, wantErr: nil},
		{name: "absolute cleaned", args: args{abs: "/home/user/./.codex/"}, want: InventoryRoot{abs: "/home/user/.codex"}, wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInventoryRoot(tt.args.abs)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewInventoryRoot() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InventoryRoot{})); diff != "" {
				t.Errorf("NewInventoryRoot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInventoryRoot_String(t *testing.T) {
	type fields struct {
		abs string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value returns empty", fields: fields{}, want: ""},
		{name: "absolute returns stored path", fields: fields{abs: "/home/user/.codex"}, want: "/home/user/.codex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := InventoryRoot{
				abs: tt.fields.abs,
			}
			if got := r.String(); got != tt.want {
				t.Errorf("InventoryRoot.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryRoot_IsZero(t *testing.T) {
	type fields struct {
		abs string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "zero value is zero", fields: fields{}, want: true},
		{name: "non-empty is not zero", fields: fields{abs: "/home"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := InventoryRoot{
				abs: tt.fields.abs,
			}
			if got := r.IsZero(); got != tt.want {
				t.Errorf("InventoryRoot.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryRoot_Join(t *testing.T) {
	root := must(NewInventoryRoot("/home/user/.codex"))
	skillPath := must(source.NewArtifactPath("skills/foo/SKILL.md"))
	flatPath := must(source.NewArtifactPath("AGENTS.md"))
	type fields struct {
		root InventoryRoot
	}
	type args struct {
		rel source.ArtifactPath
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    AbsoluteArtifactPath
		wantErr error
	}{
		{
			name:    "skill artifact joined",
			fields:  fields{root: root},
			args:    args{rel: skillPath},
			want:    AbsoluteArtifactPath{root: root, rel: skillPath, abs: filepath.FromSlash("/home/user/.codex/skills/foo/SKILL.md")},
			wantErr: nil,
		},
		{
			name:    "flat file joined",
			fields:  fields{root: root},
			args:    args{rel: flatPath},
			want:    AbsoluteArtifactPath{root: root, rel: flatPath, abs: filepath.FromSlash("/home/user/.codex/AGENTS.md")},
			wantErr: nil,
		},
		{
			name:    "zero root rejected",
			fields:  fields{root: InventoryRoot{}},
			args:    args{rel: skillPath},
			want:    AbsoluteArtifactPath{},
			wantErr: ErrInvalidInventoryRoot,
		},
		{
			name:    "zero rel rejected",
			fields:  fields{root: root},
			args:    args{rel: source.ArtifactPath{}},
			want:    AbsoluteArtifactPath{},
			wantErr: source.ErrInvalidArtifactPath,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.root.Join(tt.args.rel)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("InventoryRoot.Join() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(AbsoluteArtifactPath{}, InventoryRoot{}, source.ArtifactPath{})); diff != "" {
				t.Errorf("InventoryRoot.Join() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewInventoryRoots(t *testing.T) {
	userRoot := must(NewInventoryRoot("/home/user/.codex"))
	projectRoot := must(NewInventoryRoot("/repo/.codex"))
	type args struct {
		userRoot    InventoryRoot
		projectRoot InventoryRoot
	}
	tests := []struct {
		name    string
		args    args
		want    InventoryRoots
		wantErr error
	}{
		{name: "user-only accepted", args: args{userRoot: userRoot, projectRoot: InventoryRoot{}}, want: InventoryRoots{user: userRoot, project: InventoryRoot{}}, wantErr: nil},
		{name: "user and project accepted", args: args{userRoot: userRoot, projectRoot: projectRoot}, want: InventoryRoots{user: userRoot, project: projectRoot}, wantErr: nil},
		{name: "zero user rejected", args: args{userRoot: InventoryRoot{}, projectRoot: projectRoot}, want: InventoryRoots{}, wantErr: ErrInvalidInventoryRoot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInventoryRoots(tt.args.userRoot, tt.args.projectRoot)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewInventoryRoots() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InventoryRoots{}, InventoryRoot{})); diff != "" {
				t.Errorf("NewInventoryRoots() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInventoryRoots_For(t *testing.T) {
	userRoot := must(NewInventoryRoot("/home/user/.codex"))
	projectRoot := must(NewInventoryRoot("/repo/.codex"))
	withProject := must(NewInventoryRoots(userRoot, projectRoot))
	userOnly := must(NewInventoryRoots(userRoot, InventoryRoot{}))
	type fields struct {
		roots InventoryRoots
	}
	type args struct {
		scope Scope
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    InventoryRoot
		wantErr error
	}{
		{name: "user scope returns user root", fields: fields{roots: withProject}, args: args{scope: ScopeUser}, want: userRoot, wantErr: nil},
		{name: "project scope returns project root", fields: fields{roots: withProject}, args: args{scope: ScopeProject}, want: projectRoot, wantErr: nil},
		{name: "project requested but unconfigured", fields: fields{roots: userOnly}, args: args{scope: ScopeProject}, want: InventoryRoot{}, wantErr: ErrProjectRootNotConfigured},
		{name: "invalid scope rejected", fields: fields{roots: withProject}, args: args{scope: Scope("bogus")}, want: InventoryRoot{}, wantErr: ErrInvalidScope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.roots.For(tt.args.scope)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("InventoryRoots.For() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InventoryRoot{})); diff != "" {
				t.Errorf("InventoryRoots.For() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAbsoluteArtifactPath_String(t *testing.T) {
	type fields struct {
		abs string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero returns empty", fields: fields{}, want: ""},
		{name: "non-zero returns stored", fields: fields{abs: "/x/y"}, want: "/x/y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := AbsoluteArtifactPath{
				abs: tt.fields.abs,
			}
			if got := p.String(); got != tt.want {
				t.Errorf("AbsoluteArtifactPath.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAbsoluteArtifactPath_IsZero(t *testing.T) {
	type fields struct {
		abs string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "zero is zero", fields: fields{}, want: true},
		{name: "non-empty is not zero", fields: fields{abs: "/x"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := AbsoluteArtifactPath{
				abs: tt.fields.abs,
			}
			if got := p.IsZero(); got != tt.want {
				t.Errorf("AbsoluteArtifactPath.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAbsoluteArtifactPath_Root(t *testing.T) {
	root := must(NewInventoryRoot("/home/user/.codex"))
	type fields struct {
		root InventoryRoot
	}
	tests := []struct {
		name   string
		fields fields
		want   InventoryRoot
	}{
		{name: "returns embedded root", fields: fields{root: root}, want: root},
		{name: "zero root returned for zero value", fields: fields{}, want: InventoryRoot{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := AbsoluteArtifactPath{
				root: tt.fields.root,
			}
			if diff := cmp.Diff(tt.want, p.Root(), cmp.AllowUnexported(InventoryRoot{})); diff != "" {
				t.Errorf("AbsoluteArtifactPath.Root() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAbsoluteArtifactPath_RelativePath(t *testing.T) {
	rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
	type fields struct {
		rel source.ArtifactPath
	}
	tests := []struct {
		name   string
		fields fields
		want   source.ArtifactPath
	}{
		{name: "returns embedded rel", fields: fields{rel: rel}, want: rel},
		{name: "zero rel returned for zero value", fields: fields{}, want: source.ArtifactPath{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := AbsoluteArtifactPath{
				rel: tt.fields.rel,
			}
			if diff := cmp.Diff(tt.want, p.RelativePath(), cmp.AllowUnexported(source.ArtifactPath{})); diff != "" {
				t.Errorf("AbsoluteArtifactPath.RelativePath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
