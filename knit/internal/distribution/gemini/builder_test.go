package gemini

import (
	"bytes"
	"context"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"sigs.k8s.io/yaml"
)

func boolPtr(b bool) *bool { return &b }

func splitFrontmatter(t *testing.T, b []byte) (map[string]any, []byte) {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("---\n")) {
		t.Fatalf("artifact has no frontmatter:\n%s", b)
	}
	rest := b[len("---\n"):]
	end := bytes.Index(rest, []byte("---\n"))
	if end < 0 {
		t.Fatalf("artifact has unterminated frontmatter:\n%s", b)
	}
	var fm map[string]any
	if err := yaml.Unmarshal(rest[:end], &fm); err != nil {
		t.Fatalf("parse frontmatter: %v", err)
	}
	return fm, rest[end+len("---\n"):]
}

func TestBuilder_Target(t *testing.T) {
	if got := NewBuilder().Target(); got != Target {
		t.Errorf("Target() = %q, want %q", got, Target)
	}
}

func TestBuilder_Build_emptyPack(t *testing.T) {
	got, err := NewBuilder().Build(context.Background(), &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
	})
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Build() returned %d artifacts, want 0", len(got))
	}
}

func TestBuilder_Build_skill(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{{
			ID:          "p.skill.s",
			Kind:        source.KindSkill,
			Name:        "p-s",
			Description: "neutral",
			Body:        []byte("skill body\n"),
			Tools: map[source.Target]source.ToolConfig{
				Target: {Frontmatter: map[string]any{"description": "override"}},
			},
		}},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	if got[0].Target != Target || got[0].Path != "skills/p-s/SKILL.md" {
		t.Errorf("artifact = (%q, %q), want (%q, %q)", got[0].Target, got[0].Path, Target, "skills/p-s/SKILL.md")
	}
	fm, body := splitFrontmatter(t, got[0].Content)
	if fm["name"] != "p-s" || fm["description"] != "override" {
		t.Errorf("frontmatter = %#v", fm)
	}
	if string(body) != "skill body\n" {
		t.Errorf("body = %q", body)
	}
}

func TestBuilder_Build_agent(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{{
			ID:          "p.agent.a",
			Kind:        source.KindAgent,
			Name:        "p-a",
			Description: "agent",
			Body:        []byte("agent body\n"),
		}},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 1 || got[0].Path != "agents/p-a.md" {
		t.Fatalf("artifacts = %#v", got)
	}
	fm, body := splitFrontmatter(t, got[0].Content)
	if fm["name"] != "p-a" || fm["description"] != "agent" {
		t.Errorf("frontmatter = %#v", fm)
	}
	if string(body) != "agent body\n" {
		t.Errorf("body = %q", body)
	}
}

func TestBuilder_Build_disabledEntry(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{{
			ID:   "p.skill.s",
			Kind: source.KindSkill,
			Name: "p-s",
			Tools: map[source.Target]source.ToolConfig{
				Target: {Enabled: boolPtr(false)},
			},
		}},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Build() returned %d artifacts, want 0", len(got))
	}
}
