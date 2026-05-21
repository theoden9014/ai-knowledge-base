package inventory

import (
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestNewInstallationIDFromArtifactPath(t *testing.T) {
	type args struct {
		rel source.ArtifactPath
	}
	tests := []struct {
		name    string
		args    args
		want    InstallationID
		wantErr error
	}{
		{name: "zero rel rejected", args: args{rel: source.ArtifactPath{}}, want: InstallationID(""), wantErr: ErrInvalidInstallationID},
		{name: "skill artifact path accepted", args: args{rel: must(source.NewArtifactPath("skills/foo/SKILL.md"))}, want: InstallationID("skills/foo/SKILL.md"), wantErr: nil},
		{name: "flat artifact path accepted", args: args{rel: must(source.NewArtifactPath("AGENTS.md"))}, want: InstallationID("AGENTS.md"), wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInstallationIDFromArtifactPath(tt.args.rel)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewInstallationIDFromArtifactPath() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NewInstallationIDFromArtifactPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstallationID_EncodedBaseName(t *testing.T) {
	tests := []struct {
		name string
		id   InstallationID
		want string
	}{
		{name: "empty stays empty", id: InstallationID(""), want: ""},
		{name: "single segment unchanged", id: InstallationID("AGENTS.md"), want: "AGENTS.md"},
		{name: "two segments encoded", id: InstallationID("agents/x.toml"), want: "agents_x.toml"},
		{name: "three segments encoded", id: InstallationID("skills/foo/SKILL.md"), want: "skills_foo_SKILL.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.EncodedBaseName(); got != tt.want {
				t.Errorf("InstallationID.EncodedBaseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstallationIDFromBaseName(t *testing.T) {
	type args struct {
		base string
	}
	tests := []struct {
		name  string
		args  args
		want  InstallationID
		want1 bool
	}{
		{name: "empty returns false", args: args{base: ""}, want: InstallationID(""), want1: false},
		{name: "single segment unchanged", args: args{base: "AGENTS.md"}, want: InstallationID("AGENTS.md"), want1: true},
		{name: "two segments decoded", args: args{base: "agents_x.toml"}, want: InstallationID("agents/x.toml"), want1: true},
		{name: "three segments decoded", args: args{base: "skills_foo_SKILL.md"}, want: InstallationID("skills/foo/SKILL.md"), want1: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := InstallationIDFromBaseName(tt.args.base)
			if got != tt.want {
				t.Errorf("InstallationIDFromBaseName() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("InstallationIDFromBaseName() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestInstallationID_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "flat", path: "AGENTS.md"},
		{name: "two segments", path: "agents/x.toml"},
		{name: "three segments", path: "skills/foo/SKILL.md"},
		{name: "deep nested", path: "deep/nested/path/with-hyphen/file.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := must(source.NewArtifactPath(tt.path))
			id, err := NewInstallationIDFromArtifactPath(ap)
			if err != nil {
				t.Fatalf("NewInstallationIDFromArtifactPath() unexpected error: %v", err)
			}
			base := id.EncodedBaseName()
			roundtrip, ok := InstallationIDFromBaseName(base)
			if !ok {
				t.Fatalf("InstallationIDFromBaseName() returned false for %q", base)
			}
			if roundtrip != id {
				t.Errorf("round-trip mismatch: original=%q encoded=%q decoded=%q", id, base, roundtrip)
			}
		})
	}
}
