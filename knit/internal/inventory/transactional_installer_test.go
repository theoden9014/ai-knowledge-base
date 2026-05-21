package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestTransactionalInstaller_Target(t *testing.T) {
	f := newTransactionalFixture(t, true)
	inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
	if got := inst.Target(); got != txTestTarget {
		t.Errorf("Target() = %q, want %q", got, txTestTarget)
	}
}

func TestTransactionalInstaller_Install(t *testing.T) {
	ctx := context.Background()

	t.Run("target mismatch", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", source.Target("other-target"))
		_, err := inst.Install(ctx, ScopeUser, artifact)
		if !errors.Is(err, ErrTargetMismatch) {
			t.Errorf("Install() error = %v, want ErrTargetMismatch", err)
		}
	})

	t.Run("invalid artifact path", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := source.Artifact{Target: txTestTarget, Path: "/abs/path"}
		_, err := inst.Install(ctx, ScopeUser, artifact)
		if !errors.Is(err, source.ErrInvalidArtifactPath) {
			t.Errorf("Install() error = %v, want ErrInvalidArtifactPath", err)
		}
	})

	t.Run("path policy violation", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "forbidden/path.md", txTestTarget)
		_, err := inst.Install(ctx, ScopeUser, artifact)
		if !errors.Is(err, source.ErrInvalidArtifactPath) {
			t.Errorf("Install() error = %v, want ErrInvalidArtifactPath", err)
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		_, err := inst.Install(ctx, Scope("bogus"), artifact)
		if !errors.Is(err, ErrInvalidScope) {
			t.Errorf("Install() error = %v, want ErrInvalidScope", err)
		}
	})

	t.Run("project root not configured", func(t *testing.T) {
		f := newTransactionalFixture(t, false)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		_, err := inst.Install(ctx, ScopeProject, artifact)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("Install() error = %v, want ErrProjectRootNotConfigured", err)
		}
	})

	t.Run("unmanaged artifact exists", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		// Seed the storage backend with an artifact but no label.
		rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
		abs := must(f.resolver.Resolve(ScopeUser, rel))
		if err := f.store.Write(ctx, abs, []byte("pre-existing"), 0o644); err != nil {
			t.Fatalf("seed write: %v", err)
		}
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		_, err := inst.Install(ctx, ScopeUser, artifact)
		if !errors.Is(err, ErrUnmanagedArtifactExists) {
			t.Errorf("Install() error = %v, want ErrUnmanagedArtifactExists", err)
		}
	})

	t.Run("already installed", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		if _, err := inst.Install(ctx, ScopeUser, artifact); err != nil {
			t.Fatalf("first Install: %v", err)
		}
		_, err := inst.Install(ctx, ScopeUser, artifact)
		if !errors.Is(err, ErrAlreadyInstalled) {
			t.Errorf("second Install() error = %v, want ErrAlreadyInstalled", err)
		}
	})

	t.Run("happy path user scope", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		got, err := inst.Install(ctx, ScopeUser, artifact)
		if err != nil {
			t.Fatalf("Install() unexpected error: %v", err)
		}
		if got.Label.Target != txTestTarget {
			t.Errorf("Installation.Label.Target = %q, want %q", got.Label.Target, txTestTarget)
		}
		if got.Label.Scope != ScopeUser {
			t.Errorf("Installation.Label.Scope = %q, want %q", got.Label.Scope, ScopeUser)
		}
		if got.ID == "" {
			t.Errorf("Installation.ID is empty")
		}
		rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
		abs := must(f.resolver.Resolve(ScopeUser, rel))
		present, err := f.store.Exists(ctx, abs)
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if !present {
			t.Errorf("artifact not written to store at %q", abs.String())
		}
		if _, err := f.labels.Get(ctx, ScopeUser, got.ID); err != nil {
			t.Errorf("label not stored: %v", err)
		}
	})

	t.Run("happy path project scope", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "agents/x.toml", txTestTarget)
		got, err := inst.Install(ctx, ScopeProject, artifact)
		if err != nil {
			t.Fatalf("Install() unexpected error: %v", err)
		}
		if got.Label.Scope != ScopeProject {
			t.Errorf("Installation.Label.Scope = %q, want %q", got.Label.Scope, ScopeProject)
		}
	})
}
