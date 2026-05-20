package gemini

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pelletier/go-toml/v2"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"sigs.k8s.io/yaml"
)

func boolPtr(b bool) *bool { return &b }

// splitFrontmatter extracts frontmatter (YAML) and body (raw bytes) from
// Markdown in the form "---\n<yaml>\n---\n<body>". It is a test-only helper.
// If there is no frontmatter, ok=false.
func splitFrontmatter(t *testing.T, b []byte) (fm map[string]any, body []byte, ok bool) {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return nil, b, false
	}
	rest := b[len("---\n"):]
	idx := bytes.Index(rest, []byte("\n---\n"))
	if idx < 0 {
		return nil, b, false
	}
	yamlPart := rest[:idx]
	body = rest[idx+len("\n---\n"):]
	fm = map[string]any{}
	if err := yaml.Unmarshal(yamlPart, &fm); err != nil {
		t.Fatalf("splitFrontmatter: yaml unmarshal failed: %v\nfrontmatter=%s", err, yamlPart)
	}
	return fm, body, true
}

func TestBuilder_Target(t *testing.T) {
	b := NewBuilder()
	if got, want := b.Target(), Target; got != want {
		t.Errorf("Builder.Target() = %q, want %q", got, want)
	}
}

func TestBuilder_Build_emptyPack(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: []source.Target{Target},
		Entries:      nil,
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("Build() returned %d artifacts on empty pack, want 0", len(got))
	}
}

