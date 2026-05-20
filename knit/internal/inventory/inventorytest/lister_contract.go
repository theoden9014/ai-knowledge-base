package inventorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// ListerFactory creates a Lister for contract testing.
type ListerFactory func(t *testing.T) ListerHarness

// ListerHarness provides the Lister plus the auxiliary information needed to
// drive contract tests.
type ListerHarness struct {
	// Lister is the implementation under test.
	Lister inventory.Lister

	// SupportedTarget is the Target handled by Lister.
	SupportedTarget source.Target

	// SeedInstalled prepares entities already placed in Inventory so the Lister
	// can discover them during contract tests.
	// The returned Installations are the ones that List() is expected to
	// reconstruct.
	SeedInstalled func(t *testing.T, scope inventory.Scope, n int) []inventory.Installation
}

// RunListerContract verifies the full Lister contract.
func RunListerContract(t *testing.T, factory ListerFactory) {
	t.Helper()
	t.Run("Target/matches harness SupportedTarget", func(t *testing.T) {
		h := factory(t)
		if got, want := h.Lister.Target(), h.SupportedTarget; got != want {
			t.Errorf("Lister.Target() = %q, want %q", got, want)
		}
	})

	t.Run("List/returns ErrInvalidScope for invalid scope", func(t *testing.T) {
		h := factory(t)
		_, err := h.Lister.List(context.Background(), inventory.Scope("__nonexistent_scope__"))
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("List() with invalid Scope: err = %v, want errors.Is(err, ErrInvalidScope)", err)
		}
	})

	t.Run("List/returns empty slice when nothing is installed", func(t *testing.T) {
		h := factory(t)
		got, err := h.Lister.List(context.Background(), inventory.ScopeUser)
		if err != nil {
			t.Fatalf("List() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("List() returned %d items on empty Inventory, want 0", len(got))
		}
	})

	t.Run("List/returns all seeded installations with non-zero Label and matching Target", func(t *testing.T) {
		h := factory(t)
		seeded := h.SeedInstalled(t, inventory.ScopeUser, 3)
		got, err := h.Lister.List(context.Background(), inventory.ScopeUser)
		if err != nil {
			t.Fatalf("List() error = %v, want nil", err)
		}
		if len(got) != len(seeded) {
			t.Errorf("List() returned %d items, want %d", len(got), len(seeded))
		}
		for i, inst := range got {
			if inst.Label.IsZero() {
				t.Errorf("List()[%d] has zero Label, want non-zero", i)
			}
			if inst.Label.Target != h.SupportedTarget {
				t.Errorf("List()[%d].Label.Target = %q, want %q", i, inst.Label.Target, h.SupportedTarget)
			}
			if inst.Label.Scope != inventory.ScopeUser {
				t.Errorf("List()[%d].Label.Scope = %q, want %q", i, inst.Label.Scope, inventory.ScopeUser)
			}
		}
	})

	t.Run("List/does not include other Scope's installations", func(t *testing.T) {
		h := factory(t)
		_ = h.SeedInstalled(t, inventory.ScopeProject, 2)
		got, err := h.Lister.List(context.Background(), inventory.ScopeUser)
		if err != nil {
			t.Fatalf("List() error = %v, want nil", err)
		}
		for i, inst := range got {
			if inst.Label.Scope != inventory.ScopeUser {
				t.Errorf("List(ScopeUser)[%d].Label.Scope = %q, want %q (must exclude other scopes)", i, inst.Label.Scope, inventory.ScopeUser)
			}
		}
	})

}
