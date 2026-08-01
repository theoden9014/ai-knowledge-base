package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestDefaultRoots(t *testing.T) {
	got := DefaultRoots("/home/user", "/repo", "/custom/codex")
	want := Roots{
		UserSkills:    "/home/user/.agents",
		ProjectSkills: "/repo/.agents",
		UserAgents:    "/custom/codex",
		ProjectAgents: "/repo/.codex",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DefaultRoots mismatch (-want +got):\n%s", diff)
	}
}

func TestArtifactResolverRoutesByArtifactFamily(t *testing.T) {
	roots := Roots{
		UserSkills:    "/home/user/.agents",
		ProjectSkills: "/repo/.agents",
		UserAgents:    "/home/user/.codex",
		ProjectAgents: "/repo/.codex",
	}
	resolver := must(buildResolver(roots))
	tests := []struct {
		name  string
		scope inventory.Scope
		path  string
		want  string
	}{
		{"user skill", inventory.ScopeUser, "skills/a/SKILL.md", "/home/user/.agents/skills/a/SKILL.md"},
		{"project skill", inventory.ScopeProject, "skills/a/SKILL.md", "/repo/.agents/skills/a/SKILL.md"},
		{"user agent", inventory.ScopeUser, "agents/a.toml", "/home/user/.codex/agents/a.toml"},
		{"project agent", inventory.ScopeProject, "agents/a.toml", "/repo/.codex/agents/a.toml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := must(source.NewArtifactPath(tt.path))
			got := must(resolver.Resolve(tt.scope, p))
			if got.String() != tt.want {
				t.Errorf("Resolve() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestArtifactResolverRejectsPartialProjectRoots(t *testing.T) {
	_, err := buildResolver(Roots{
		UserSkills:    "/home/user/.agents",
		UserAgents:    "/home/user/.codex",
		ProjectSkills: "/repo/.agents",
	})
	if err == nil {
		t.Fatal("buildResolver() err = nil, want partial project roots rejected")
	}
}

func TestInstallerWithRootsWritesSkillsAndAgentsSeparately(t *testing.T) {
	base := t.TempDir()
	roots := DefaultRoots(filepath.Join(base, "user"), filepath.Join(base, "project"), "")
	labels := inventory.NewSidecarLabelStore(
		Target,
		filepath.Join(base, "user-labels"),
		filepath.Join(base, "project-labels"),
	)
	installer := must(NewInstallerWithRoots(roots, labels))
	lister := must(NewListerWithRoots(roots, labels))
	uninstaller := must(NewUninstallerWithRoots(roots, labels))
	artifacts := []source.Artifact{
		{Target: Target, Path: "skills/a/SKILL.md", Content: []byte("skill")},
		{Target: Target, Path: "agents/a.toml", Content: []byte("agent")},
	}
	var installations []inventory.Installation
	for _, artifact := range artifacts {
		installed, err := installer.Install(context.Background(), inventory.ScopeUser, artifact)
		if err != nil {
			t.Fatalf("Install(%q): %v", artifact.Path, err)
		}
		installations = append(installations, installed)
	}

	assertFileContent(t, filepath.Join(roots.UserSkills, "skills/a/SKILL.md"), "skill")
	assertFileContent(t, filepath.Join(roots.UserAgents, "agents/a.toml"), "agent")

	listed, err := lister.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(listed))
	}

	for _, installed := range installations {
		if err := uninstaller.Uninstall(context.Background(), installed); err != nil {
			t.Fatalf("Uninstall(%q): %v", installed.Artifact.Path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(roots.UserSkills, "skills/a/SKILL.md"),
		filepath.Join(roots.UserAgents, "agents/a.toml"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("artifact %q still exists after uninstall: %v", path, err)
		}
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(got) != want {
		t.Errorf("ReadFile(%q) = %q, want %q", path, got, want)
	}
}