func TestBuilder_Build_disabledEntriesExcluded(t *testing.T) {
	// DefaultTools does not include gemini and per-entry enabled is also false.
	// Build should therefore be empty because the entry is filtered out by
	// pack.EntriesFor.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: nil,
		Entries: []source.Entry{
			{
				ID:   "p.skill.foo",
				Kind: source.KindSkill,
				Name: "p-foo",
				Path: "skills/foo.md",
				Body: []byte("hello"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {Enabled: boolPtr(false)},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("Build() returned %d artifacts for disabled entry, want 0", len(got))
	}
}

func TestBuilder_Build_skill(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.skill.orchestrator",
				Kind:        source.KindSkill,
				Name:        "p-orchestrator",
				Description: "Coordinates work.",
				Tags:        []string{"core"},
				Path:        "skills/orchestrator.md",
				Body:        []byte("# Orchestrator\n\nbody.\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"extra": "from-tools",
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if a.Target != Target {
		t.Errorf("Artifact.Target = %q, want %q", a.Target, Target)
	}
	if want := "skills/p-orchestrator/SKILL.md"; a.Path != want {
		t.Errorf("Artifact.Path = %q, want %q", a.Path, want)
	}
	wantSourceEntryIDs := []string{"p.skill.orchestrator"}
	if diff := cmp.Diff(wantSourceEntryIDs, a.SourceEntryIDs); diff != "" {
		t.Errorf("Artifact.SourceEntryIDs mismatch (-want +got):\n%s", diff)
	}
	fm, body, ok := splitFrontmatter(t, a.Content)
	if !ok {
		t.Fatalf("Artifact.Content has no frontmatter:\n%s", a.Content)
	}
	wantFM := map[string]any{
		"name":        "p-orchestrator",
		"description": "Coordinates work.",
		"extra":       "from-tools",
	}
	if diff := cmp.Diff(wantFM, fm); diff != "" {
		t.Errorf("frontmatter mismatch (-want +got):\n%s", diff)
	}
	if string(body) != "# Orchestrator\n\nbody.\n" {
		t.Errorf("body = %q, want %q", body, "# Orchestrator\n\nbody.\n")
	}
}

func TestBuilder_Build_skill_frontmatter_override(t *testing.T) {
	// Verify that Tools[gemini].Frontmatter can override name / description.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.skill.s",
				Kind:        source.KindSkill,
				Name:        "neutral-name",
				Description: "neutral",
				Path:        "skills/s.md",
				Body:        []byte("x"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"name":        "override-name",
							"description": "override-desc",
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	fm, _, ok := splitFrontmatter(t, got[0].Content)
	if !ok {
		t.Fatalf("no frontmatter:\n%s", got[0].Content)
	}
	if fm["name"] != "override-name" {
		t.Errorf("name = %v, want override-name", fm["name"])
	}
	if fm["description"] != "override-desc" {
		t.Errorf("description = %v, want override-desc", fm["description"])
	}
}

func TestBuilder_Build_agent(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.agent.solid-reviewer",
				Kind:        source.KindAgent,
				Name:        "p-solid-reviewer",
				Description: "Reviews for SOLID.",
				Path:        "agents/solid-reviewer.md",
				Body:        []byte("System prompt body.\n"),
				Agent: &source.AgentMeta{
					UsesSkills: []string{"p.skill.orchestrator"},
				},
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"model":       "gemini-3-flash",
							"temperature": 0.2,
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if want := "agents/p-solid-reviewer.md"; a.Path != want {
		t.Errorf("Artifact.Path = %q, want %q", a.Path, want)
	}
	fm, body, ok := splitFrontmatter(t, a.Content)
	if !ok {
		t.Fatalf("no frontmatter:\n%s", a.Content)
	}
	// uses_skills is not reflected in the generated output.
	if _, present := fm["uses_skills"]; present {
		t.Errorf("uses_skills should not be present in agent frontmatter, got %v", fm["uses_skills"])
	}
	if fm["name"] != "p-solid-reviewer" {
		t.Errorf("name = %v, want p-solid-reviewer", fm["name"])
	}
	if fm["description"] != "Reviews for SOLID." {
		t.Errorf("description = %v, want Reviews for SOLID.", fm["description"])
	}
	if fm["model"] != "gemini-3-flash" {
		t.Errorf("model = %v, want gemini-3-flash", fm["model"])
	}
	// 0.2 comes back as float64 through YAML.
	if v, _ := fm["temperature"].(float64); v != 0.2 {
		t.Errorf("temperature = %v, want 0.2", fm["temperature"])
	}
	if string(body) != "System prompt body.\n" {
		t.Errorf("body = %q, want %q", body, "System prompt body.\n")
	}
}

func TestBuilder_Build_rule_concatenated(t *testing.T) {
	pack := &source.Pack{
		Name:         "mypack",
		Version:      "0.1.0",
		Description:  "d",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:   "mypack.rule.a",
				Kind: source.KindRule,
				Name: "rule-a",
				Path: "rules/a.md",
				Body: []byte("rule A body\n"),
			},
			{
				ID:   "mypack.rule.b",
				Kind: source.KindRule,
				Name: "rule-b",
				Path: "rules/b.md",
				Body: []byte("rule B body\n"),
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if a.Path != "GEMINI.md" {
		t.Errorf("Artifact.Path = %q, want GEMINI.md", a.Path)
	}
	wantIDs := []string{"mypack.rule.a", "mypack.rule.b"}
	if diff := cmp.Diff(wantIDs, a.SourceEntryIDs); diff != "" {
		t.Errorf("SourceEntryIDs mismatch (-want +got):\n%s", diff)
	}
	c := string(a.Content)
	// No frontmatter.
	if strings.HasPrefix(c, "---") {
		t.Errorf("GEMINI.md should not start with frontmatter, got:\n%s", c)
	}
	// H1 = pack name.
	if !strings.Contains(c, "# mypack\n") {
		t.Errorf("missing H1 pack name; content=\n%s", c)
	}
	// Each entry's H2 and body appear in manifest order.
	idxA := strings.Index(c, "## rule-a")
	idxB := strings.Index(c, "## rule-b")
	if idxA < 0 || idxB < 0 {
		t.Fatalf("missing H2 entries; content=\n%s", c)
	}
	if idxA >= idxB {
		t.Errorf("entry order wrong; rule-a should precede rule-b. content=\n%s", c)
	}
	if !strings.Contains(c, "rule A body") || !strings.Contains(c, "rule B body") {
		t.Errorf("missing entry body; content=\n%s", c)
	}
}

func TestBuilder_Build_rule_frontmatter_conflict(t *testing.T) {
	// If Tools[gemini].Frontmatter is non-empty for a rule kind,
	// ErrFrontmatterMergeConflict is returned.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:   "p.rule.x",
				Kind: source.KindRule,
				Name: "rx",
				Path: "rules/x.md",
				Body: []byte("body\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{"foo": "bar"},
					},
				},
			},
		},
	}
	_, err := NewBuilder().Build(context.Background(), pack)
	if !errors.Is(err, ErrFrontmatterMergeConflict) {
		t.Errorf("Build() err = %v, want errors.Is(err, ErrFrontmatterMergeConflict)", err)
	}
}

func TestBuilder_Build_prompt_toml(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.prompt.review",
				Kind:        source.KindPrompt,
				Name:        "p-review",
				Description: "Review the diff.",
				Path:        "prompts/review.md",
				Body:        []byte("Please review the following diff.\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"flag":  true,
							"count": int64(3),
							"items": []any{"a", "b"},
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d artifacts, want 1", len(got))
	}
	a := got[0]
	if want := "commands/p-review.toml"; a.Path != want {
		t.Errorf("Artifact.Path = %q, want %q", a.Path, want)
	}
	// Parse and inspect values directly to avoid depending on implementation-
	// specific TOML encoder key ordering.
	parsed := map[string]any{}
	if err := toml.Unmarshal(a.Content, &parsed); err != nil {
		t.Fatalf("toml.Unmarshal failed: %v\ncontent=\n%s", err, a.Content)
	}
	if parsed["prompt"] != "Please review the following diff.\n" {
		t.Errorf("prompt = %q, want body content", parsed["prompt"])
	}
	if parsed["description"] != "Review the diff." {
		t.Errorf("description = %q, want Review the diff.", parsed["description"])
	}
	if parsed["flag"] != true {
		t.Errorf("flag = %v, want true", parsed["flag"])
	}
	// TOML integers come back as int64.
	if v, _ := parsed["count"].(int64); v != 3 {
		t.Errorf("count = %v (type %T), want int64(3)", parsed["count"], parsed["count"])
	}
	items, ok := parsed["items"].([]any)
	if !ok || len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Errorf("items = %v, want [a b]", parsed["items"])
	}
}

