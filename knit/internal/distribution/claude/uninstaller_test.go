package claude

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

// TestUninstaller_Contract verifies Uninstaller's inventory.Uninstaller
// contract using the inventorytest contract harness.
func TestUninstaller_Contract(t *testing.T) {
	inventorytest.RunUninstallerContract(t, func(t *testing.T) inventorytest.UninstallerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		ins := NewInstaller(userRoot, projectRoot, labels)
		uns := NewUninstaller(userRoot, projectRoot, labels)

		seed := func(t *testing.T, scope inventory.Scope) inventory.Installation {
			t.Helper()
			art := sampleSkillArtifact()
			inst, err := ins.Install(context.Background(), scope, art)
			if err != nil {
				t.Fatalf("seed Install error: %v", err)
			}
			return inst
		}
		fabricate := func(t *testing.T, scope inventory.Scope) inventory.Installation {
			t.Helper()
			art := sampleSkillArtifact()
			return inventory.Installation{
				ID:    ins.resolver.installationID(art.Path),
				Label: inventory.Label{Target: Target, Scope: scope},
				Provenance: inventory.Provenance{
					SourceEntryIDs: []string{"p.skill.sample"},
				},
				Artifact: source.Artifact{Target: Target, Path: art.Path},
			}
		}

		return inventorytest.UninstallerHarness{
			Uninstaller:      uns,
			SupportedTarget:  Target,
			SeedInstalled:    seed,
			FabricateMissing: fabricate,
		}
	})
}

func TestUninstaller_Target(t *testing.T) {
	labels := inventory.NewSidecarLabelStore(Target, "/u/.knit/labels", "")
	uns := NewUninstaller("/u/.claude", "/p/.claude", labels)
	if got := uns.Target(); got != Target {
		t.Errorf("Uninstaller.Target() = %v, want %v", got, Target)
	}
}

// TestUninstaller_Uninstall_removesArtifactAndLabel verifies that the success
// path removes both the artifact file and the label record.
func TestUninstaller_Uninstall_removesArtifactAndLabel(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	uns := NewUninstaller(userRoot, projectRoot, labels)

	inst, err := ins.Install(context.Background(), inventory.ScopeUser, sampleSkillArtifact())
	if err != nil {
		t.Fatalf("Install seed: %v", err)
	}
	if err := uns.Uninstall(context.Background(), inst); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	absArtifact := filepath.Join(userRoot, inst.Artifact.Path)
	if _, err := os.Stat(absArtifact); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("artifact still exists: err = %v", err)
	}
	if _, err := labels.Get(context.Background(), inventory.ScopeUser, inst.ID); !errors.Is(err, inventory.ErrInstallationNotFound) {
		t.Errorf("label still present: err = %v, want ErrInstallationNotFound", err)
	}
}

// TestUninstaller_Uninstall_isNotIdempotentByItself verifies that a repeated
// Uninstall returns ErrInstallationNotFound
// (idempotence is the caller's responsibility).
func TestUninstaller_Uninstall_isNotIdempotentByItself(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	uns := NewUninstaller(userRoot, projectRoot, labels)

	inst, err := ins.Install(context.Background(), inventory.ScopeUser, sampleSkillArtifact())
	if err != nil {
		t.Fatalf("Install seed: %v", err)
	}
	if err := uns.Uninstall(context.Background(), inst); err != nil {
		t.Fatalf("first Uninstall: %v", err)
	}
	err = uns.Uninstall(context.Background(), inst)
	if !errors.Is(err, inventory.ErrInstallationNotFound) {
		t.Errorf("second Uninstall err = %v, want ErrInstallationNotFound", err)
	}
}

// TestUninstaller_Uninstall_errorOrdering verifies error precedence.
//
//	ErrTargetMismatch > ErrInvalidScope > ErrProjectRootNotConfigured >
//	ErrInstallationNotFound
func TestUninstaller_Uninstall_errorOrdering(t *testing.T) {
	t.Run("target mismatch precedes invalid scope", func(t *testing.T) {
		userRoot, projectRoot, labels := newTempRoots(t)
		uns := NewUninstaller(userRoot, projectRoot, labels)
		inst := inventory.Installation{
			Label:    inventory.Label{Target: source.Target("__other__"), Scope: inventory.Scope("__bogus__")},
			Artifact: source.Artifact{Target: Target, Path: "skills/x/SKILL.md"},
		}
		err := uns.Uninstall(context.Background(), inst)
		if !errors.Is(err, inventory.ErrTargetMismatch) {
			t.Errorf("err = %v, want ErrTargetMismatch", err)
		}
	})
	t.Run("invalid scope precedes project root not configured", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		uns := NewUninstaller(userRoot, "", labels)
		inst := inventory.Installation{
			Label:    inventory.Label{Target: Target, Scope: inventory.Scope("__bogus__")},
			Artifact: source.Artifact{Target: Target, Path: "skills/x/SKILL.md"},
		}
		err := uns.Uninstall(context.Background(), inst)
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("err = %v, want ErrInvalidScope", err)
		}
	})
	t.Run("project root not configured precedes installation not found", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		uns := NewUninstaller(userRoot, "", labels)
		inst := inventory.Installation{
			Label:    inventory.Label{Target: Target, Scope: inventory.ScopeProject},
			Artifact: source.Artifact{Target: Target, Path: "skills/x/SKILL.md"},
		}
		err := uns.Uninstall(context.Background(), inst)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("err = %v, want ErrProjectRootNotConfigured", err)
		}
	})
}
