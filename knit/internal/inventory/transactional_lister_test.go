package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestTransactionalLister_Target(t *testing.T) {
	f := newTransactionalFixture(t, true)
	l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
	if got := l.Target(); got != txTestTarget {
		t.Errorf("Target() = %q, want %q", got, txTestTarget)
	}
}

func TestTransactionalLister_List(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid scope", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
		_, err := l.List(ctx, Scope("bogus"))
		if !errors.Is(err, ErrInvalidScope) {
			t.Errorf("List() error = %v, want ErrInvalidScope", err)
		}
	})

	t.Run("project root not configured", func(t *testing.T) {
		f := newTransactionalFixture(t, false)
		l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
		_, err := l.List(ctx, ScopeProject)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("List() error = %v, want ErrProjectRootNotConfigured", err)
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
		got, err := l.List(ctx, ScopeUser)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty result, got %d entries", len(got))
		}
	})

	t.Run("returns installed entries", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
		first := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		second := sampleArtifact(t, "agents/x.toml", txTestTarget)
		if _, err := inst.Install(ctx, ScopeUser, first); err != nil {
			t.Fatalf("Install first: %v", err)
		}
		if _, err := inst.Install(ctx, ScopeUser, second); err != nil {
			t.Fatalf("Install second: %v", err)
		}
		got, err := l.List(ctx, ScopeUser)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 installations, got %d", len(got))
		}
		for _, in := range got {
			if in.Label.Target != txTestTarget {
				t.Errorf("Installation.Label.Target = %q, want %q", in.Label.Target, txTestTarget)
			}
			if in.Label.Scope != ScopeUser {
				t.Errorf("Installation.Label.Scope = %q, want %q", in.Label.Scope, ScopeUser)
			}
		}
	})

	t.Run("orphan label excluded", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		l := must(NewTransactionalLister(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		if _, err := inst.Install(ctx, ScopeUser, artifact); err != nil {
			t.Fatalf("Install: %v", err)
		}
		// Simulate external removal of the artifact (label remains).
		rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
		abs := must(f.resolver.Resolve(ScopeUser, rel))
		if err := f.store.Remove(ctx, abs); err != nil {
			t.Fatalf("simulate external remove: %v", err)
		}
		got, err := l.List(ctx, ScopeUser)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected orphan label to be excluded, got %d entries", len(got))
		}
	})
}
