package source

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewArtifactPath(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    ArtifactPath
		wantErr error
	}{
		{name: "empty rejected", args: args{s: ""}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "absolute single slash rejected", args: args{s: "/"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "absolute path rejected", args: args{s: "/foo"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "double-leading-slash rejected", args: args{s: "//x"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "dotdot bare segment rejected", args: args{s: ".."}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "dotdot leading rejected", args: args{s: "../x"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "dotdot middle rejected", args: args{s: "a/../b"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "dotdot trailing rejected", args: args{s: "a/.."}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "NUL byte rejected", args: args{s: "a\x00b"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "backslash rejected", args: args{s: "a\\b"}, want: ArtifactPath{}, wantErr: ErrInvalidArtifactPath},
		{name: "skill artifact accepted", args: args{s: "skills/foo/SKILL.md"}, want: ArtifactPath{value: "skills/foo/SKILL.md"}, wantErr: nil},
		{name: "flat file accepted", args: args{s: "AGENTS.md"}, want: ArtifactPath{value: "AGENTS.md"}, wantErr: nil},
		{name: "agent toml accepted", args: args{s: "agents/x.toml"}, want: ArtifactPath{value: "agents/x.toml"}, wantErr: nil},
		{name: "deep nested with hyphen accepted", args: args{s: "deep/nested/path/with-hyphen/file.md"}, want: ArtifactPath{value: "deep/nested/path/with-hyphen/file.md"}, wantErr: nil},
		{name: "triple-dot segment accepted", args: args{s: "..."}, want: ArtifactPath{value: "..."}, wantErr: nil},
		{name: "dot-prefixed name accepted", args: args{s: ".hidden"}, want: ArtifactPath{value: ".hidden"}, wantErr: nil},
		{name: "double-dot-prefixed name accepted", args: args{s: "..foo"}, want: ArtifactPath{value: "..foo"}, wantErr: nil},
		{name: "double-dot-suffixed name accepted", args: args{s: "foo.."}, want: ArtifactPath{value: "foo.."}, wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewArtifactPath(tt.args.s)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewArtifactPath() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ArtifactPath{})); diff != "" {
				t.Errorf("NewArtifactPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestArtifactPath_String(t *testing.T) {
	type fields struct {
		value string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value returns empty", fields: fields{value: ""}, want: ""},
		{name: "non-zero returns underlying", fields: fields{value: "skills/foo/SKILL.md"}, want: "skills/foo/SKILL.md"},
		{name: "flat file returns as-is", fields: fields{value: "AGENTS.md"}, want: "AGENTS.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ArtifactPath{
				value: tt.fields.value,
			}
			if got := p.String(); got != tt.want {
				t.Errorf("ArtifactPath.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactPath_IsZero(t *testing.T) {
	type fields struct {
		value string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "zero value is zero", fields: fields{value: ""}, want: true},
		{name: "non-empty is not zero", fields: fields{value: "AGENTS.md"}, want: false},
		{name: "deep path is not zero", fields: fields{value: "skills/foo/SKILL.md"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ArtifactPath{
				value: tt.fields.value,
			}
			if got := p.IsZero(); got != tt.want {
				t.Errorf("ArtifactPath.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactPath_TopSegment(t *testing.T) {
	type fields struct {
		value string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value yields empty", fields: fields{value: ""}, want: ""},
		{name: "no slash returns whole path", fields: fields{value: "AGENTS.md"}, want: "AGENTS.md"},
		{name: "first segment returned", fields: fields{value: "skills/foo/SKILL.md"}, want: "skills"},
		{name: "single-segment with trailing slash returns prefix", fields: fields{value: "skills/"}, want: "skills"},
		{name: "two-segment returns first", fields: fields{value: "agents/x.toml"}, want: "agents"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ArtifactPath{
				value: tt.fields.value,
			}
			if got := p.TopSegment(); got != tt.want {
				t.Errorf("ArtifactPath.TopSegment() = %v, want %v", got, tt.want)
			}
		})
	}
}
