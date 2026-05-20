package inventorytest_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory/inventorytest"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

const fakeTarget source.Target = "__fake__"

// fakeInventory implements the minimal in-memory Installer / Uninstaller /
// Lister used by the contract harness self-tests.
type fakeInventory struct {
	// items maps (Scope, Path) to Installation.
	items map[string]inventory.Installation
}

func newFakeInventory() *fakeInventory {
	return &fakeInventory{items: map[string]inventory.Installation{}}
}

func (f *fakeInventory) key(scope inventory.Scope, path string) string {
	return string(scope) + "::" + path
}

func (f *fakeInventory) Target() source.Target { return fakeTarget }

func (f *fakeInventory) Install(_ context.Context, scope inventory.Scope, artifact source.Artifact) (inventory.Installation, error) {
	if artifact.Target != fakeTarget {
		return inventory.Installation{}, fmt.Errorf("fake: %w", inventory.ErrTargetMismatch)
	}
	if err := scope.Validate(); err != nil {
		return inventory.Installation{}, fmt.Errorf("fake: %w", err)
	}
	k := f.key(scope, artifact.Path)
	if _, ok := f.items[k]; ok {
		return inventory.Installation{}, fmt.Errorf("fake: %w", inventory.ErrAlreadyInstalled)
	}
	inst := inventory.Installation{
		ID:    inventory.InstallationID(k),
		Label: inventory.Label{Target: fakeTarget, Scope: scope},
		Provenance: inventory.Provenance{
			SourceEntryIDs: artifact.SourceEntryIDs,
		},
		Artifact: artifact,
	}
	f.items[k] = inst
	return inst, nil
}

func (f *fakeInventory) Uninstall(_ context.Context, inst inventory.Installation) error {
	if inst.Label.Target != fakeTarget {
		return fmt.Errorf("fake: %w", inventory.ErrTargetMismatch)
	}
	if err := inst.Label.Scope.Validate(); err != nil {
		return fmt.Errorf("fake: %w", err)
	}
	k := string(inst.ID)
	if _, ok := f.items[k]; !ok {
		return fmt.Errorf("fake: %w", inventory.ErrInstallationNotFound)
	}
	delete(f.items, k)
	return nil
}

func (f *fakeInventory) List(_ context.Context, scope inventory.Scope) ([]inventory.Installation, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("fake: %w", err)
	}
	out := make([]inventory.Installation, 0)
	for _, inst := range f.items {
		if inst.Label.Scope == scope {
			out = append(out, inst)
		}
	}
	return out, nil
}

func sampleArtifact(path string) source.Artifact {
	return source.Artifact{
		Target:         fakeTarget,
		Path:           path,
		Content:        []byte("# fake"),
		SourceEntryIDs: []string{"fake-pack.skill.fake-entry"},
	}
}

func TestRunInstallerContract_Passes(t *testing.T) {
	inventorytest.RunInstallerContract(t, func(_ *testing.T) inventorytest.InstallerHarness {
		f := newFakeInventory()
		return inventorytest.InstallerHarness{
			Installer:       f,
			SupportedTarget: fakeTarget,
			SampleArtifact:  sampleArtifact("skills/example.md"),
			Uninstaller:     f,
		}
	})
}

func TestRunUninstallerContract_Passes(t *testing.T) {
	inventorytest.RunUninstallerContract(t, func(t *testing.T) inventorytest.UninstallerHarness {
		f := newFakeInventory()
		return inventorytest.UninstallerHarness{
			Uninstaller:     f,
			SupportedTarget: fakeTarget,
			SeedInstalled: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				t.Helper()
				inst, err := f.Install(context.Background(), scope, sampleArtifact("skills/seed.md"))
				if err != nil {
					t.Fatalf("seed Install error: %v", err)
				}
				return inst
			},
			FabricateMissing: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				t.Helper()
				return inventory.Installation{
					ID:    inventory.InstallationID("__missing_key__"),
					Label: inventory.Label{Target: fakeTarget, Scope: scope},
				}
			},
		}
	})
}

func TestRunListerContract_Passes(t *testing.T) {
	inventorytest.RunListerContract(t, func(t *testing.T) inventorytest.ListerHarness {
		f := newFakeInventory()
		return inventorytest.ListerHarness{
			Lister:          f,
			SupportedTarget: fakeTarget,
			SeedInstalled: func(t *testing.T, scope inventory.Scope, n int) []inventory.Installation {
				t.Helper()
				out := make([]inventory.Installation, 0, n)
				for i := 0; i < n; i++ {
					inst, err := f.Install(context.Background(), scope, sampleArtifact(fmt.Sprintf("skills/seed-%d.md", i)))
					if err != nil {
						t.Fatalf("seed Install %d error: %v", i, err)
					}
					out = append(out, inst)
				}
				return out
			},
		}
	})
}

// Ensure the sentinel errors exposed by the contract harness are
// still detectable through wrap (regression guard for the contract).
func TestContract_WrappedSentinelDetectable(t *testing.T) {
	wrapped := fmt.Errorf("fake: %w", inventory.ErrTargetMismatch)
	if !errors.Is(wrapped, inventory.ErrTargetMismatch) {
		t.Errorf("errors.Is(wrapped, ErrTargetMismatch) = false, want true")
	}
}
