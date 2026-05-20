package inventory

import (
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestLabel_IsZero(t *testing.T) {
	tests := []struct {
		name string
		l    Label
		want bool
	}{
		{
			name: "both fields zero -> IsZero true",
			l:    Label{},
			want: true,
		},
		{
			name: "only Target set -> IsZero false",
			l:    Label{Target: source.Target("claude")},
			want: false,
		},
		{
			name: "only Scope set -> IsZero false",
			l:    Label{Scope: ScopeUser},
			want: false,
		},
		{
			name: "both fields set -> IsZero false",
			l:    Label{Target: source.Target("claude"), Scope: ScopeProject},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.IsZero(); got != tt.want {
				t.Errorf("Label.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
