package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory/inventorytest"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestInstaller_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	if got, want := NewInstaller(userRoot, projectRoot, labels).Target(), Target; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestInstaller_Contract(t *testing.T) {
	inventorytest.RunInstallerContract(t, func(t *testing.T) inventorytest.InstallerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		u := NewUninstaller(userRoot, projectRoot, labels)
		return inventorytest.InstallerHarness{
			Installer:       i,
			SupportedTarget: Target,
			SampleArtifact:  makeSampleArtifact(),
			Uninstaller:     u,
		}
	})
}

func TestInstaller_Install_WritesFileAndSidecar(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	a := makeSampleArtifact()
	got, err := i.Install(context.Background(), inventory.ScopeUser, a)
	if err != nil {
		t.Fatalf("Install() err = %v", err)
	}
	// The artifact file is written.
	want := filepath.Join(userRoot, "skills", "sample", "SKILL.md")
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("expected artifact written at %q, ReadFile err = %v", want, err)
	}
	if diff := cmp.Diff(a.Content, body); diff != "" {
		t.Errorf("written content mismatch (-want +got):\n%s", diff)
	}
	// The sidecar exists.
	sidecarDir := filepath.Join(userRoot, ".knit", "labels", "codex", "user")
	entries, err := os.ReadDir(sidecarDir)
	if err != nil {
		t.Fatalf("sidecar dir not found: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("sidecar count = %d, want 1", len(entries))
	}
	// The Installation struct is populated correctly.
	if got.Label.Target != Target || got.Label.Scope != inventory.ScopeUser {
		t.Errorf("Installation.Label = %+v, want Target=%q Scope=%q", got.Label, Target, inventory.ScopeUser)
	}
	if got.ID != inventory.InstallationID(a.Path) {
		t.Errorf("Installation.ID = %q, want %q", got.ID, a.Path)
	}
	if diff := cmp.Diff(a.SourceEntryIDs, got.Provenance.SourceEntryIDs); diff != "" {
		t.Errorf("Provenance.SourceEntryIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestInstaller_Install_ProjectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTempRootsUserOnly(t)
	i := NewInstaller(userRoot, "", labels)
	_, err := i.Install(context.Background(), inventory.ScopeProject, makeSampleArtifact())
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("Install() err = %v, want errors.Is(err, ErrProjectRootNotConfigured)", err)
	}
}

func TestInstaller_Install_InvalidArtifactPath(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	tests := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"absolute", "/etc/passwd"},
		{"parent escape", "../escape.md"},
		{"unknown top segment", "rules/x.md"},
		{"prompts subdirectory", "prompts/sub/x.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := makeSampleArtifact()
			a.Path = tt.path
			_, err := i.Install(context.Background(), inventory.ScopeUser, a)
			if !errors.Is(err, ErrInvalidArtifactPath) {
				t.Errorf("Install(path=%q) err = %v, want errors.Is(err, ErrInvalidArtifactPath)", tt.path, err)
			}
		})
	}
}

func TestInstaller_Install_TargetMismatch(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	a := makeSampleArtifact()
	a.Target = source.Target("other")
	_, err := i.Install(context.Background(), inventory.ScopeUser, a)
	if !errors.Is(err, inventory.ErrTargetMismatch) {
		t.Errorf("Install() err = %v, want ErrTargetMismatch", err)
	}
}

func TestInstaller_Install_UnmanagedArtifactExists(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	a := makeSampleArtifact()
	// Place an existing file that simulates a user-authored file (no sidecar).
	dest := filepath.Join(userRoot, "skills", "sample", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(dest, []byte("user handwritten"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := i.Install(context.Background(), inventory.ScopeUser, a)
	if !errors.Is(err, ErrUnmanagedArtifactExists) {
		t.Errorf("Install() err = %v, want ErrUnmanagedArtifactExists", err)
	}
	// The existing file is not overwritten.
	body, rerr := os.ReadFile(dest)
	if rerr != nil {
		t.Fatalf("ReadFile after failed Install: %v", rerr)
	}
	if string(body) != "user handwritten" {
		t.Errorf("existing file overwritten: %q", string(body))
	}
}

func TestInstaller_Install_ManagedExistingReturnsAlreadyInstalled(t *testing.T) {
	// Reinstalling to a path already managed by knit (with a sidecar) returns
	// ErrAlreadyInstalled, not ErrUnmanagedArtifactExists.
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	a := makeSampleArtifact()
	if _, err := i.Install(context.Background(), inventory.ScopeUser, a); err != nil {
		t.Fatalf("first Install err = %v", err)
	}
	_, err := i.Install(context.Background(), inventory.ScopeUser, a)
	if !errors.Is(err, inventory.ErrAlreadyInstalled) {
		t.Errorf("second Install err = %v, want ErrAlreadyInstalled", err)
	}
}

// TestInstaller_Install_errorOrdering verifies Install's error precedence.
// The order is aligned with claude/gemini:
//
//	ErrTargetMismatch > ErrInvalidScope > ErrProjectRootNotConfigured >
//	ErrInvalidArtifactPath > ErrAlreadyInstalled > ErrUnmanagedArtifactExists
func TestInstaller_Install_errorOrdering(t *testing.T) {
	t.Run("target mismatch precedes invalid scope", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		a := makeSampleArtifact()
		a.Target = source.Target("__other__")
		_, err := i.Install(context.Background(), inventory.Scope("__bogus__"), a)
		if !errors.Is(err, inventory.ErrTargetMismatch) {
			t.Errorf("err = %v, want ErrTargetMismatch", err)
		}
	})
	t.Run("invalid scope precedes invalid path", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		a := makeSampleArtifact()
		a.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.Scope("__bogus__"), a)
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("err = %v, want ErrInvalidScope", err)
		}
	})
	t.Run("project root not configured precedes invalid path", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		i := NewInstaller(userRoot, "", labels)
		a := makeSampleArtifact()
		a.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.ScopeProject, a)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("err = %v, want ErrProjectRootNotConfigured", err)
		}
	})
	t.Run("invalid path precedes already-installed", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		// Seed a normal artifact first.
		if _, err := i.Install(context.Background(), inventory.ScopeUser, makeSampleArtifact()); err != nil {
			t.Fatalf("seed Install error: %v", err)
		}
		bad := makeSampleArtifact()
		bad.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.ScopeUser, bad)
		if !errors.Is(err, ErrInvalidArtifactPath) {
			t.Errorf("err = %v, want ErrInvalidArtifactPath", err)
		}
	})
	t.Run("already-installed precedes unmanaged-artifact-exists", func(t *testing.T) {
		// When both the sidecar and the artifact file exist (that is, they came
		// from knit), a repeated Install must prefer ErrAlreadyInstalled over
		// ErrUnmanagedArtifactExists.
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		a := makeSampleArtifact()
		if _, err := i.Install(context.Background(), inventory.ScopeUser, a); err != nil {
			t.Fatalf("seed Install error: %v", err)
		}
		_, err := i.Install(context.Background(), inventory.ScopeUser, a)
		if !errors.Is(err, inventory.ErrAlreadyInstalled) {
			t.Errorf("err = %v, want ErrAlreadyInstalled (sidecar precedes unmanaged check)", err)
		}
		if errors.Is(err, ErrUnmanagedArtifactExists) {
			t.Errorf("err must not be ErrUnmanagedArtifactExists when sidecar exists; got %v", err)
		}
	})
}