func TestBuilder_Build_prompt_toml_frontmatter_overrides(t *testing.T) {
	// Tools[gemini].Frontmatter overrides TOML top-level keys.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.prompt.r",
				Kind:        source.KindPrompt,
				Name:        "p-r",
				Description: "neutral-desc",
				Path:        "prompts/r.md",
				Body:        []byte("neutral body"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"description": "override-desc",
							"prompt":      "override-body",
						},
					},
				},
			},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	parsed := map[string]any{}
	if err := toml.Unmarshal(got[0].Content, &parsed); err != nil {
		t.Fatalf("toml.Unmarshal failed: %v", err)
	}
	if parsed["description"] != "override-desc" {
		t.Errorf("description = %v, want override-desc", parsed["description"])
	}
	if parsed["prompt"] != "override-body" {
		t.Errorf("prompt = %v, want override-body", parsed["prompt"])
	}
}

func TestBuilder_Build_prompt_toml_unsupported_value(t *testing.T) {
	// Passing a value that cannot be TOML-encoded, such as a function value,
	// returns ErrUnsupportedFrontmatterValue.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:   "p.prompt.x",
				Kind: source.KindPrompt,
				Name: "p-x",
				Path: "prompts/x.md",
				Body: []byte("body"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {
						Frontmatter: map[string]any{
							"bad": func() {},
						},
					},
				},
			},
		},
	}
	_, err := NewBuilder().Build(context.Background(), pack)
	if !errors.Is(err, ErrUnsupportedFrontmatterValue) {
		t.Errorf("Build() err = %v, want errors.Is(err, ErrUnsupportedFrontmatterValue)", err)
	}
}

func TestBuilder_Build_mixed_kinds(t *testing.T) {
	// When Entries of different kinds are mixed, each should be emitted to the
	// correct Path.
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{ID: "p.skill.s", Kind: source.KindSkill, Name: "p-s", Path: "skills/s.md", Body: []byte("sb")},
			{ID: "p.agent.a", Kind: source.KindAgent, Name: "p-a", Path: "agents/a.md", Body: []byte("ab"),
				Agent: &source.AgentMeta{UsesSkills: nil}},
			{ID: "p.rule.r", Kind: source.KindRule, Name: "r", Path: "rules/r.md", Body: []byte("rb\n")},
			{ID: "p.prompt.q", Kind: source.KindPrompt, Name: "p-q", Path: "prompts/q.md", Body: []byte("qb")},
		},
	}
	got, err := NewBuilder().Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	paths := make([]string, 0, len(got))
	for _, a := range got {
		if a.Target != Target {
			t.Errorf("artifact %q has Target=%q, want %q", a.Path, a.Target, Target)
		}
		paths = append(paths, a.Path)
	}
	sort.Strings(paths)
	want := []string{
		"GEMINI.md",
		"agents/p-a.md",
		"commands/p-q.toml",
		"skills/p-s/SKILL.md",
	}
	if diff := cmp.Diff(want, paths); diff != "" {
		t.Errorf("Artifact.Path set mismatch (-want +got):\n%s", diff)
	}
}
