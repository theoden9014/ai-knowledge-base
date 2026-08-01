package source

import (
	"errors"
	"testing"
)

func mustAsset(t *testing.T, p string) SkillAsset {
	t.Helper()
	a, err := NewSkillAsset(p, []byte("x"))
	if err != nil {
		t.Fatalf("NewSkillAsset(%q): %v", p, err)
	}
	return a
}

func TestNewSkillMeta_normal(t *testing.T) {
	t.Parallel()

	t.Run("empty assets", func(t *testing.T) {
		t.Parallel()
		m, err := NewSkillMeta("skills/orchestrator", nil)
		if err != nil {
			t.Fatalf("NewSkillMeta err: %v", err)
		}
		if m == nil {
			t.Fatal("nil meta")
		}
		if m.Root() != "skills/orchestrator" {
			t.Errorf("Root() = %q, want %q", m.Root(), "skills/orchestrator")
		}
		if got := m.Assets(); len(got) != 0 {
			t.Errorf("Assets() len = %d, want 0", len(got))
		}
	})

	t.Run("multiple assets", func(t *testing.T) {
		t.Parallel()
		a1 := mustAsset(t, "scripts/a.sh")
		a2 := mustAsset(t, "refs/b.md")
		m, err := NewSkillMeta("skills/foo", []SkillAsset{a1, a2})
		if err != nil {
			t.Fatalf("NewSkillMeta err: %v", err)
		}
		got := m.Assets()
		if len(got) != 2 {
			t.Fatalf("Assets() len = %d, want 2", len(got))
		}
	})
}

func TestNewSkillMetaWithInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		invocation SkillInvocation
		want       SkillInvocation
		wantErr    error
	}{
		{name: "both", invocation: SkillInvocationBoth, want: SkillInvocationBoth},
		{name: "manual", invocation: SkillInvocationManual, want: SkillInvocationManual},
		{name: "empty normalizes to both", invocation: "", want: SkillInvocationBoth},
		{name: "unknown rejected", invocation: "automatic", wantErr: ErrInvalidSkillInvocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := NewSkillMetaWithInvocation("skills/foo", nil, tt.invocation)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && m.Invocation() != tt.want {
				t.Errorf("Invocation() = %q, want %q", m.Invocation(), tt.want)
			}
		})
	}
}

func TestNewSkillMeta_invalidRoot(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		root string
	}{
		{"empty", ""},
		{"trailing slash", "skills/foo/"},
		{"parent traversal", "skills/../escape"},
		{"absolute", "/skills/foo"},
		{"backslash", "skills\\foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewSkillMeta(tc.root, nil)
			if !errors.Is(err, ErrInvalidSkillRoot) {
				t.Errorf("err = %v, want ErrInvalidSkillRoot", err)
			}
		})
	}
}

func TestNewSkillMeta_duplicateAssets(t *testing.T) {
	t.Parallel()
	a := mustAsset(t, "scripts/run.sh")
	_, err := NewSkillMeta("skills/foo", []SkillAsset{a, a})
	if !errors.Is(err, ErrDuplicateSkillAsset) {
		t.Errorf("err = %v, want ErrDuplicateSkillAsset", err)
	}
}

func TestSkillMeta_defensiveCopy(t *testing.T) {
	t.Parallel()
	a := mustAsset(t, "scripts/run.sh")
	in := []SkillAsset{a}
	m, err := NewSkillMeta("skills/foo", in)
	if err != nil {
		t.Fatalf("NewSkillMeta err: %v", err)
	}
	in = append(in, mustAsset(t, "refs/x.md"))
	_ = in
	if got := m.Assets(); len(got) != 1 {
		t.Errorf("Assets() len = %d, want 1 (defensive copy broken)", len(got))
	}
}
