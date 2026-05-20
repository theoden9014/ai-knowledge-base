package codex

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

func TestLister_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	if got, want := NewLister(userRoot, projectRoot, labels).Target(), Target; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestLister_Contract(t *testing.T) {
	inventorytest.RunListerContract(t, func(t *testing.T) inventorytest.ListerHarness {
		userRoot, projectRoot, labels := newTempRoots(t)
		i := NewInstaller(userRoot, projectRoot, labels)
		l := NewLister(userRoot, projectRoot, labels)
		return inventorytest.ListerHarness{
			Lister:          l,
			SupportedTarget: Target,
			SeedInstalled: func(t *testing.T, scope inventory.Scope, n int) []inventory.Installation {
				out := make([]inventory.Installation, 0, n)
				for k := 0; k < n; k++ {
					p := fmt.Sprintf("skills/s%d/SKILL.md", k)
					out = append(out, seedInstall(t, i, scope, p))
				}
				return out
			},
		}
	})
}

func TestLister_List_EmptyReturnsNilOrEmptySlice(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	l := NewLister(userRoot, projectRoot, labels)
	got, err := l.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List() on empty Inventory err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("List() on empty Inventory returned %d items, want 0", len(got))
	}
}

func TestLister_List_ExcludesOrphanedSidecar(t *testing.T) {
	userRoot, projectRoot, labels := newTempRoots(t)
	i := NewInstaller(userRoot, projectRoot, labels)
	l := NewLister(userRoot, projectRoot, labels)
	inst := seedInstall(t, i, inventory.ScopeUser, "skills/orphan/SKILL.md")
	// Remove only the artifact file and keep the sidecar.
	if err := os.Remove(filepath.Join(userRoot, "skills", "orphan", "SKILL.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	got, err := l.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	for _, x := range got {
		if x.ID == inst.ID {
			t.Errorf("List() returned orphaned installation (%v), want excluded", x.ID)
		}
	}
}

func TestLister_List_ProjectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTempRootsUserOnly(t)
	l := NewLister(userRoot, "", labels)
	_, err := l.List(context.Background(), inventory.ScopeProject)
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("List() err = %v, want ErrProjectRootNotConfigured", err)
	}
}
