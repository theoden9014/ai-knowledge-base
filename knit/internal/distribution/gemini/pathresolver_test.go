package gemini

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

func Test_newPathResolver(t *testing.T) {
	type args struct {
		userRoot    string
		projectRoot string
	}
	tests := []struct {
		name string
		args args
		want *pathResolver
	}{
		{
			name: "both roots populated",
			args: args{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			want: &pathResolver{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
		},
		{
			name: "project root empty allowed",
			args: args{userRoot: "/u/.gemini", projectRoot: ""},
			want: &pathResolver{userRoot: "/u/.gemini", projectRoot: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newPathResolver(tt.args.userRoot, tt.args.projectRoot); !cmp.Equal(tt.want, got, cmp.AllowUnexported(pathResolver{})) {
				t.Errorf("newPathResolver() mismatch (-want +got):\n%s", cmp.Diff(tt.want, got, cmp.AllowUnexported(pathResolver{})))
			}
		})
	}
}

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
			name:    "user scope returns userRoot",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser},
			want:    "/u/.gemini",
			wantErr: nil,
		},
		{
			name:    "project scope returns projectRoot",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeProject},
			want:    "/p/.gemini",
			wantErr: nil,
		},
		{
			name:    "invalid scope returns ErrInvalidScope",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.Scope("__bogus__")},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "project scope without projectRoot returns ErrProjectRootNotConfigured",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: ""},
			args:    args{scope: inventory.ScopeProject},
			want:    "",
			wantErr: ErrProjectRootNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{
				userRoot:    tt.fields.userRoot,
				projectRoot: tt.fields.projectRoot,
			}
			got, err := r.root(tt.args.scope)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("pathResolver.root() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Errorf("pathResolver.root() = %v, want %v", got, tt.want)
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
			name:    "skill SKILL.md under user scope",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "skills/orchestrator/SKILL.md"},
			want:    filepath.Join("/u/.gemini", "skills/orchestrator/SKILL.md"),
			wantErr: nil,
		},
		{
			name:    "agent .md under project scope",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeProject, artifactPath: "agents/solid-reviewer.md"},
			want:    filepath.Join("/p/.gemini", "agents/solid-reviewer.md"),
			wantErr: nil,
		},
		{
			name:    "GEMINI.md at root",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "GEMINI.md"},
			want:    filepath.Join("/u/.gemini", "GEMINI.md"),
			wantErr: nil,
		},
		{
			name:    "commands prompt TOML file",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "commands/review.toml"},
			want:    filepath.Join("/u/.gemini", "commands/review.toml"),
			wantErr: nil,
		},
		{
			name:    "empty path is invalid",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: ""},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "absolute path is invalid",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "/etc/passwd"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "parent traversal is invalid",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "../etc/passwd"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "unknown top-level segment is invalid",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "hooks/foo.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "CLAUDE.md is not a recognized top-level for gemini",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "CLAUDE.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "invalid scope precedes invalid path",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"},
			args:    args{scope: inventory.Scope("__bogus__"), artifactPath: "../etc/passwd"},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "project scope without projectRoot precedes invalid path",
			fields:  fields{userRoot: "/u/.gemini", projectRoot: ""},
			args:    args{scope: inventory.ScopeProject, artifactPath: "../etc/passwd"},
			want:    "",
			wantErr: ErrProjectRootNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{
				userRoot:    tt.fields.userRoot,
				projectRoot: tt.fields.projectRoot,
			}
			got, err := r.resolveArtifactPath(tt.args.scope, tt.args.artifactPath)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("pathResolver.resolveArtifactPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Errorf("pathResolver.resolveArtifactPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pathResolver_installationID(t *testing.T) {
	tests := []struct {
		name         string
		artifactPath string
		want         inventory.InstallationID
	}{
		{name: "skill path", artifactPath: "skills/orchestrator/SKILL.md", want: inventory.InstallationID("skills/orchestrator/SKILL.md")},
		{name: "GEMINI.md", artifactPath: "GEMINI.md", want: inventory.InstallationID("GEMINI.md")},
		{name: "command toml", artifactPath: "commands/review.toml", want: inventory.InstallationID("commands/review.toml")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"}
			if got := r.installationID(tt.artifactPath); !cmp.Equal(tt.want, got) {
				t.Errorf("pathResolver.installationID() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			name:        "user scope resolved",
			userRoot:    "/u/.gemini",
			projectRoot: "/p/.gemini",
			scope:       inventory.ScopeUser,
			wantRoot:    "/u/.gemini",
		},
		{
			name:        "project scope resolved",
			userRoot:    "/u/.gemini",
			projectRoot: "/p/.gemini",
			scope:       inventory.ScopeProject,
			wantRoot:    "/p/.gemini",
		},
		{
			name:        "invalid scope -> ErrInvalidScope",
			userRoot:    "/u/.gemini",
			projectRoot: "/p/.gemini",
			scope:       inventory.Scope("__bogus__"),
			wantErr:     inventory.ErrInvalidScope,
		},
		{
			name:        "project scope without projectRoot -> ErrProjectRootNotConfigured",
			userRoot:    "/u/.gemini",
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
				t.Fatalf("pathResolver.resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Root() != tt.wantRoot {
				t.Errorf("resolved.Root() = %v, want %v", got.Root(), tt.wantRoot)
			}
			if got.Scope() != tt.scope {
				t.Errorf("resolved.Scope() = %v, want %v", got.Scope(), tt.scope)
			}
		})
	}
}

func Test_resolved_artifactPath(t *testing.T) {
	tests := []struct {
		name         string
		artifactPath string
		want         string
		wantErr      error
	}{
		{name: "skill SKILL.md", artifactPath: "skills/orchestrator/SKILL.md", want: filepath.Join("/u/.gemini", "skills/orchestrator/SKILL.md")},
		{name: "GEMINI.md at root", artifactPath: "GEMINI.md", want: filepath.Join("/u/.gemini", "GEMINI.md")},
		{name: "empty path is invalid", artifactPath: "", wantErr: ErrInvalidArtifactPath},
		{name: "absolute path is invalid", artifactPath: "/etc/passwd", wantErr: ErrInvalidArtifactPath},
		{name: "parent traversal is invalid", artifactPath: "../etc/passwd", wantErr: ErrInvalidArtifactPath},
		{name: "unknown top-level segment is invalid", artifactPath: "hooks/foo.md", wantErr: ErrInvalidArtifactPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{userRoot: "/u/.gemini", projectRoot: "/p/.gemini"}
			rs, err := r.resolve(inventory.ScopeUser)
			if err != nil {
				t.Fatalf("resolve(): %v", err)
			}
			got, err := rs.artifactPath(tt.artifactPath)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolved.artifactPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Errorf("resolved.artifactPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
