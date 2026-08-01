package source

import "testing"

func TestKind_IsValid(t *testing.T) {
	tests := []struct {
		name string
		k    Kind
		want bool
	}{
		{name: "skill is valid", k: KindSkill, want: true},
		{name: "agent is valid", k: KindAgent, want: true},
		{name: "legacy rule is invalid", k: Kind("rule"), want: false},
		{name: "legacy prompt is invalid", k: Kind("prompt"), want: false},
		{name: "unknown value is invalid", k: Kind("unknown"), want: false},
		{name: "empty string is invalid", k: Kind(""), want: false},
		{name: "capitalized form is invalid", k: Kind("Skill"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.IsValid(); got != tt.want {
				t.Errorf("Kind.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    Kind
		want string
	}{
		{name: "skill", k: KindSkill, want: "skill"},
		{name: "agent", k: KindAgent, want: "agent"},
		{name: "preserves unknown literal", k: Kind("custom"), want: "custom"},
		{name: "empty string", k: Kind(""), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.String(); got != tt.want {
				t.Errorf("Kind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
