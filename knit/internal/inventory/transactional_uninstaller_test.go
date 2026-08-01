package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestTransactionalUninstaller_Target(t *testing.T) {
	f := newTransactionalFixture(t, true)
	un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
	if got := un.Target(); got != txTestTarget {
		t.Errorf("Target() = %q, want %q", got, txTestTarget)
	}
}

func TestTransactionalUninstaller_Uninstall(t *testing.T) {
	ctx := context.Background()

	t.Run("target mismatch", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		installation := Installation{
			Label: Label{Target: source.Target("other-target"), Scope: ScopeUser},
			Artifact: source.Artifact{
				Target: source.Target("other-target"),
				Path:   "skills/foo/SKILL.md",
			},
		}
		err := un.Uninstall(ctx, installation)
		if !errors.Is(err, ErrTargetMismatch) {
			t.Errorf("Uninstall() error = %v, want ErrTargetMismatch", err)
		}
	})

	t.Run("installation not found", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		installation := Installation{
			ID:    InstallationID("skills/foo/SKILL.md"),
			Label: Label{Target: txTestTarget, Scope: ScopeUser},
			Artifact: source.Artifact{
				Target: txTestTarget,
				Path:   "skills/foo/SKILL.md",
			},
		}
		err := un.Uninstall(ctx, installation)
		if !errors.Is(err, ErrInstallationNotFound) {
			t.Errorf("Uninstall() error = %v, want ErrInstallationNotFound", err)
		}
	})

	t.Run("happy path removes artifact and label", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		installed, err := inst.Install(ctx, ScopeUser, artifact)
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
		if err := un.Uninstall(ctx, installed); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}
		rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
		abs := must(f.resolver.Resolve(ScopeUser, rel))
		present, _ := f.store.Exists(ctx, abs)
		if present {
			t.Errorf("artifact still present after Uninstall")
		}
		if _, err := f.labels.Get(ctx, ScopeUser, installed.ID); !errors.Is(err, ErrInstallationNotFound) {
			t.Errorf("label still present: err = %v", err)
		}
	})

	t.Run("orphan label removes label only", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		artifact := sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)
		installed, err := inst.Install(ctx, ScopeUser, artifact)
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
		// Simulate external deletion of the artifact while the label remains.
		rel := must(source.NewArtifactPath("skills/foo/SKILL.md"))
		abs := must(f.resolver.Resolve(ScopeUser, rel))
		if err := f.store.Remove(ctx, abs); err != nil {
			t.Fatalf("simulate external remove: %v", err)
		}
		if err := un.Uninstall(ctx, installed); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}
		if _, err := f.labels.Get(ctx, ScopeUser, installed.ID); !errors.Is(err, ErrInstallationNotFound) {
			t.Errorf("orphan label not removed: err = %v", err)
		}
	})

	t.Run("persisted label path is authoritative", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		first := must(inst.Install(ctx, ScopeUser, sampleArtifact(t, "skills/first/SKILL.md", txTestTarget)))
		second := must(inst.Install(ctx, ScopeUser, sampleArtifact(t, "skills/second/SKILL.md", txTestTarget)))

		tampered := first
		tampered.Artifact.Path = second.Artifact.Path
		if err := un.Uninstall(ctx, tampered); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}

		firstPath := must(f.resolver.Resolve(ScopeUser, must(source.NewArtifactPath(first.Artifact.Path))))
		secondPath := must(f.resolver.Resolve(ScopeUser, must(source.NewArtifactPath(second.Artifact.Path))))
		if present, _ := f.store.Exists(ctx, firstPath); present {
			t.Error("first artifact still present")
		}
		if present, _ := f.store.Exists(ctx, secondPath); !present {
			t.Error("second artifact was deleted through tampered Artifact.Path")
		}
	})

	t.Run("empty id cannot fall back to caller path", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		installed := must(inst.Install(ctx, ScopeUser, sampleArtifact(t, "skills/foo/SKILL.md", txTestTarget)))
		installed.ID = ""

		err := un.Uninstall(ctx, installed)
		if !errors.Is(err, ErrInvalidInstallationID) {
			t.Fatalf("Uninstall err = %v, want ErrInvalidInstallationID", err)
		}
		abs := must(f.resolver.Resolve(ScopeUser, must(source.NewArtifactPath(installed.Artifact.Path))))
		if present, _ := f.store.Exists(ctx, abs); !present {
			t.Error("artifact was deleted through empty-ID fallback")
		}
	})

	t.Run("corrupted label path cannot select another artifact", func(t *testing.T) {
		f := newTransactionalFixture(t, true)
		inst := must(NewTransactionalInstaller(f.store, f.labels, f.resolver))
		un := must(NewTransactionalUninstaller(f.store, f.labels, f.resolver))
		first := must(inst.Install(ctx, ScopeUser, sampleArtifact(t, "skills/first/SKILL.md", txTestTarget)))
		second := must(inst.Install(ctx, ScopeUser, sampleArtifact(t, "skills/second/SKILL.md", txTestTarget)))

		data := must(f.labels.Get(ctx, ScopeUser, first.ID))
		data.ArtifactPath = second.Artifact.Path
		if err := f.labels.Delete(ctx, ScopeUser, first.ID); err != nil {
			t.Fatalf("Delete label: %v", err)
		}
		if err := f.labels.Set(ctx, ScopeUser, first.ID, data); err != nil {
			t.Fatalf("Set corrupted label: %v", err)
		}

		err := un.Uninstall(ctx, first)
		if !errors.Is(err, ErrInstallationIdentityMismatch) {
			t.Fatalf("Uninstall err = %v, want ErrInstallationIdentityMismatch", err)
		}
		secondPath := must(f.resolver.Resolve(ScopeUser, must(source.NewArtifactPath(second.Artifact.Path))))
		if present, _ := f.store.Exists(ctx, secondPath); !present {
			t.Error("second artifact was deleted through corrupted label")
		}
	})
}
