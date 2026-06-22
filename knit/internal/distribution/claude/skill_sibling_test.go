package claude

import (
	"bytes"
	"context"
	"io/fs"
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

func TestBuilder_skillWithSiblings(t *testing.T) {
	t.Parallel()
	enabled := true
	asset1 := mustSkillAssetT(t, "scripts/run.sh", []byte("echo hi\n"))
	asset2 := mustSkillAssetT(t, "refs/sub/notes.md", []byte("# notes\n"))
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:    "p.skill.orchestrator",
				Kind:  source.KindSkill,
				Name:  "p-orchestrator",
				Path:  "skills/orchestrator",
				Body:  []byte("body\n"),
				Skill: mustSkillMetaT(t, "skills/orchestrator", asset1, asset2),
				Tools: map[source.Target]source.ToolConfig{
					Target: {Enabled: &enabled},
				},
			},
		},
	}
	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 3 {
		t.Fatalf("len(arts) = %d, want 3", len(arts))
	}
	paths := make([]string, len(arts))
	for i, a := range arts {
		paths[i] = a.Path
		if len(a.SourceEntryIDs) != 1 || a.SourceEntryIDs[0] != "p.skill.orchestrator" {
			t.Errorf("artifact %s SourceEntryIDs = %v, want [p.skill.orchestrator]", a.Path, a.SourceEntryIDs)
		}
	}
	sort.Strings(paths)
	want := []string{
		"skills/p-orchestrator/SKILL.md",
		"skills/p-orchestrator/refs/sub/notes.md",
		"skills/p-orchestrator/scripts/run.sh",
	}
	for i, w := range want {
		if paths[i] != w {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], w)
		}
	}
}

func TestBuilder_skillSiblingModeIsZero(t *testing.T) {
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
	for _, a := range arts {
		if a.Mode != fs.FileMode(0) {
			t.Errorf("artifact %s Mode = %v, want 0", a.Path, a.Mode)
		}
	}
}

func TestBuilder_skillFrontmatterMergeOrder(t *testing.T) {
	t.Parallel()
	enabled := true
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.skill.a",
				Kind:        source.KindSkill,
				Name:        "p-a",
				Description: "neutral desc",
				Path:        "skills/a",
				Body:        []byte("body\n"),
				Skill:       mustSkillMetaT(t, "skills/a"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Enabled: &enabled,
						Frontmatter: map[string]any{
							"description": "target desc",
							"extra":       "added",
						},
					},
				},
			},
		},
	}
	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("len(arts) = %d, want 1", len(arts))
	}
	body := arts[0].Content
	if !bytes.Contains(body, []byte("description: target desc")) {
		t.Errorf("target frontmatter did not override description; content=%s", body)
	}
	if bytes.Contains(body, []byte("description: neutral desc")) {
		t.Errorf("neutral description still present; content=%s", body)
	}
	if !bytes.Contains(body, []byte("extra: added")) {
		t.Errorf("target-added frontmatter missing; content=%s", body)
	}
}

func TestBuilder_skillNotEnabledProducesNoArtifacts(t *testing.T) {
	t.Parallel()
	disabled := false
	asset := mustSkillAssetT(t, "scripts/run.sh", []byte("x"))
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{},
		Entries: []source.Entry{
			{
				ID:    "p.skill.a",
				Kind:  source.KindSkill,
				Name:  "p-a",
				Path:  "skills/a",
				Body:  []byte("body\n"),
				Skill: mustSkillMetaT(t, "skills/a", asset),
				Tools: map[source.Target]source.ToolConfig{Target: {Enabled: &disabled}},
			},
		},
	}
	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 0 {
		t.Errorf("len(arts) = %d, want 0", len(arts))
	}
}
