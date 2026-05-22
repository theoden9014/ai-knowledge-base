package claude

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestNewBuilder(t *testing.T) {
	got := NewBuilder()
	if got == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if got.Target() != Target {
		t.Errorf("Builder.Target() = %q, want %q", got.Target(), Target)
	}
}

func TestBuilder_Target(t *testing.T) {
	tests := []struct {
		name string
		b    *Builder
		want source.Target
	}{
		{
			name: "returns claude.Target",
			b:    NewBuilder(),
			want: Target,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.Target(); !cmp.Equal(tt.want, got) {
				t.Errorf("Builder.Target() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}

// boolPtr is a helper that returns &b.
func boolPtr(b bool) *bool { return &b }

func TestBuilder_Build(t *testing.T) {
	type args struct {
		ctx  context.Context
		pack *source.Pack
	}
	tests := []struct {
		name    string
		b       *Builder
		args    args
		want    []source.Artifact
		wantErr error
	}{
		{
			name: "empty pack returns no artifacts",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries:      nil,
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "skill entry produces SKILL.md",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:          "p.skill.orchestrator",
							Kind:        source.KindSkill,
							Name:        "p-orchestrator",
							Description: "desc",
							Path:        "skills/orchestrator.md",
							Body:        []byte("# Orchestrator\nbody\n"),
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target: Target,
					Path:   "skills/p-orchestrator/SKILL.md",
					// Frontmatter keys are alphabetical for deterministic ordering.
					Content: []byte("---\n" +
						"description: desc\n" +
						"name: p-orchestrator\n" +
						"---\n" +
						"# Orchestrator\nbody\n"),
					SourceEntryIDs: []string{"p.skill.orchestrator"},
				},
			},
			wantErr: nil,
		},
		{
			name: "skill entry merges tools.claude.frontmatter",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:          "p.skill.s",
							Kind:        source.KindSkill,
							Name:        "p-s",
							Description: "neutral-desc",
							Path:        "skills/s.md",
							Body:        []byte("body\n"),
							Tools: map[source.Target]source.ToolConfig{
								Target: {
									Frontmatter: map[string]any{
										"user-invocable": false,
										"description":    "override-desc",
									},
								},
							},
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target: Target,
					Path:   "skills/p-s/SKILL.md",
					// description is overridden by frontmatter, and user-invocable is merged in as an extra field.
					Content: []byte("---\n" +
						"description: override-desc\n" +
						"name: p-s\n" +
						"user-invocable: false\n" +
						"---\n" +
						"body\n"),
					SourceEntryIDs: []string{"p.skill.s"},
				},
			},
			wantErr: nil,
		},
		{
			name: "agent entry includes skills derived from uses_skills",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:          "p.agent.a",
							Kind:        source.KindAgent,
							Name:        "p-a",
							Description: "agent-desc",
							Path:        "agents/a.md",
							Body:        []byte("agent body\n"),
							Agent: &source.AgentMeta{
								UsesSkills: []string{"p.skill.s1", "p.skill.s2"},
							},
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target: Target,
					Path:   "agents/p-a.md",
					Content: []byte("---\n" +
						"description: agent-desc\n" +
						"name: p-a\n" +
						"skills:\n" +
						"- p-s1\n" +
						"- p-s2\n" +
						"---\n" +
						"agent body\n"),
					SourceEntryIDs: []string{"p.agent.a"},
				},
			},
			wantErr: nil,
		},
		{
			name: "prompt entry produces commands file without frontmatter",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:          "p.prompt.r",
							Kind:        source.KindPrompt,
							Name:        "p-review",
							Description: "review prompt",
							Path:        "prompts/review.md",
							Body:        []byte("Please review.\n"),
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target:         Target,
					Path:           "commands/p-review.md",
					Content:        []byte("Please review.\n"),
					SourceEntryIDs: []string{"p.prompt.r"},
				},
			},
			wantErr: nil,
		},
		{
			name: "rule entries are concatenated in manifest order with H1/H2 headings, body newline padding and one blank line between entries",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:          "p.rule.r1",
							Kind:        source.KindRule,
							Name:        "p-r1",
							Description: "rule1",
							Path:        "rules/r1.md",
							// No trailing newline; Builder should append it.
							Body: []byte("rule1 body"),
						},
						{
							ID:          "p.rule.r2",
							Kind:        source.KindRule,
							Name:        "p-r2",
							Description: "rule2",
							Path:        "rules/r2.md",
							// Already has a trailing newline.
							Body: []byte("rule2 body\n"),
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target: Target,
					Path:   "CLAUDE.md",
					Content: []byte(
						"# p\n" +
							"\n" +
							"## p-r1\n" +
							"\n" +
							"rule1 body\n" +
							"\n" +
							"## p-r2\n" +
							"\n" +
							"rule2 body\n"),
					SourceEntryIDs: []string{"p.rule.r1", "p.rule.r2"},
				},
			},
			wantErr: nil,
		},
		{
			name: "rule entry with tools.claude.frontmatter returns ErrFrontmatterMergeConflict",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:   "p.rule.r1",
							Kind: source.KindRule,
							Name: "p-r1",
							Path: "rules/r1.md",
							Body: []byte("body\n"),
							Tools: map[source.Target]source.ToolConfig{
								Target: {Frontmatter: map[string]any{"k": "v"}},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: ErrFrontmatterMergeConflict,
		},
		{
			name: "prompt entry with tools.claude.frontmatter returns ErrFrontmatterMergeConflict",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:   "p.prompt.p1",
							Kind: source.KindPrompt,
							Name: "p-p1",
							Path: "prompts/p1.md",
							Body: []byte("body\n"),
							Tools: map[source.Target]source.ToolConfig{
								Target: {Frontmatter: map[string]any{"k": "v"}},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: ErrFrontmatterMergeConflict,
		},
		{
			name: "entry disabled for claude is skipped",
			b:    NewBuilder(),
			args: args{
				ctx: context.Background(),
				pack: &source.Pack{
					Name:         "p",
					Version:      "0.1.0",
					Description:  "x",
					DefaultTools: []source.Target{Target},
					Entries: []source.Entry{
						{
							ID:   "p.skill.kept",
							Kind: source.KindSkill,
							Name: "p-kept",
							Path: "skills/kept.md",
							Body: []byte("kept\n"),
						},
						{
							ID:   "p.skill.dropped",
							Kind: source.KindSkill,
							Name: "p-dropped",
							Path: "skills/dropped.md",
							Body: []byte("dropped\n"),
							Tools: map[source.Target]source.ToolConfig{
								Target: {Enabled: boolPtr(false)},
							},
						},
					},
				},
			},
			want: []source.Artifact{
				{
					Target: Target,
					Path:   "skills/p-kept/SKILL.md",
					// When description is an empty string, sigs.k8s.io/yaml emits
					// `description: ""` with double quotes.
					// This format depends on sigs.k8s.io/yaml behavior, so the
					// expected value must be updated if a future upgrade changes quoting rules.
					Content: []byte("---\n" +
						"description: \"\"\n" +
						"name: p-kept\n" +
						"---\n" +
						"kept\n"),
					SourceEntryIDs: []string{"p.skill.kept"},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.b
			got, err := b.Build(tt.args.ctx, tt.args.pack)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Builder.Build() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Builder.Build() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
				if len(tt.want) == len(got) {
					for i := range tt.want {
						if !bytes.Equal(tt.want[i].Content, got[i].Content) {
							t.Errorf("artifact[%d].Content mismatch:\nwant:\n%s\n\ngot:\n%s",
								i, tt.want[i].Content, got[i].Content)
						}
					}
				}
			}
		})
	}
}

