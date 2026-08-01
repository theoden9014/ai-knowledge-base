package codex

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// boolPtr is a small helper for setting *bool fields in test cases.
func boolPtr(b bool) *bool { return &b }

func TestBuilder_Target(t *testing.T) {
	b := NewBuilder()
	if got, want := b.Target(), Target; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestBuilder_Build_KindSkill(t *testing.T) {
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		Description:  "demo pack",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "demo.skill.orchestrator",
				Kind:        source.KindSkill,
				Name:        "demo-orchestrator",
				Description: "orchestrator skill",
				Path:        "skills/orchestrator",
				Body:        []byte("# body\n"),
			},
		},
	}

	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if a.Target != Target {
		t.Errorf("Artifact.Target = %q, want %q", a.Target, Target)
	}
	if a.Path != "skills/demo-orchestrator/SKILL.md" {
		t.Errorf("Artifact.Path = %q, want skills/demo-orchestrator/SKILL.md", a.Path)
	}
	content := string(a.Content)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("SKILL.md should start with YAML frontmatter delimiter, got:\n%s", content)
	}
	if !strings.Contains(content, "name: demo-orchestrator") {
		t.Errorf("SKILL.md frontmatter missing 'name: demo-orchestrator'\n%s", content)
	}
	if !strings.Contains(content, "description: orchestrator skill") {
		t.Errorf("SKILL.md frontmatter missing description\n%s", content)
	}
	if !strings.Contains(content, "# body") {
		t.Errorf("SKILL.md body missing\n%s", content)
	}
	wantSourceEntryIDs := []string{"demo.skill.orchestrator"}
	if diff := cmp.Diff(wantSourceEntryIDs, a.SourceEntryIDs); diff != "" {
		t.Errorf("SourceEntryIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestBuilder_Build_KindSkill_FrontmatterMerge(t *testing.T) {
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "demo.skill.s",
				Kind:        source.KindSkill,
				Name:        "demo-s",
				Description: "original",
				Path:        "skills/s",
				Body:        []byte("body\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"description": "overridden",
							"extra":       "value",
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	content := string(got[0].Content)
	if !strings.Contains(content, "description: overridden") {
		t.Errorf("frontmatter merge: description should be 'overridden', got:\n%s", content)
	}
	if !strings.Contains(content, "extra: value") {
		t.Errorf("frontmatter merge: extra key missing\n%s", content)
	}
}

func TestBuilder_Build_KindAgent_TOML(t *testing.T) {
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "demo.agent.reviewer",
				Kind:        source.KindAgent,
				Name:        "demo-reviewer",
				Description: "reviewer agent",
				Path:        "agents/reviewer.md",
				Body:        []byte("Reviewer instructions.\n"),
				Agent:       &source.AgentMeta{UsesSkills: []string{"demo.skill.s"}},
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"model": "gpt-5",
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if a.Path != "agents/demo-reviewer.toml" {
		t.Errorf("agent Artifact.Path = %q, want agents/demo-reviewer.toml", a.Path)
	}
	content := string(a.Content)
	if !tomlAssign(content, "name", "demo-reviewer") {
		t.Errorf("agent TOML missing name field\n%s", content)
	}
	if !tomlAssign(content, "description", "reviewer agent") {
		t.Errorf("agent TOML missing description field\n%s", content)
	}
	if !strings.Contains(content, "developer_instructions") {
		t.Errorf("agent TOML missing developer_instructions\n%s", content)
	}
	if !strings.Contains(content, "Reviewer instructions.") {
		t.Errorf("agent TOML missing Body content\n%s", content)
	}
	if !tomlAssign(content, "model", "gpt-5") {
		t.Errorf("agent TOML missing merged 'model' field\n%s", content)
	}
	// uses_skills should be ignored in the current phase.
	if strings.Contains(content, "uses_skills") || strings.Contains(content, "skills.config") {
		t.Errorf("agent TOML should not include uses_skills mapping in current phase\n%s", content)
	}
}

func TestBuilder_Build_KindAgent_TOMLEscapesTripleQuoteCollision(t *testing.T) {
	// The Body contains """. Build should not return an error; it should handle
	// the collision through internal escaping.
	// Signature contract: do not add a new sentinel error.
	bodyWithCollision := []byte("contains \"\"\" inside body\n")
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "demo.agent.a",
				Kind:        source.KindAgent,
				Name:        "demo-a",
				Description: "d",
				Path:        "agents/a.md",
				Body:        bodyWithCollision,
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() with triple-quote-collision body err = %v, want nil (escape should be internal)", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	// Sanity check: the emitted TOML should remain valid enough to contain at
	// least the developer_instructions line.
	if !strings.Contains(string(got[0].Content), "developer_instructions") {
		t.Errorf("agent TOML missing developer_instructions key on collision body")
	}
}

func TestBuilder_Build_RespectsToolsEnabled(t *testing.T) {
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "demo.skill.included",
				Kind:        source.KindSkill,
				Name:        "demo-included",
				Description: "yes",
				Path:        "skills/included",
				Body:        []byte("body\n"),
			},
			{
				ID:          "demo.skill.excluded",
				Kind:        source.KindSkill,
				Name:        "demo-excluded",
				Description: "no",
				Path:        "skills/excluded",
				Body:        []byte("body\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {Enabled: boolPtr(false)},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	for _, a := range got {
		if strings.Contains(a.Path, "excluded") {
			t.Errorf("disabled entry was built: %q", a.Path)
		}
	}
	wantPaths := []string{"skills/demo-included/SKILL.md"}
	var gotPaths []string
	for _, a := range got {
		gotPaths = append(gotPaths, a.Path)
	}
	if diff := cmp.Diff(wantPaths, gotPaths); diff != "" {
		t.Errorf("paths mismatch (-want +got):\n%s", diff)
	}
}

func TestBuilder_Build_DefaultToolsExclusion(t *testing.T) {
	// If codex is not included in DefaultTools, Tools[codex].Enabled is also
	// absent, so nothing should be built.
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{}, // codex excluded
		Entries: []source.Entry{
			{
				ID:          "demo.skill.a",
				Kind:        source.KindSkill,
				Name:        "demo-a",
				Description: "x",
				Path:        "skills/a",
				Body:        []byte("body\n"),
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Build() returned %d artifacts, want 0 (codex not in DefaultTools)", len(got))
	}
}

func TestBuilder_Build_EmitsCodexTargetForAllArtifacts(t *testing.T) {
	pack := &source.Pack{
		Name:         "demo",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:   "demo.skill.a",
				Kind: source.KindSkill,
				Name: "demo-a",
				Path: "skills/a",
				Body: []byte("body\n"),
			},
			{
				ID:   "demo.agent.b",
				Kind: source.KindAgent,
				Name: "demo-b",
				Path: "agents/b.md",
				Body: []byte("body\n"),
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Build() returned %d artifacts, want 2", len(got))
	}
	for i, a := range got {
		if a.Target != Target {
			t.Errorf("Build()[%d].Target = %q, want %q", i, a.Target, Target)
		}
	}
}
