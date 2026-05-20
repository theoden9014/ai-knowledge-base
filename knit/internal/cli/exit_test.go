package cli

import "testing"

func TestExitCode_Int(t *testing.T) {
	tests := []struct {
		name string
		c    ExitCode
		want int
	}{
		{name: "success", c: ExitSuccess, want: 0},
		{name: "general", c: ExitGeneral, want: 1},
		{name: "usage", c: ExitUsage, want: 2},
		{name: "config", c: ExitConfig, want: 3},
		{name: "notfound", c: ExitNotFound, want: 4},
		{name: "conflict", c: ExitConflict, want: 5},
		{name: "partial", c: ExitPartial, want: 6},
		{name: "raw value preserved", c: ExitCode(42), want: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Int(); got != tt.want {
				t.Errorf("ExitCode.Int() = %v, want %v", got, tt.want)
			}
		})
	}
}