// TestBuilder_Build_isIdempotent verifies that calling Build twice with the
// same Pack input returns the same artifact sequence.
func TestBuilder_Build_isIdempotent(t *testing.T) {
	pack := &source.Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "x",
		DefaultTools: []source.Target{Target},
		Entries: []source.Entry{
			{
				ID:          "p.skill.s",
				Kind:        source.KindSkill,
				Name:        "p-s",
				Description: "d",
				Path:        "skills/s.md",
				Body:        []byte("body\n"),
				Tools: map[source.Target]source.ToolConfig{
					Target: {Frontmatter: map[string]any{"zeta": 1, "alpha": "a"}},
				},
			},
		},
	}
	b := NewBuilder()
	first, err := b.Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	second, err := b.Build(context.Background(), pack)
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}
	if diff := cmp.Diff(first, second); diff != "" {
		t.Errorf("Build is not idempotent (-first +second):\n%s", diff)
	}
	// Confirm that frontmatter key ordering is fixed alphabetically.
	if !bytes.Contains(first[0].Content, []byte("alpha: a\n")) {
		t.Errorf("expected alpha key in frontmatter; got:\n%s", first[0].Content)
	}
	if strings.Index(string(first[0].Content), "alpha:") >
		strings.Index(string(first[0].Content), "zeta:") {
		t.Errorf("frontmatter keys are not sorted; got:\n%s", first[0].Content)
	}
}
