package source

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewEntryID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    EntryID
		wantErr error
	}{
		{name: "empty rejected", args: args{s: ""}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "single component rejected", args: args{s: "pack"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "two components rejected", args: args{s: "pack.skill"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "leading dot rejected", args: args{s: ".skill.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "trailing dot rejected", args: args{s: "pack.skill."}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "double dot rejected", args: args{s: "pack..name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "invalid kind rejected", args: args{s: "pack.foo.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "uppercase in pack rejected", args: args{s: "Pack.skill.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "uppercase in name rejected", args: args{s: "pack.skill.Name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "underscore in pack rejected", args: args{s: "my_pack.skill.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "leading hyphen in pack rejected", args: args{s: "-pack.skill.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "trailing hyphen in pack rejected", args: args{s: "pack-.skill.name"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "double hyphen in name rejected", args: args{s: "pack.skill.foo--bar"}, want: EntryID{}, wantErr: ErrInvalidEntryID},
		{name: "skill accepted", args: args{s: "structure-behavior-design.skill.orchestrator"}, want: EntryID{pack: "structure-behavior-design", kind: KindSkill, name: "orchestrator"}, wantErr: nil},
		{name: "agent accepted", args: args{s: "pack.agent.reviewer"}, want: EntryID{pack: "pack", kind: KindAgent, name: "reviewer"}, wantErr: nil},
		{name: "rule accepted", args: args{s: "pack.rule.guidelines"}, want: EntryID{pack: "pack", kind: KindRule, name: "guidelines"}, wantErr: nil},
		{name: "prompt accepted", args: args{s: "pack.prompt.greeting"}, want: EntryID{pack: "pack", kind: KindPrompt, name: "greeting"}, wantErr: nil},
		{name: "digits allowed", args: args{s: "pack1.skill.entry2"}, want: EntryID{pack: "pack1", kind: KindSkill, name: "entry2"}, wantErr: nil},
		{name: "multi-segment kebab accepted", args: args{s: "a-b-c.skill.x-y-z"}, want: EntryID{pack: "a-b-c", kind: KindSkill, name: "x-y-z"}, wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEntryID(tt.args.s)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewEntryID() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(EntryID{})); diff != "" {
				t.Errorf("NewEntryID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEntryID_Pack(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value returns empty", fields: fields{}, want: ""},
		{name: "returns pack component", fields: fields{pack: "structure-behavior-design", kind: KindSkill, name: "orchestrator"}, want: "structure-behavior-design"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.Pack(); got != tt.want {
				t.Errorf("EntryID.Pack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryID_Kind(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   Kind
	}{
		{name: "zero value returns zero Kind", fields: fields{}, want: Kind("")},
		{name: "returns skill", fields: fields{pack: "p", kind: KindSkill, name: "n"}, want: KindSkill},
		{name: "returns agent", fields: fields{pack: "p", kind: KindAgent, name: "n"}, want: KindAgent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.Kind(); got != tt.want {
				t.Errorf("EntryID.Kind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryID_Name(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value returns empty", fields: fields{}, want: ""},
		{name: "returns name component", fields: fields{pack: "p", kind: KindSkill, name: "orchestrator"}, want: "orchestrator"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.Name(); got != tt.want {
				t.Errorf("EntryID.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryID_String(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero value returns empty", fields: fields{}, want: ""},
		{name: "skill formatted", fields: fields{pack: "structure-behavior-design", kind: KindSkill, name: "orchestrator"}, want: "structure-behavior-design.skill.orchestrator"},
		{name: "agent formatted", fields: fields{pack: "pack", kind: KindAgent, name: "reviewer"}, want: "pack.agent.reviewer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.String(); got != tt.want {
				t.Errorf("EntryID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryID_IsZero(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "zero value is zero", fields: fields{}, want: true},
		{name: "any field set is not zero", fields: fields{pack: "p", kind: KindSkill, name: "n"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.IsZero(); got != tt.want {
				t.Errorf("EntryID.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryID_Equal(t *testing.T) {
	type fields struct {
		pack string
		kind Kind
		name string
	}
	type args struct {
		other EntryID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "equal", fields: fields{pack: "p", kind: KindSkill, name: "n"}, args: args{other: EntryID{pack: "p", kind: KindSkill, name: "n"}}, want: true},
		{name: "pack differs", fields: fields{pack: "p", kind: KindSkill, name: "n"}, args: args{other: EntryID{pack: "q", kind: KindSkill, name: "n"}}, want: false},
		{name: "kind differs", fields: fields{pack: "p", kind: KindSkill, name: "n"}, args: args{other: EntryID{pack: "p", kind: KindAgent, name: "n"}}, want: false},
		{name: "name differs", fields: fields{pack: "p", kind: KindSkill, name: "n"}, args: args{other: EntryID{pack: "p", kind: KindSkill, name: "m"}}, want: false},
		{name: "both zero are equal", fields: fields{}, args: args{other: EntryID{}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := EntryID{
				pack: tt.fields.pack,
				kind: tt.fields.kind,
				name: tt.fields.name,
			}
			if got := id.Equal(tt.args.other); got != tt.want {
				t.Errorf("EntryID.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
