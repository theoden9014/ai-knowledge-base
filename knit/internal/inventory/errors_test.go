package inventory

import (
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrors_Distinct verifies that each sentinel error defined by the
// inventory package remains distinguishable from the others and is not
// confused by errors.Is.
func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{
		ErrInvalidScope,
		ErrTargetMismatch,
		ErrAlreadyInstalled,
		ErrInstallationNotFound,
	}

	tests := []struct {
		name   string
		target error
		others []error
	}{
		{
			name:   "ErrInvalidScope is distinct from the others",
			target: ErrInvalidScope,
			others: []error{ErrTargetMismatch, ErrAlreadyInstalled, ErrInstallationNotFound},
		},
		{
			name:   "ErrTargetMismatch is distinct from the others",
			target: ErrTargetMismatch,
			others: []error{ErrInvalidScope, ErrAlreadyInstalled, ErrInstallationNotFound},
		},
		{
			name:   "ErrAlreadyInstalled is distinct from the others",
			target: ErrAlreadyInstalled,
			others: []error{ErrInvalidScope, ErrTargetMismatch, ErrInstallationNotFound},
		},
		{
			name:   "ErrInstallationNotFound is distinct from the others",
			target: ErrInstallationNotFound,
			others: []error{ErrInvalidScope, ErrTargetMismatch, ErrAlreadyInstalled},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target == nil {
				t.Fatalf("sentinel error %q is nil", tt.name)
			}
			if !errors.Is(tt.target, tt.target) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.target, tt.target)
			}
			for _, other := range tt.others {
				if errors.Is(tt.target, other) {
					t.Errorf("errors.Is(%v, %v) = true, want false (sentinels should be distinct)", tt.target, other)
				}
			}
		})
	}

	if got, want := len(sentinels), 4; got != want {
		t.Errorf("len(sentinels) = %d, want %d (test list out of sync with package)", got, want)
	}
}

// TestSentinelErrors_WrapDetectable verifies the contract that callers can
// still detect sentinel errors with errors.Is even when a distribution
// implementation wraps them with fmt.Errorf.
func TestSentinelErrors_WrapDetectable(t *testing.T) {
	tests := []struct {
		name   string
		target error
	}{
		{name: "wrapped ErrInvalidScope", target: ErrInvalidScope},
		{name: "wrapped ErrTargetMismatch", target: ErrTargetMismatch},
		{name: "wrapped ErrAlreadyInstalled", target: ErrAlreadyInstalled},
		{name: "wrapped ErrInstallationNotFound", target: ErrInstallationNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("distribution: contextual info: %w", tt.target)
			if !errors.Is(wrapped, tt.target) {
				t.Errorf("errors.Is(wrapped, %v) = false, want true", tt.target)
			}
		})
	}
}
