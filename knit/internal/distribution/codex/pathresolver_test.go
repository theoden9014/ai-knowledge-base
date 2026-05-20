package codex

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

func Test_pathResolver_root(t *testing.T) {
	type fields struct {
		userRoot    string
		projectRoot string
	}
	type args struct {
		scope inventory.Scope
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name:    "ScopeUser returns userRoot",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser},
			want:    "/u/.codex",
			wantErr: nil,
		},
		{
			name:    "ScopeProject returns projectRoot",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeProject},
			want:    "/p/.codex",
			wantErr: nil,
		},
		{
			name:    "ScopeProject with empty projectRoot returns ErrProjectRootNotConfigured",
			fields:  fields{userRoot: "/u/.codex", projectRoot: ""},
			args:    args{scope: inventory.ScopeProject},
			want:    "",
			wantErr: ErrProjectRootNotConfigured,
		},
		{
			name:    "invalid scope returns ErrInvalidScope",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.Scope("nope")},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPathResolver(tt.fields.userRoot, tt.fields.projectRoot)
			got, err := r.root(tt.args.scope)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("root() err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("root() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_pathResolver_resolveArtifactPath(t *testing.T) {
	type fields struct {
		userRoot    string
		projectRoot string
	}
	type args struct {
		scope        inventory.Scope
		artifactPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name:    "valid SKILL.md under skills/<name>/",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "skills/orchestrator/SKILL.md"},
			want:    "/u/.codex/skills/orchestrator/SKILL.md",
			wantErr: nil,
		},
		{
			name:    "valid agents/<name>.toml",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeProject, artifactPath: "agents/solid-reviewer.toml"},
			want:    "/p/.codex/agents/solid-reviewer.toml",
			wantErr: nil,
		},
		{
			name:    "valid prompts/<name>.md (flat)",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "prompts/review.md"},
			want:    "/u/.codex/prompts/review.md",
			wantErr: nil,
		},
		{
			name:    "valid AGENTS.md at root",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "AGENTS.md"},
			want:    "/u/.codex/AGENTS.md",
			wantErr: nil,
		},
		{
			name:    "empty path returns ErrInvalidArtifactPath",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: ""},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "absolute path returns ErrInvalidArtifactPath",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "/abs/SKILL.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "parent ref escape returns ErrInvalidArtifactPath",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "skills/../../etc/passwd"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "unknown top segment returns ErrInvalidArtifactPath",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "rules/foo.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "prompts/ subdirectory is rejected (Codex: top-level only)",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "prompts/sub/cmd.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "invalid scope returns ErrInvalidScope",
			fields:  fields{userRoot: "/u/.codex", projectRoot: "/p/.codex"},
			args:    args{scope: inventory.Scope("nope"), artifactPath: "AGENTS.md"},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "ScopeProject with empty projectRoot returns ErrProjectRootNotConfigured",
			fields:  fields{userRoot: "/u/.codex", projectRoot: ""},
			args:    args{scope: inventory.ScopeProject, artifactPath: "AGENTS.md"},
			want:    "",
			wantErr: ErrProjectRootNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPathResolver(tt.fields.userRoot, tt.fields.projectRoot)
			got, err := r.resolveArtifactPath(tt.args.scope, tt.args.artifactPath)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveArtifactPath() err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveArtifactPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_pathResolver_installationID(t *testing.T) {
	tests := []struct {
		name string
		path string
		want inventory.InstallationID
	}{
		{
			name: "skill path",
			path: "skills/orchestrator/SKILL.md",
			want: inventory.InstallationID("skills/orchestrator/SKILL.md"),
		},
		{
			name: "agent path",
			path: "agents/solid-reviewer.toml",
			want: inventory.InstallationID("agents/solid-reviewer.toml"),
		},
		{
			name: "AGENTS.md",
			path: "AGENTS.md",
			want: inventory.InstallationID("AGENTS.md"),
		},
		{
			name: "prompt path",
			path: "prompts/review.md",
			want: inventory.InstallationID("prompts/review.md"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPathResolver("/u/.codex", "/p/.codex")
			if diff := cmp.Diff(tt.want, r.installationID(tt.path)); diff != "" {
				t.Errorf("installationID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Test_pathResolver_resolve verifies that resolve performs scope validation
// once and exposes consistent Root() through the returned resolved.
func Test_pathResolver_resolve(t *testing.T) {
	tests := []struct {
		name        string
		userRoot    string
		projectRoot string
		scope       inventory.Scope
		wantRoot    string
		wantErr     error
	}{
		{
			name:        "ScopeUser",
			userRoot:    "/u/.codex",
			projectRoot: "/p/.codex",
			scope:       inventory.ScopeUser,
			wantRoot:    "/u/.codex",
		},
		{
			name:        "ScopeProject",
			userRoot:    "/u/.codex",
			projectRoot: "/p/.codex",
			scope:       inventory.ScopeProject,
			wantRoot:    "/p/.codex",
		},
		{
			name:        "invalid scope -> ErrInvalidScope",
			userRoot:    "/u/.codex",
			projectRoot: "/p/.codex",
			scope:       inventory.Scope("__bogus__"),
			wantErr:     inventory.ErrInvalidScope,
		},
		{
			name:        "ScopeProject without projectRoot -> ErrProjectRootNotConfigured",
			userRoot:    "/u/.codex",
			projectRoot: "",
			scope:       inventory.ScopeProject,
			wantErr:     ErrProjectRootNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPathResolver(tt.userRoot, tt.projectRoot)
			got, err := r.resolve(tt.scope)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolve() err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Root() != tt.wantRoot {
				t.Errorf("resolved.Root() = %q, want %q", got.Root(), tt.wantRoot)
			}
			if got.Scope() != tt.scope {
				t.Errorf("resolved.Scope() = %q, want %q", got.Scope(), tt.scope)
			}
		})
	}
}
