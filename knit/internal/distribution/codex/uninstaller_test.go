package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory/inventorytest"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestUninstaller_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	if got, want := must(NewUninstaller(userRoot, projectRoot, labels)).Target(), Target; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestUninstaller_Contract(t *testing.T) {
	inventorytest.RunUninstallerContract(t, func(t *testing.T) inventorytest.UninstallerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := must(NewInstaller(userRoot, projectRoot, labels))
		u := must(NewUninstaller(userRoot, projectRoot, labels))
		return inventorytest.UninstallerHarness{
			Uninstaller:     u,
			SupportedTarget: Target,
			SeedInstalled: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				return seedInstall(t, i, scope, "skills/seeded/SKILL.md")
			},
			FabricateMissing: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				return inventory.Installation{
					ID: inventory.InstallationID("skills/never-installed/SKILL.md"),
					Label: inventory.Label{
						Target: Target,
						Scope:  scope,
					},
					Artifact: source.Artifact{
						Target: Target,
						Path:   "skills/never-installed/SKILL.md",
					},
				}
			},
		}
	})
}

func TestUninstaller_Uninstall_RemovesFileAndSidecarAndEmptyDir(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := must(NewInstaller(userRoot, projectRoot, labels))
	u := must(NewUninstaller(userRoot, projectRoot, labels))
	got := seedInstall(t, i, inventory.ScopeUser, "skills/x/SKILL.md")

	if err := u.Uninstall(context.Background(), got); err != nil {
		t.Fatalf("Uninstall() err = %v", err)
	}
	// The artifact file is removed.
	if _, err := os.Stat(filepath.Join(userRoot, "skills", "x", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("SKILL.md still exists after Uninstall, err = %v", err)
	}
	// The parent directory skills/x/ is also removed if it becomes empty.
	if _, err := os.Stat(filepath.Join(userRoot, "skills", "x")); !os.IsNotExist(err) {
		t.Errorf("skills/x/ directory still exists after Uninstall (should be removed when empty), err = %v", err)
	}
	// The sidecar is removed as well: verify via the LabelStore API so
	// the test does not encode SidecarLabelStore's internal layout.
	if _, gErr := labels.Get(context.Background(), inventory.ScopeUser, got.ID); !errors.Is(gErr, inventory.ErrInstallationNotFound) {
		t.Errorf("LabelStore.Get(%q) err = %v, want ErrInstallationNotFound", got.ID, gErr)
	}
}

func TestUninstaller_Uninstall_DoesNotRemoveDirIfOtherFilesRemain(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := must(NewInstaller(userRoot, projectRoot, labels))
	u := must(NewUninstaller(userRoot, projectRoot, labels))
	got := seedInstall(t, i, inventory.ScopeUser, "skills/x/SKILL.md")

	// Put a non-knit-managed file in the parent directory.
	extra := filepath.Join(userRoot, "skills", "x", "scripts.sh")
	if err := os.WriteFile(extra, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile extra: %v", err)
	}

	if err := u.Uninstall(context.Background(), got); err != nil {
		t.Fatalf("Uninstall() err = %v", err)
	}
	if _, err := os.Stat(extra); err != nil {
		t.Errorf("extra file should NOT be removed: stat err = %v", err)
	}
}

func TestUninstaller_Uninstall_ProjectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTempRootsUserOnly(t)
	u := must(NewUninstaller(userRoot, "", labels))
	inst := inventory.Installation{
		ID: inventory.InstallationID("AGENTS.md"),
		Label: inventory.Label{
			Target: Target,
			Scope:  inventory.ScopeProject,
		},
		Artifact: source.Artifact{Target: Target, Path: "AGENTS.md"},
	}
	err := u.Uninstall(context.Background(), inst)
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("Uninstall() err = %v, want ErrProjectRootNotConfigured", err)
	}
}
