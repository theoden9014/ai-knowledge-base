package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Reinstaller folds the "uninstall the existing installations of a pack
// and reinstall the freshly built artifacts" sequence into a single
// inventory-side type. CLI subcommands can call Reinstall and react to a
// structured ReinstallReport instead of editing this sequence inline,
// which was previously a transaction script living in cli/cmd_update.go.
//
// Rollback policy on Install failure is intentionally "best-effort,
// one-way": once Uninstall removes the prior installations, a subsequent
// Install failure leaves them removed. The caller can re-run the update
// after fixing the cause. Full 2PC is out of scope.
type Reinstaller struct {
	installer   Installer
	uninstaller Uninstaller
	lister      Lister
}

// NewReinstaller validates that the three roles agree on the same Target
// and returns a Reinstaller. Returns ErrTargetMismatch when the targets
// disagree so callers fail fast rather than mid-Reinstall.
func NewReinstaller(installer Installer, uninstaller Uninstaller, lister Lister) (*Reinstaller, error) {
	if installer == nil || uninstaller == nil || lister == nil {
		return nil, errors.New("inventory: reinstaller requires non-nil installer, uninstaller, and lister")
	}
	t := installer.Target()
	if uninstaller.Target() != t || lister.Target() != t {
		return nil, fmt.Errorf("%w: installer=%q uninstaller=%q lister=%q",
			ErrTargetMismatch, t, uninstaller.Target(), lister.Target())
	}
	return &Reinstaller{installer: installer, uninstaller: uninstaller, lister: lister}, nil
}

// Target returns the shared distribution target.
func (r *Reinstaller) Target() source.Target { return r.installer.Target() }

// ReinstallReport captures the outcome of a Reinstall call so callers
// (typically cli/cmd_update) can render appropriate output without
// re-deriving the counts.
type ReinstallReport struct {
	// PriorCount is the number of prior Installations that matched
	// packName and were uninstalled. Zero means no installation of
	// packName existed at this (Scope, Target).
	PriorCount int

	// InstalledCount is the number of artifacts successfully installed
	// after the prior uninstall pass.
	InstalledCount int

	// NoPriorInstallation is true when PriorCount == 0. Surfaced as a
	// field so callers can distinguish "skipped" from "installed 0"
	// without re-checking PriorCount.
	NoPriorInstallation bool
}

// Reinstall removes every Installation belonging to packName under scope
// and then installs each artifact.
//
// Contract:
//   - When no prior Installation matches packName, returns a report with
//     NoPriorInstallation=true and a nil error. The caller can treat this
//     as a no-op (typically: warn the user and skip the target).
//   - Uninstall errors abort the call before any Install runs.
//   - The first Install failure is returned wrapped, and the partial
//     InstalledCount is preserved in the returned report.
//   - artifacts may be empty: prior installations are still removed,
//     and InstalledCount is reported as 0. Callers should surface this
//     to the user explicitly.
func (r *Reinstaller) Reinstall(ctx context.Context, scope Scope, packName string, artifacts []source.Artifact) (ReinstallReport, error) {
	var report ReinstallReport
	insts, err := r.lister.List(ctx, scope)
	if err != nil {
		return report, err
	}
	for _, inst := range insts {
		if !inst.Provenance.BelongsToPack(packName) {
			continue
		}
		report.PriorCount++
		if err := r.uninstaller.Uninstall(ctx, inst); err != nil {
			if errors.Is(err, ErrInstallationNotFound) {
				// Tolerate parallel removal: another process or the
				// caller's earlier action already cleaned this entry.
				continue
			}
			return report, fmt.Errorf("inventory: reinstall uninstall %s: %w", inst.ID, err)
		}
	}
	if report.PriorCount == 0 {
		report.NoPriorInstallation = true
		return report, nil
	}
	for _, art := range artifacts {
		if _, err := r.installer.Install(ctx, scope, art); err != nil {
			return report, fmt.Errorf("inventory: reinstall install %s: %w (re-run to recover)", art.Path, err)
		}
		report.InstalledCount++
	}
	return report, nil
}
