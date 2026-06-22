package source

import (
	"errors"
	"fmt"
	"testing"
)

func TestSkillResolutionErrors_umbrella(t *testing.T) {
	t.Parallel()
	cases := []error{
		ErrSkillPathNotFound,
		ErrSkillPathNotDirectory,
		ErrSkillBodyNotFound,
	}
	for _, sentinel := range cases {
		t.Run(sentinel.Error(), func(t *testing.T) {
			t.Parallel()
			wrapped := fmt.Errorf("source: foo: %w", sentinel)
			if !errors.Is(wrapped, sentinel) {
				t.Errorf("errors.Is(wrapped, %v) = false, want true", sentinel)
			}
			if !errors.Is(wrapped, ErrSkillResolution) {
				t.Errorf("errors.Is(wrapped, ErrSkillResolution) = false, want true")
			}
		})
	}
}

func TestSkillResolutionErrors_independence(t *testing.T) {
	t.Parallel()
	pairs := []struct {
		a, b error
	}{
		{ErrSkillPathNotFound, ErrSkillPathNotDirectory},
		{ErrSkillPathNotFound, ErrSkillBodyNotFound},
		{ErrSkillPathNotDirectory, ErrSkillBodyNotFound},
	}
	for _, p := range pairs {
		if errors.Is(p.a, p.b) {
			t.Errorf("errors.Is(%v, %v) = true, want false", p.a, p.b)
		}
	}
}

func TestSkillValueObjectErrors_notUmbrella(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("foo: %w", ErrInvalidSkillAssetPath)
	if errors.Is(wrapped, ErrSkillResolution) {
		t.Error("ErrInvalidSkillAssetPath must not satisfy ErrSkillResolution")
	}
	wrapped2 := fmt.Errorf("foo: %w", ErrInvalidSkillRoot)
	if errors.Is(wrapped2, ErrSkillResolution) {
		t.Error("ErrInvalidSkillRoot must not satisfy ErrSkillResolution")
	}
}
