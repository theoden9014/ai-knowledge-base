package gemini

import (
	"context"
	"errors"
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

func mustManualSkillMetaT(t *testing.T, root string, assets ...source.SkillAsset) *source.SkillMeta {
	t.Helper()
	m, err := source.NewSkillMetaWithInvocation(root, assets, source.SkillInvocationManual)
	if err != nil {
		t.Fatalf("NewSkillMetaWithInvocation: %v", err)
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

func TestGeminiBuilder_skillWithSiblings(t *testing.T) {
	t.Parallel()
	enabled := true
	asset := mustSkillAssetT(t, "scripts/run.sh", []byte("echo hi\n"))
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
	for _, a := range arts {
		if len(a.SourceEntryIDs) != 1 || a.SourceEntryIDs[0] != "p.skill.a" {
			t.Errorf("artifact %s SourceEntryIDs = %v, want [p.skill.a]", a.Path, a.SourceEntryIDs)
		}
	}
}

func TestGeminiBuilder_manualSkillIsUnsupported(t *testing.T) {
	t.Parallel()
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{{
			ID:    "p.skill.manual",
			Kind:  source.KindSkill,
			Name:  "p-manual",
			Path:  "skills/manual",
			Body:  []byte("body\n"),
			Skill: mustManualSkillMetaT(t, "skills/manual"),
		}},
	}

	_, err := NewBuilder().Build(context.Background(), pack)
	if !errors.Is(err, ErrUnsupportedSkillInvocation) {
		t.Fatalf("Build err = %v, want ErrUnsupportedSkillInvocation", err)
	}
}

func TestGeminiBuilder_disabledManualSkillIsSkipped(t *testing.T) {
	t.Parallel()
	disabled := false
	pack := &source.Pack{
		Name: "p",
		Entries: []source.Entry{{
			ID:    "p.skill.manual",
			Kind:  source.KindSkill,
			Name:  "p-manual",
			Path:  "skills/manual",
			Body:  []byte("body\n"),
			Skill: mustManualSkillMetaT(t, "skills/manual"),
			Tools: map[source.Target]source.ToolConfig{
				Target: {Enabled: &disabled},
			},
		}},
	}

	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err = %v, want nil for disabled entry", err)
	}
	if len(arts) != 0 {
		t.Errorf("len(arts) = %d, want 0", len(arts))
	}
}
