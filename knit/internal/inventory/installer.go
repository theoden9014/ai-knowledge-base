package inventory

import (
	"context"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Installer is responsible for placing an Artifact into Inventory and adding a
// Label that marks it as managed by knit.
//
// Each Installer implementation is dedicated to a single Target and exposes
// the target it owns via Target(). If multiple Targets must be handled
// together, the CLI side composes multiple Installers.
//
// Implementations live under distribution/<target> and encapsulate
// Target-specific path resolution (the placement destination for each Scope)
// and the Label persistence mechanism.
type Installer interface {
	// Target returns the distribution target handled by this Installer.
	Target() source.Target

	// Install places the given Artifact into the Inventory for the specified
	// Scope and returns the resulting Installation.
	//
	// Contract:
	//   - If artifact.Target does not match Installer.Target(),
	//     return ErrTargetMismatch.
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - If an Installation with the same Label (Target, Scope) and the same
	//     placement destination already exists in Inventory, return
	//     ErrAlreadyInstalled. A caller that needs to reinstall must first
	//     remove the existing Installation through Uninstaller and then call
	//     Install again.
	//   - The returned Installation must point to a real entity that exists in
	//     Inventory, and its Label must be non-zero.
	Install(ctx context.Context, scope Scope, artifact source.Artifact) (Installation, error)
}
