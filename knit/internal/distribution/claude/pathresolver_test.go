package claude

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
			args: args{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			want: &pathResolver{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
		},
		{
			name: "project root empty allowed",
			args: args{userRoot: "/u/.claude", projectRoot: ""},
			want: &pathResolver{userRoot: "/u/.claude", projectRoot: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newPathResolver(tt.args.userRoot, tt.args.projectRoot); !cmp.Equal(tt.want, got, cmp.AllowUnexported(pathResolver{})) {
				t.Errorf("newPathResolver() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got, cmp.AllowUnexported(pathResolver{})))
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
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser},
			want:    "/u/.claude",
			wantErr: nil,
		},
		{
			name:    "project scope returns projectRoot",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeProject},
			want:    "/p/.claude",
			wantErr: nil,
		},
		{
			name:    "invalid scope returns ErrInvalidScope",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.Scope("__bogus__")},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "project scope without projectRoot returns ErrProjectRootNotConfigured",
			fields:  fields{userRoot: "/u/.claude", projectRoot: ""},
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
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "skills/orchestrator/SKILL.md"},
			want:    filepath.Join("/u/.claude", "skills/orchestrator/SKILL.md"),
			wantErr: nil,
		},
		{
			name:    "agent .md under project scope",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeProject, artifactPath: "agents/solid-reviewer.md"},
			want:    filepath.Join("/p/.claude", "agents/solid-reviewer.md"),
			wantErr: nil,
		},
		{
			name:    "CLAUDE.md at root",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "CLAUDE.md"},
			want:    filepath.Join("/u/.claude", "CLAUDE.md"),
			wantErr: nil,
		},
		{
			name:    "commands prompt file",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "commands/review.md"},
			want:    filepath.Join("/u/.claude", "commands/review.md"),
			wantErr: nil,
		},
		{
			name:    "empty path is invalid",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: ""},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "absolute path is invalid",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "/etc/passwd"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "parent traversal is invalid",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "../etc/passwd"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "unknown top-level segment is invalid",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.ScopeUser, artifactPath: "hooks/foo.md"},
			want:    "",
			wantErr: ErrInvalidArtifactPath,
		},
		{
			name:    "invalid scope precedes invalid path",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.Scope("__bogus__"), artifactPath: "../etc/passwd"},
			want:    "",
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "project scope without projectRoot precedes invalid path",
			fields:  fields{userRoot: "/u/.claude", projectRoot: ""},
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
		{name: "CLAUDE.md", artifactPath: "CLAUDE.md", want: inventory.InstallationID("CLAUDE.md")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{userRoot: "/u/.claude", projectRoot: "/p/.claude"}
			if got := r.installationID(tt.artifactPath); !cmp.Equal(tt.want, got) {
				t.Errorf("pathResolver.installationID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_pathResolver_resolve verifies that resolve performs Scope evaluation
// once and that the methods on the returned *resolved value produce consistent
// results.
func Test_pathResolver_resolve(t *testing.T) {
	type fields struct {
		userRoot    string
		projectRoot string
	}
	type args struct {
		scope inventory.Scope
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantRoot string
		wantErr  error
	}{
		{
			name:     "user scope populates root",
			fields:   fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:     args{scope: inventory.ScopeUser},
			wantRoot: "/u/.claude",
			wantErr:  nil,
		},
		{
			name:     "project scope populates root",
			fields:   fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:     args{scope: inventory.ScopeProject},
			wantRoot: "/p/.claude",
			wantErr:  nil,
		},
		{
			name:    "invalid scope returns ErrInvalidScope",
			fields:  fields{userRoot: "/u/.claude", projectRoot: "/p/.claude"},
			args:    args{scope: inventory.Scope("__bogus__")},
			wantErr: inventory.ErrInvalidScope,
		},
		{
			name:    "project scope without projectRoot returns ErrProjectRootNotConfigured",
			fields:  fields{userRoot: "/u/.claude", projectRoot: ""},
			args:    args{scope: inventory.ScopeProject},
			wantErr: ErrProjectRootNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pathResolver{
				userRoot:    tt.fields.userRoot,
				projectRoot: tt.fields.projectRoot,
			}
			got, err := r.resolve(tt.args.scope)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("pathResolver.resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Scope() != tt.args.scope {
				t.Errorf("resolved.Scope() = %v, want %v", got.Scope(), tt.args.scope)
			}
			if got.Root() != tt.wantRoot {
				t.Errorf("resolved.Root() = %v, want %v", got.Root(), tt.wantRoot)
			}
			if abs, aErr := got.ResolveArtifactPath("CLAUDE.md"); aErr != nil {
				t.Errorf("ResolveArtifactPath() unexpected error: %v", aErr)
			} else if abs != filepath.Join(tt.wantRoot, "CLAUDE.md") {
				t.Errorf("ResolveArtifactPath() = %v, want %v", abs, filepath.Join(tt.wantRoot, "CLAUDE.md"))
			}
		})
	}
}

// Test_pathResolver_resolve_artifactValidation verifies that ResolveArtifactPath
// validates only the artifact path and does not re-evaluate Scope.
func Test_pathResolver_resolve_artifactValidation(t *testing.T) {
	r := &pathResolver{userRoot: "/u/.claude", projectRoot: "/p/.claude"}
	rv, err := r.resolve(inventory.ScopeUser)
	if err != nil {
		t.Fatalf("resolve(): %v", err)
	}
	tests := []struct {
		name    string
		relPath string
		want    string
		wantErr error
	}{
		{name: "valid CLAUDE.md", relPath: "CLAUDE.md", want: filepath.Join("/u/.claude", "CLAUDE.md")},
		{name: "valid skill path", relPath: "skills/p-s/SKILL.md", want: filepath.Join("/u/.claude", "skills/p-s/SKILL.md")},
		{name: "empty path is invalid", relPath: "", wantErr: ErrInvalidArtifactPath},
		{name: "absolute path is invalid", relPath: "/etc/passwd", wantErr: ErrInvalidArtifactPath},
		{name: "parent traversal is invalid", relPath: "../escape.md", wantErr: ErrInvalidArtifactPath},
		{name: "unknown top-level is invalid", relPath: "hooks/foo.md", wantErr: ErrInvalidArtifactPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rv.ResolveArtifactPath(tt.relPath)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveArtifactPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Errorf("ResolveArtifactPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
