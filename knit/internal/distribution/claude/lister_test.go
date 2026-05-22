package claude

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory/inventorytest"
)

// TestLister_Contract verifies Lister's inventory.Lister contract using the
// inventorytest contract harness.
func TestLister_Contract(t *testing.T) {
	inventorytest.RunListerContract(t, func(t *testing.T) inventorytest.ListerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		ins := must(NewInstaller(userRoot, projectRoot, labels))
		lst := must(NewLister(userRoot, projectRoot, labels))
		seed := func(t *testing.T, scope inventory.Scope, n int) []inventory.Installation {
			t.Helper()
			result := make([]inventory.Installation, 0, n)
			for k := 0; k < n; k++ {
				art := sampleSkillArtifact()
				art.Path = fmt.Sprintf("skills/p-sample-%d/SKILL.md", k)
				art.SourceEntryIDs = []string{fmt.Sprintf("p.skill.sample-%d", k)}
				inst, err := ins.Install(context.Background(), scope, art)
				if err != nil {
					t.Fatalf("seed Install error: %v", err)
				}
				result = append(result, inst)
			}
			return result
		}
		return inventorytest.ListerHarness{
			Lister:          lst,
			SupportedTarget: Target,
			SeedInstalled:   seed,
		}
	})
}

func TestLister_Target(t *testing.T) {
	labels := inventory.NewSidecarLabelStore(Target, "/u/.knit/labels", "")
	lst := must(NewLister("/u/.claude", "/p/.claude", labels))
	if got := lst.Target(); got != Target {
		t.Errorf("Lister.Target() = %v, want %v", got, Target)
	}
}

// TestLister_List_returnsEmptyWhenLabelsAbsent verifies that an empty slice is
// returned when nothing has been installed.
func TestLister_List_returnsEmptyWhenLabelsAbsent(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	lst := must(NewLister(userRoot, projectRoot, labels))
	got, err := lst.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() returned %d items on empty inventory, want 0", len(got))
	}
}

// TestLister_List_excludesOrphanLabel verifies that a label record without a
// corresponding artifact file (a leftover) is excluded from results.
func TestLister_List_excludesOrphanLabel(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	ins := must(NewInstaller(userRoot, projectRoot, labels))
	uns := must(NewUninstaller(userRoot, projectRoot, labels))
	lst := must(NewLister(userRoot, projectRoot, labels))

	a1 := sampleSkillArtifact()
	a1.Path = "skills/p-a/SKILL.md"
	inst1, err := ins.Install(context.Background(), inventory.ScopeUser, a1)
	if err != nil {
		t.Fatalf("Install a1: %v", err)
	}
	a2 := sampleSkillArtifact()
	a2.Path = "skills/p-b/SKILL.md"
	if _, err := ins.Install(context.Background(), inventory.ScopeUser, a2); err != nil {
		t.Fatalf("Install a2: %v", err)
	}
	absArtifact := filepath.Join(userRoot, inst1.Artifact.Path)
	if err := os.Remove(absArtifact); err != nil {
		t.Fatalf("remove a1 artifact: %v", err)
	}
	got, err := lst.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("List() = %d items, want 1 (orphan should be excluded)", len(got))
	}
	for _, it := range got {
		if it.Artifact.Path == a1.Path {
			t.Errorf("orphan installation was not excluded: %+v", it)
		}
	}
	if err := uns.Uninstall(context.Background(), inst1); err != nil {
		t.Errorf("Uninstall orphan: %v", err)
	}
}

// TestLister_List_excludesUnmanagedFile verifies that an artifact file not
// managed by knit and lacking a label is not included in results.
func TestLister_List_excludesUnmanagedFile(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	lst := must(NewLister(userRoot, projectRoot, labels))
	abs := filepath.Join(userRoot, "skills", "manual", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte("manual"), 0o644); err != nil {
		t.Fatalf("write manual: %v", err)
	}
	got, err := lst.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() = %d items, want 0 (unmanaged file should be excluded)", len(got))
	}
}

// TestLister_List_errorOrdering verifies error precedence.
//
//	ErrInvalidScope > ErrProjectRootNotConfigured
func TestLister_List_errorOrdering(t *testing.T) {
	t.Run("invalid scope precedes project root not configured", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		lst := must(NewLister(userRoot, "", labels))
		_, err := lst.List(context.Background(), inventory.Scope("__bogus__"))
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("err = %v, want ErrInvalidScope", err)
		}
	})
	t.Run("project root not configured for ScopeProject", func(t *testing.T) {
		userRoot, labels := newTempRootsUserOnly(t)
		lst := must(NewLister(userRoot, "", labels))
		_, err := lst.List(context.Background(), inventory.ScopeProject)
		if !errors.Is(err, ErrProjectRootNotConfigured) {
			t.Errorf("err = %v, want ErrProjectRootNotConfigured", err)
		}
	})
}
