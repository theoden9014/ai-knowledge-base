package inventorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// InstallerFactory creates an Installer for contract testing.
// It is expected to construct a fresh Inventory state on each call so tests
// remain isolated, for example by allocating an internal tempdir.
//
// SupportedTarget returns the Target handled by the Installer under test.
// Contract tests use this Target when constructing Artifacts.
type InstallerFactory func(t *testing.T) InstallerHarness

// InstallerHarness provides the Installer plus the auxiliary information
// needed to drive contract tests.
type InstallerHarness struct {
	// Installer is the implementation under test.
	Installer inventory.Installer

	// SupportedTarget is the Target handled by Installer.
	// It must match Installer.Target(), which the contract test verifies.
	SupportedTarget source.Target

	// SampleArtifact is the sample Artifact used for Install calls in contract
	// tests.
	//
	// Preconditions:
	//   - Target must match SupportedTarget.
	//   - The harness provider must guarantee that calling Install twice with
	//     SampleArtifact for the same Scope collides on the same placement
	//     destination (equivalent to InstallationID) and returns
	//     ErrAlreadyInstalled on the second call. The duplicate Install check in
	//     the contract test depends on this.
	//   - SampleArtifact must have a deterministic placement destination. If the
	//     destination varies with time or similar factors, duplicate detection
	//     may become a false positive.
	SampleArtifact source.Artifact

	// Uninstaller is an Uninstaller sharing the same Inventory, used to verify
	// the "reinstall" contract: Uninstall -> Install again.
	//
	// If nil, the reinstall subtest is skipped. Distributions that want to test
	// Installer alone may omit it, but real implementations from Wave2 onward
	// are strongly encouraged to provide it.
	Uninstaller inventory.Uninstaller
}

// RunInstallerContract verifies the full Installer contract.
// It is intended to be called from distribution/<target> tests like this:
//
//	func TestMyInstaller_Contract(t *testing.T) {
//	    inventorytest.RunInstallerContract(t, func(t *testing.T) inventorytest.InstallerHarness {
//	        // Build an Installer / Uninstaller pair isolated for this test.
//	    })
//	}
func RunInstallerContract(t *testing.T, factory InstallerFactory) {
	t.Helper()
	t.Run("Target/matches harness SupportedTarget", func(t *testing.T) {
		h := factory(t)
		if got, want := h.Installer.Target(), h.SupportedTarget; got != want {
			t.Errorf("Installer.Target() = %q, want %q", got, want)
		}
	})

	t.Run("Install/returns ErrTargetMismatch when artifact.Target differs", func(t *testing.T) {
		h := factory(t)
		artifact := h.SampleArtifact
		artifact.Target = source.Target("__nonexistent_target__")
		_, err := h.Installer.Install(context.Background(), inventory.ScopeUser, artifact)
		if !errors.Is(err, inventory.ErrTargetMismatch) {
			t.Errorf("Install() with mismatched Target: err = %v, want errors.Is(err, ErrTargetMismatch)", err)
		}
	})

	t.Run("Install/returns ErrInvalidScope when scope is invalid", func(t *testing.T) {
		h := factory(t)
		_, err := h.Installer.Install(context.Background(), inventory.Scope("__nonexistent_scope__"), h.SampleArtifact)
		if !errors.Is(err, inventory.ErrInvalidScope) {
			t.Errorf("Install() with invalid Scope: err = %v, want errors.Is(err, ErrInvalidScope)", err)
		}
	})

	t.Run("Install/returns ErrAlreadyInstalled on duplicate install of SampleArtifact", func(t *testing.T) {
		// The harness provider must satisfy the precondition that two Install
		// calls with SampleArtifact collide on the same placement destination.
		// See the InstallerHarness.SampleArtifact doc comment.
		h := factory(t)
		ctx := context.Background()
		if _, err := h.Installer.Install(ctx, inventory.ScopeUser, h.SampleArtifact); err != nil {
			t.Fatalf("first Install() error = %v, want nil", err)
		}
		_, err := h.Installer.Install(ctx, inventory.ScopeUser, h.SampleArtifact)
		if !errors.Is(err, inventory.ErrAlreadyInstalled) {
			t.Errorf("second Install() with same Artifact: err = %v, want errors.Is(err, ErrAlreadyInstalled)", err)
		}
	})

	t.Run("Install/success returns Installation with non-zero Label", func(t *testing.T) {
		h := factory(t)
		got, err := h.Installer.Install(context.Background(), inventory.ScopeUser, h.SampleArtifact)
		if err != nil {
			t.Fatalf("Install() error = %v, want nil", err)
		}
		if got.Label.IsZero() {
			t.Errorf("Install() returned Installation with zero Label, want non-zero")
		}
		if got.Label.Target != h.SupportedTarget {
			t.Errorf("Install() Installation.Label.Target = %q, want %q", got.Label.Target, h.SupportedTarget)
		}
		if got.Label.Scope != inventory.ScopeUser {
			t.Errorf("Install() Installation.Label.Scope = %q, want %q", got.Label.Scope, inventory.ScopeUser)
		}
	})

	t.Run("Install/reinstall after Uninstall succeeds (no ErrAlreadyInstalled)", func(t *testing.T) {
		// This verifies the contract from installer.go: callers that need
		// reinstallation must first remove the existing Installation via
		// Uninstaller and then call Install again. Skip if the harness does not
		// provide an Uninstaller.
		h := factory(t)
		if h.Uninstaller == nil {
			t.Skip("InstallerHarness.Uninstaller is nil; skipping reinstall contract")
		}
		ctx := context.Background()
		first, err := h.Installer.Install(ctx, inventory.ScopeUser, h.SampleArtifact)
		if err != nil {
			t.Fatalf("first Install() error = %v, want nil", err)
		}
		if err := h.Uninstaller.Uninstall(ctx, first); err != nil {
			t.Fatalf("Uninstall() error = %v, want nil", err)
		}
		got, err := h.Installer.Install(ctx, inventory.ScopeUser, h.SampleArtifact)
		if err != nil {
			t.Errorf("reinstall Install() after Uninstall: err = %v, want nil", err)
			return
		}
		if got.Label.IsZero() {
			t.Errorf("reinstall returned Installation with zero Label, want non-zero")
		}
	})
}
