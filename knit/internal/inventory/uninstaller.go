package inventory

import (
	"context"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Uninstaller is responsible for removing labeled Installations from
// Inventory.
//
// Each Uninstaller implementation is dedicated to a single Target and exposes
// the target it owns via Target(). Scope is not passed as an argument;
// Installation.Label.Scope is treated as authoritative instead. Installer and
// Lister accept scope explicitly, but Uninstaller works on an already
// identified entity, so duplicating the scope input is intentionally avoided.
//
// Implementations live under distribution/<target> and encapsulate
// Target-specific path resolution and identifying the removal target by using
// the Label.
type Uninstaller interface {
	// Target returns the distribution target handled by this Uninstaller.
	Target() source.Target

	// Uninstall removes the specified Installation from Inventory.
	//
	// Contract:
	//   - If installation.Label.Target does not match Uninstaller.Target(),
	//     return ErrTargetMismatch.
	//   - If installation.Label.Scope is not an allowed value, return
	//     ErrInvalidScope.
	//   - If the target does not exist in Inventory, return
	//     ErrInstallationNotFound. Callers that need idempotent deletion may
	//     tolerate this error via errors.Is.
	//   - Implementations must not accidentally remove an unlabeled entity
	//     (the input Installation is assumed to have a non-zero Label).
	Uninstall(ctx context.Context, installation Installation) error
}
