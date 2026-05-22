package claude

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

// newTempRoots returns the user/project Inventory roots and a LabelStore wired
// against a temp filesystem. The label sidecars live under a separate knit
// root, mirroring how the cli factory composes them.
func newTempRoots(t *testing.T) (userRoot, projectRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user", ".claude")
	projectRoot = filepath.Join(base, "project", ".claude")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	projectLabelsRoot := filepath.Join(base, "project", ".knit", "labels")
	if err := os.MkdirAll(userRoot, 0o755); err != nil {
		t.Fatalf("mkdir userRoot: %v", err)
	}
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir projectRoot: %v", err)
	}
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, projectLabelsRoot)
	return userRoot, projectRoot, labels
}

// newTempRootsUserOnly returns a fixture where projectRoot is intentionally
// empty so callers can exercise ErrProjectRootNotConfigured paths.
func newTempRootsUserOnly(t *testing.T) (userRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user", ".claude")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	if err := os.MkdirAll(userRoot, 0o755); err != nil {
		t.Fatalf("mkdir userRoot: %v", err)
	}
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, "")
	return userRoot, labels
}

func sampleSkillArtifact() source.Artifact {
	return source.Artifact{
		Target:         Target,
		Path:           "skills/p-sample/SKILL.md",
		Content:        []byte("---\nname: p-sample\n---\nhello\n"),
		SourceEntryIDs: []string{"p.skill.sample"},
	}
}

// TestInstaller_Contract verifies Installer's inventory.Installer contract
// using the inventorytest contract harness.
func TestInstaller_Contract(t *testing.T) {
	inventorytest.RunInstallerContract(t, func(t *testing.T) inventorytest.InstallerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		ins := must(NewInstaller(userRoot, projectRoot, labels))
		uns := must(NewUninstaller(userRoot, projectRoot, labels))
		return inventorytest.InstallerHarness{
			Installer:       ins,
			SupportedTarget: Target,
			SampleArtifact:  sampleSkillArtifact(),
			Uninstaller:     uns,
		}
	})
}

func TestNewInstaller(t *testing.T) {
	tests := []struct {
		name        string
		userRoot    string
		projectRoot string
	}{
		{name: "constructs Installer with both roots", userRoot: "/u/.claude", projectRoot: "/p/.claude"},
		{name: "constructs Installer with empty projectRoot", userRoot: "/u/.claude", projectRoot: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := inventory.NewSidecarLabelStore(Target, "/u/.knit/labels", "")
			got := must(NewInstaller(tt.userRoot, tt.projectRoot, labels))
			if got == nil {
				t.Fatal("NewInstaller() returned nil")
			}
			if got.Target() != Target {
				t.Errorf("Installer.Target() = %q, want %q", got.Target(), Target)
			}
		})
	}
}

func TestInstaller_Target(t *testing.T) {
	labels := inventory.NewSidecarLabelStore(Target, "/u/.knit/labels", "")
	i := must(NewInstaller("/u/.claude", "/p/.claude", labels))
	if got := i.Target(); !cmp.Equal(Target, got) {
		t.Errorf("Installer.Target() = %v, want %v", got, Target)
	}
}

