package source

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func boolPtr(b bool) *bool { return &b }

func TestPack_EntryByID(t *testing.T) {
	type fields struct {
		Name         string
		Version      string
		Description  string
		DefaultTools []Target
		Entries      []Entry
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Entry
	}{
		{
			name: "returns entry when id matches",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
					{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
				},
			},
			args: args{id: "p.skill.b"},
			want: &Entry{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
		},
		{
			name: "returns nil when id is absent",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{id: "p.skill.missing"},
			want: nil,
		},
		{
			name: "returns nil for empty entries",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: nil,
			},
			args: args{id: "p.skill.a"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pack{
				Name:         tt.fields.Name,
				Version:      tt.fields.Version,
				Description:  tt.fields.Description,
				DefaultTools: tt.fields.DefaultTools,
				Entries:      tt.fields.Entries,
			}
			if got := p.EntryByID(tt.args.id); !cmp.Equal(tt.want, got) {
				t.Errorf("Pack.EntryByID() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestPack_EntriesByKind(t *testing.T) {
	type fields struct {
		Name         string
		Version      string
		Description  string
		DefaultTools []Target
		Entries      []Entry
	}
	type args struct {
		kind Kind
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*Entry
	}{
		{
			name: "returns matching skills in manifest order",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
					{ID: "p.agent.x", Kind: KindAgent, Name: "p-x", Description: "x", Agent: &AgentMeta{UsesSkills: []string{"p.skill.a"}}},
					{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
				},
			},
			args: args{kind: KindSkill},
			want: []*Entry{
				{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
			},
		},
		{
			name: "returns empty slice when no entries match",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{kind: KindRule},
			want: nil,
		},
		{
			name: "returns rules only",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				Entries: []Entry{
					{ID: "p.rule.r1", Kind: KindRule, Name: "p-r1", Description: "r1"},
					{ID: "p.prompt.q", Kind: KindPrompt, Name: "p-q", Description: "q"},
					{ID: "p.rule.r2", Kind: KindRule, Name: "p-r2", Description: "r2"},
				},
			},
			args: args{kind: KindRule},
			want: []*Entry{
				{ID: "p.rule.r1", Kind: KindRule, Name: "p-r1", Description: "r1"},
				{ID: "p.rule.r2", Kind: KindRule, Name: "p-r2", Description: "r2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pack{
				Name:         tt.fields.Name,
				Version:      tt.fields.Version,
				Description:  tt.fields.Description,
				DefaultTools: tt.fields.DefaultTools,
				Entries:      tt.fields.Entries,
			}
			if got := p.EntriesByKind(tt.args.kind); !cmp.Equal(tt.want, got) {
				t.Errorf("Pack.EntriesByKind() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestPack_EntriesFor(t *testing.T) {
	type fields struct {
		Name         string
		Version      string
		Description  string
		DefaultTools []Target
		Entries      []Entry
	}
	type args struct {
		target Target
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*Entry
	}{
		{
			name: "default_tools enables entries when per-target Enabled is nil",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
					{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
				},
			},
			args: args{target: Target("claude")},
			want: []*Entry{
				{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
			},
		},
		{
			name: "per-target Enabled false overrides default_tools",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
					{
						ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b",
						Tools: map[Target]ToolConfig{
							Target("claude"): {Enabled: boolPtr(false)},
						},
					},
				},
			},
			args: args{target: Target("claude")},
			want: []*Entry{
				{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
			},
		},
		{
			name: "per-target Enabled true overrides absence from default_tools",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{
						ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a",
						Tools: map[Target]ToolConfig{
							Target("codex"): {Enabled: boolPtr(true)},
						},
					},
					{ID: "p.skill.b", Kind: KindSkill, Name: "p-b", Description: "b"},
				},
			},
			args: args{target: Target("codex")},
			want: []*Entry{
				{
					ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a",
					Tools: map[Target]ToolConfig{
						Target("codex"): {Enabled: boolPtr(true)},
					},
				},
			},
		},
		{
			name: "returns nil when nothing is enabled",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: nil,
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{target: Target("claude")},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pack{
				Name:         tt.fields.Name,
				Version:      tt.fields.Version,
				Description:  tt.fields.Description,
				DefaultTools: tt.fields.DefaultTools,
				Entries:      tt.fields.Entries,
			}
			if got := p.EntriesFor(tt.args.target); !cmp.Equal(tt.want, got) {
				t.Errorf("Pack.EntriesFor() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestPack_IsEntryEnabledFor(t *testing.T) {
	type fields struct {
		Name         string
		Version      string
		Description  string
		DefaultTools []Target
		Entries      []Entry
	}
	type args struct {
		entryID string
		target  Target
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "default_tools fallback yields true",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{entryID: "p.skill.a", target: Target("claude")},
			want: true,
		},
		{
			name: "per-target Enabled false yields false",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{
						ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a",
						Tools: map[Target]ToolConfig{
							Target("claude"): {Enabled: boolPtr(false)},
						},
					},
				},
			},
			args: args{entryID: "p.skill.a", target: Target("claude")},
			want: false,
		},
		{
			name: "per-target Enabled true yields true even without default_tools",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: nil,
				Entries: []Entry{
					{
						ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a",
						Tools: map[Target]ToolConfig{
							Target("codex"): {Enabled: boolPtr(true)},
						},
					},
				},
			},
			args: args{entryID: "p.skill.a", target: Target("codex")},
			want: true,
		},
		{
			name: "missing id returns false",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{entryID: "p.skill.missing", target: Target("claude")},
			want: false,
		},
		{
			name: "not in default_tools and no per-target config returns false",
			fields: fields{
				Name: "p", Version: "0.1.0", Description: "d",
				DefaultTools: []Target{Target("claude")},
				Entries: []Entry{
					{ID: "p.skill.a", Kind: KindSkill, Name: "p-a", Description: "a"},
				},
			},
			args: args{entryID: "p.skill.a", target: Target("codex")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pack{
				Name:         tt.fields.Name,
				Version:      tt.fields.Version,
				Description:  tt.fields.Description,
				DefaultTools: tt.fields.DefaultTools,
				Entries:      tt.fields.Entries,
			}
			if got := p.IsEntryEnabledFor(tt.args.entryID, tt.args.target); got != tt.want {
				t.Errorf("Pack.IsEntryEnabledFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
