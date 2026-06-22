package codex

import (
	"context"
	"sort"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func mustSkillMetaT(t *testing.T, root string, assets ...source.SkillAsset) *source.SkillMeta {
	t.Helper()
	m, err := source.NewSkillMeta(root, assets)
	if err != nil {
		t.Fatalf("NewSkillMeta: %v", err)
	}
	return m
}

func mustSkillAssetT(t *testing.T, p string, content []byte) source.SkillAsset {
	t.Helper()
	a, err := source.NewSkillAsset(p, content)
	if err != nil {
		t.Fatalf("NewSkillAsset(%q): %v", p, err)
	}
	return a
}

func TestCodexBuilder_skillWithSiblings(t *testing.T) {
	t.Parallel()
	enabled := true
	asset := mustSkillAssetT(t, "scripts/run.sh", []byte("x"))
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:    "p.skill.a",
				Kind:  source.KindSkill,
				Name:  "p-a",
				Path:  "skills/a",
				Body:  []byte("body\n"),
				Skill: mustSkillMetaT(t, "skills/a", asset),
				Tools: map[source.Target]source.ToolConfig{Target: {Enabled: &enabled}},
			},
		},
	}
	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 2 {
		t.Fatalf("len(arts) = %d, want 2", len(arts))
	}
	paths := []string{arts[0].Path, arts[1].Path}
	sort.Strings(paths)
	want := []string{"skills/p-a/SKILL.md", "skills/p-a/scripts/run.sh"}
	for i, w := range want {
		if paths[i] != w {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], w)
		}
	}
}

func TestCodexPathPolicy_skillSiblingsAccepted(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		p    string
		ok   bool
	}{
		{"body", "skills/orchestrator/SKILL.md", true},
		{"asset subdir", "skills/orchestrator/scripts/run.sh", true},
		{"asset deep nest", "skills/orchestrator/refs/a/b/c.md", true},
		{"bare skill dir", "skills/orchestrator", false},
		{"empty name", "skills//SKILL.md", false},
	}
	pp := newPathPolicy()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ap, err := source.NewArtifactPath(tc.p)
			if err != nil {
				if tc.ok {
					t.Fatalf("NewArtifactPath(%q) err = %v", tc.p, err)
				}
				return
			}
			err = pp.Validate(ap)
			if tc.ok && err != nil {
				t.Errorf("Validate(%q) err = %v, want nil", tc.p, err)
			}
			if !tc.ok && err == nil {
				t.Errorf("Validate(%q) err = nil, want error", tc.p)
			}
		})
	}
}
