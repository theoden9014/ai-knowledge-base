package source

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func boolPtr(b bool) *bool { return &b }

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