// TestInstaller_Install_errorOrdering verifies Install's error precedence.
//
//	ErrTargetMismatch > ErrInvalidScope > ErrProjectRootNotConfigured >
//	ErrInvalidArtifactPath > ErrAlreadyInstalled > ErrUnmanagedArtifactExists
func TestInstaller_Install_errorOrdering(t *testing.T) {
	t.Run("target mismatch precedes invalid scope", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := must(NewInstaller(userRoot, projectRoot, labels))
		art := sampleSkillArtifact()
		art.Target = source.Target("__other__")
		_, err := i.Install(context.Background(), inventory.Scope("__bogus__"), art)
		if !errors.Is(err, inventory.ErrTargetMismatch) {
			t.Errorf("err = %v, want ErrTargetMismatch", err)
		}
	})
	t.Run("invalid scope precedes invalid path", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := must(NewInstaller(userRoot, projectRoot, labels))
		art := sampleSkillArtifact()
		art.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.Scope("__bogus__"), art)
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("err = %v, want ErrInvalidScope", err)
		}
	})
	t.Run("project root not configured precedes invalid path", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		i := must(NewInstaller(userRoot, "", labels))
		art := sampleSkillArtifact()
		art.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.ScopeProject, art)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("err = %v, want ErrProjectRootNotConfigured", err)
		}
	})
	t.Run("invalid path precedes already-installed", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := must(NewInstaller(userRoot, projectRoot, labels))
		if _, err := i.Install(context.Background(), inventory.ScopeUser, sampleSkillArtifact()); err != nil {
			t.Fatalf("seed Install error: %v", err)
		}
		bad := sampleSkillArtifact()
		bad.Path = "../escape.md"
		_, err := i.Install(context.Background(), inventory.ScopeUser, bad)
		if !errors.Is(err, ErrInvalidArtifactPath) {
			t.Errorf("err = %v, want ErrInvalidArtifactPath", err)
		}
	})
}

// TestInstaller_Install_unmanagedArtifactExists verifies that
// ErrUnmanagedArtifactExists is returned when an artifact exists without a label.
func TestInstaller_Install_unmanagedArtifactExists(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := must(NewInstaller(userRoot, projectRoot, labels))
	art := sampleSkillArtifact()

	absArtifact := filepath.Join(userRoot, art.Path)
	if err := os.MkdirAll(filepath.Dir(absArtifact), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absArtifact, []byte("manual content"), 0o644); err != nil {
		t.Fatalf("write manual file: %v", err)
	}
	_, err := i.Install(context.Background(), inventory.ScopeUser, art)
	if !errors.Is(err, ErrUnmanagedArtifactExists) {
		t.Errorf("Install() err = %v, want ErrUnmanagedArtifactExists", err)
	}
	b, rErr := os.ReadFile(absArtifact)
	if rErr != nil {
		t.Fatalf("read manual file: %v", rErr)
	}
	if string(b) != "manual content" {
		t.Errorf("manual file was overwritten: got %q", string(b))
	}
}

// TestInstaller_Install_writesLabelAndArtifact verifies the success path
// writes both the label and the artifact file.
func TestInstaller_Install_writesLabelAndArtifact(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := must(NewInstaller(userRoot, projectRoot, labels))
	art := sampleSkillArtifact()
	wantSourceEntryIDs := []string{"p.skill.sample"}

	inst, err := i.Install(context.Background(), inventory.ScopeUser, art)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if inst.Label.Target != Target || inst.Label.Scope != inventory.ScopeUser {
		t.Errorf("Installation.Label = %+v", inst.Label)
	}
	if diff := cmp.Diff(wantSourceEntryIDs, inst.Provenance.SourceEntryIDs); diff != "" {
		t.Errorf("Provenance.SourceEntryIDs mismatch:\n%s", diff)
	}
	absArtifact := filepath.Join(userRoot, art.Path)
	if b, rErr := os.ReadFile(absArtifact); rErr != nil {
		t.Errorf("artifact missing: %v", rErr)
	} else if string(b) != string(art.Content) {
		t.Errorf("artifact content mismatch: got %q want %q", string(b), string(art.Content))
	}
	got, err := labels.Get(context.Background(), inventory.ScopeUser, inst.ID)
	if err != nil {
		t.Fatalf("LabelStore.Get: %v", err)
	}
	if got.ArtifactPath != art.Path {
		t.Errorf("label ArtifactPath = %q, want %q", got.ArtifactPath, art.Path)
	}
	if diff := cmp.Diff(wantSourceEntryIDs, got.SourceEntryIDs); diff != "" {
		t.Errorf("label SourceEntryIDs mismatch:\n%s", diff)
	}
}
