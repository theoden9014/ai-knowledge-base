package inventory

import (
	"errors"
	"testing"
)

func TestScope_Valid(t *testing.T) {
	tests := []struct {
		name string
		s    Scope
		want bool
	}{
		{
			name: "ScopeUser is valid",
			s:    ScopeUser,
			want: true,
		},
		{
			name: "ScopeProject is valid",
			s:    ScopeProject,
			want: true,
		},
		{
			name: "empty string is invalid",
			s:    Scope(""),
			want: false,
		},
		{
			name: "unknown value is invalid",
			s:    Scope("global"),
			want: false,
		},
		{
			name: "case-sensitive: User is invalid",
			s:    Scope("User"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Valid(); got != tt.want {
				t.Errorf("Scope.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScope_Validate(t *testing.T) {
	tests := []struct {
		name    string
		s       Scope
		wantErr bool
	}{
		{
			name:    "ScopeUser returns nil",
			s:       ScopeUser,
			wantErr: false,
		},
		{
			name:    "ScopeProject returns nil",
			s:       ScopeProject,
			wantErr: false,
		},
		{
			name:    "empty string returns ErrInvalidScope",
			s:       Scope(""),
			wantErr: true,
		},
		{
			name:    "unknown value returns ErrInvalidScope",
			s:       Scope("global"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scope.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidScope) {
				t.Errorf("Scope.Validate() error = %v, want errors.Is(err, ErrInvalidScope)", err)
			}
		})
	}
}

func TestScope_String(t *testing.T) {
	tests := []struct {
		name string
		s    Scope
		want string
	}{
		{
			name: "ScopeUser stringifies to \"user\"",
			s:    ScopeUser,
			want: "user",
		},
		{
			name: "ScopeProject stringifies to \"project\"",
			s:    ScopeProject,
			want: "project",
		},
		{
			name: "empty string stringifies to empty",
			s:    Scope(""),
			want: "",
		},
		{
			name: "arbitrary value stringifies as-is",
			s:    Scope("global"),
			want: "global",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Scope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
