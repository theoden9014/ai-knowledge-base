package source

import (
	"errors"
	"testing"
)

func TestNewSkillAsset_normal(t *testing.T) {
	t.Parallel()
	t.Run("relative path with directory", func(t *testing.T) {
		t.Parallel()
		got, err := NewSkillAsset("scripts/foo.sh", []byte("body"))
		if err != nil {
			t.Fatalf("NewSkillAsset returned err: %v", err)
		}
		if got.Path() != "scripts/foo.sh" {
			t.Errorf("Path() = %q, want %q", got.Path(), "scripts/foo.sh")
		}
		if string(got.Content()) != "body" {
			t.Errorf("Content() = %q, want %q", string(got.Content()), "body")
		}
		if got.IsZero() {
			t.Error("IsZero() = true, want false")
		}
	})

	t.Run("subdirectory path", func(t *testing.T) {
		t.Parallel()
		_, err := NewSkillAsset("references/sub/bar.md", []byte(""))
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
}

func TestNewSkillAsset_invalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		relPath string
	}{
		{"empty", ""},
		{"absolute", "/etc/foo"},
		{"parent traversal", "../escape.sh"},
		{"contains parent traversal", "scripts/../foo.sh"},
		{"SKILL.md exact match", "SKILL.md"},
		{"backslash separator", "scripts\\foo.sh"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewSkillAsset(tc.relPath, []byte("x"))
			if !errors.Is(err, ErrInvalidSkillAssetPath) {
				t.Errorf("err = %v, want ErrInvalidSkillAssetPath", err)
			}
		})
	}
}

func TestSkillAsset_defensiveCopy(t *testing.T) {
	t.Parallel()
	src := []byte("body")
	a, err := NewSkillAsset("a.txt", src)
	if err != nil {
		t.Fatalf("NewSkillAsset err: %v", err)
	}
	src[0] = 'X'
	if string(a.Content()) != "body" {
		t.Errorf("Content() = %q, want %q (defensive copy broken)", string(a.Content()), "body")
	}
}

func TestSkillAsset_zero(t *testing.T) {
	t.Parallel()
	var z SkillAsset
	if !z.IsZero() {
		t.Error("zero value IsZero() = false, want true")
	}
}
