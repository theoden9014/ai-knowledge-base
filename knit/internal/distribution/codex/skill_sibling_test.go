package codex

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"sigs.k8s.io/yaml"
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

func TestCodexBuilder_manualSkillWritesOpenAIPolicy(t *testing.T) {
	t.Parallel()
	metadata := mustSkillAssetT(t, "agents/openai.yaml", []byte(
		"interface:\n  display_name: Existing\npolicy:\n  allow_implicit_invocation: true\n",
	))
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{{
			ID:          "p.skill.manual",
			Kind:        source.KindSkill,
			Name:        "p-manual",
			Description: "manual skill",
			Path:        "skills/manual",
			Body:        []byte("body\n"),
			Skill:       mustManualSkillMetaT(t, "skills/manual", metadata),
		}},
	}

	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 2 {
		t.Fatalf("len(arts) = %d, want 2", len(arts))
	}
	var metadataContent []byte
	for _, art := range arts {
		if art.Path == "skills/p-manual/agents/openai.yaml" {
			metadataContent = art.Content
		}
	}
	if metadataContent == nil {
		t.Fatal("agents/openai.yaml artifact not found")
	}
	var got map[string]any
	if err := yaml.Unmarshal(metadataContent, &got); err != nil {
		t.Fatalf("openai.yaml is invalid YAML: %v", err)
	}
	policy, ok := got["policy"].(map[string]any)
	if !ok {
		t.Fatalf("policy = %#v, want mapping", got["policy"])
	}
	if got := policy["allow_implicit_invocation"]; got != false {
		t.Errorf("allow_implicit_invocation = %#v, want false", got)
	}
	if _, ok := got["interface"]; !ok {
		t.Errorf("existing interface metadata was not preserved: %#v", got)
	}
}

func TestCodexBuilder_manualSkillCreatesOpenAIPolicy(t *testing.T) {
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

	arts, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if len(arts) != 2 {
		t.Fatalf("len(arts) = %d, want 2", len(arts))
	}
	if arts[1].Path != "skills/p-manual/agents/openai.yaml" {
		t.Errorf("metadata path = %q, want skills/p-manual/agents/openai.yaml", arts[1].Path)
	}
}

func TestCodexBuilder_manualSkillRejectsInvalidOpenAIMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed yaml", content: "interface: ["},
		{name: "document is null", content: "null\n"},
		{name: "policy is not mapping", content: "policy: false\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := mustSkillAssetT(t, "agents/openai.yaml", []byte(tt.content))
			pack := &source.Pack{
				Name:         "p",
				DefaultTools: []source.Target{Target},
				Entries: []source.Entry{{
					ID:    "p.skill.manual",
					Kind:  source.KindSkill,
					Name:  "p-manual",
					Path:  "skills/manual",
					Body:  []byte("body\n"),
					Skill: mustManualSkillMetaT(t, "skills/manual", metadata),
				}},
			}
			_, err := NewBuilder().Build(context.Background(), pack)
			if !errors.Is(err, ErrInvalidSkillMetadata) {
				t.Fatalf("Build err = %v, want ErrInvalidSkillMetadata", err)
			}
		})
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
