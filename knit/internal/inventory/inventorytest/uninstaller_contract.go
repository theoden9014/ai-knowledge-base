package inventorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// UninstallerFactory creates an Uninstaller for contract testing.
type UninstallerFactory func(t *testing.T) UninstallerHarness

// UninstallerHarness provides the Uninstaller plus the auxiliary information
// needed to drive contract tests.
type UninstallerHarness struct {
	// Uninstaller is the implementation under test.
	Uninstaller inventory.Uninstaller

	// SupportedTarget is the Target handled by Uninstaller.
	SupportedTarget source.Target

	// SeedInstalled is a helper that places an Installation into Inventory in
	// advance so the test body can pass it to Uninstall.
	// The test body calls it directly, though pre-seeding inside the factory is
	// also acceptable.
	SeedInstalled func(t *testing.T, scope inventory.Scope) inventory.Installation

	// FabricateMissing is a helper that assembles an Installation which does
	// not actually exist but can be passed to Uninstall.
	// It is used to verify ErrInstallationNotFound.
	FabricateMissing func(t *testing.T, scope inventory.Scope) inventory.Installation
}

// RunUninstallerContract verifies the full Uninstaller contract.
func RunUninstallerContract(t *testing.T, factory UninstallerFactory) {
	t.Helper()
	t.Run("Target/matches harness SupportedTarget", func(t *testing.T) {
		h := factory(t)
		if got, want := h.Uninstaller.Target(), h.SupportedTarget; got != want {
			t.Errorf("Uninstaller.Target() = %q, want %q", got, want)
		}
	})

	t.Run("Uninstall/returns ErrTargetMismatch when Installation.Label.Target differs", func(t *testing.T) {
		h := factory(t)
		inst := h.SeedInstalled(t, inventory.ScopeUser)
		inst.Label.Target = source.Target("__nonexistent_target__")
		err := h.Uninstaller.Uninstall(context.Background(), inst)
		if !errors.Is(err, inventory.ErrTargetMismatch) {
			t.Errorf("Uninstall() with mismatched Label.Target: err = %v, want errors.Is(err, ErrTargetMismatch)", err)
		}
	})

	t.Run("Uninstall/returns ErrInvalidScope when Installation.Label.Scope is invalid", func(t *testing.T) {
		h := factory(t)
		inst := h.SeedInstalled(t, inventory.ScopeUser)
		inst.Label.Scope = inventory.Scope("__nonexistent_scope__")
		err := h.Uninstaller.Uninstall(context.Background(), inst)
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("Uninstall() with invalid Label.Scope: err = %v, want errors.Is(err, ErrInvalidScope)", err)
		}
	})

	t.Run("Uninstall/returns ErrInstallationNotFound when target is missing", func(t *testing.T) {
		h := factory(t)
		inst := h.FabricateMissing(t, inventory.ScopeUser)
		err := h.Uninstaller.Uninstall(context.Background(), inst)
		if !errors.Is(err, inventory.ErrInstallationNotFound) {
			t.Errorf("Uninstall() with missing installation: err = %v, want errors.Is(err, ErrInstallationNotFound)", err)
		}
	})

	t.Run("Uninstall/success returns nil", func(t *testing.T) {
		h := factory(t)
		inst := h.SeedInstalled(t, inventory.ScopeUser)
		if err := h.Uninstaller.Uninstall(context.Background(), inst); err != nil {
			t.Errorf("Uninstall() error = %v, want nil", err)
		}
	})
}
