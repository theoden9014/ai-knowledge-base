package cli

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
)

func TestAggregateError_Error(t *testing.T) {
	type fields struct {
		Failures []TargetFailure
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "empty aggregate",
			fields: fields{Failures: nil},
			want:   "cli: partial failure across targets (0 failures)",
		},
		{
			name: "single failure",
			fields: fields{Failures: []TargetFailure{
				{Target: "claude", Err: errors.New("write fail")},
			}},
			want: "cli: partial failure across targets (1 failure): [claude: write fail]",
		},
		{
			name: "multi failure preserves order",
			fields: fields{Failures: []TargetFailure{
				{Target: "claude", Err: errors.New("a")},
				{Target: "gemini", Err: errors.New("b")},
			}},
			want: "cli: partial failure across targets (2 failures): [claude: a; gemini: b]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &AggregateError{
				Failures: tt.fields.Failures,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("AggregateError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAggregateError_Unwrap(t *testing.T) {
	errClaude := errors.New("claude err")
	errGemini := errors.New("gemini err")
	type fields struct {
		Failures []TargetFailure
	}
	tests := []struct {
		name   string
		fields fields
		want   []error
	}{
		{
			name:   "empty returns nil",
			fields: fields{Failures: nil},
			want:   nil,
		},
		{
			name: "single",
			fields: fields{Failures: []TargetFailure{
				{Target: "claude", Err: errClaude},
			}},
			want: []error{errClaude},
		},
		{
			name: "preserves order",
			fields: fields{Failures: []TargetFailure{
				{Target: "claude", Err: errClaude},
				{Target: "gemini", Err: errGemini},
			}},
			want: []error{errClaude, errGemini},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &AggregateError{
				Failures: tt.fields.Failures,
			}
			got := e.Unwrap()
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("AggregateError.Unwrap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAggregateError_Is(t *testing.T) {
	type fields struct {
		Failures []TargetFailure
	}
	type args struct {
		target error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "matches ErrPartialFailure",
			fields: fields{Failures: nil},
			args:   args{target: ErrPartialFailure},
			want:   true,
		},
		{
			name: "matches ErrPartialFailure with failures",
			fields: fields{Failures: []TargetFailure{
				{Target: "claude", Err: errors.New("x")},
			}},
			args: args{target: ErrPartialFailure},
			want: true,
		},
		{
			name:   "does not match arbitrary error",
			fields: fields{Failures: nil},
			args:   args{target: errors.New("other")},
			want:   false,
		},
		{
			name:   "does not match child sentinel directly (handled via Unwrap)",
			fields: fields{Failures: nil},
			args:   args{target: inventory.ErrAlreadyInstalled},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &AggregateError{
				Failures: tt.fields.Failures,
			}
			if got := e.Is(tt.args.target); got != tt.want {
				t.Errorf("AggregateError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAggregateError_errorsIs verifies that errors.Is composes correctly
// across both ErrPartialFailure (matched via Is) and child sentinels
// (matched via Unwrap() []error). This is the consumer-facing behavior
// that motivated the reviewer's High #3 request.
func TestAggregateError_errorsIs(t *testing.T) {
	child := errors.New("inner")
	childWrapper := inventory.ErrAlreadyInstalled
	agg := &AggregateError{Failures: []TargetFailure{
		{Target: "claude", Err: child},
		{Target: "gemini", Err: childWrapper},
	}}
	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{name: "matches ErrPartialFailure", target: ErrPartialFailure, want: true},
		{name: "matches child via Unwrap", target: childWrapper, want: true},
		{name: "matches child direct via Unwrap", target: child, want: true},
		{name: "does not match unrelated sentinel", target: ErrUsage, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(agg, tt.target)
			if got != tt.want {
				t.Errorf("errors.Is(agg, %v) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}
