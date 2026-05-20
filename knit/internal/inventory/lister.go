package inventory

import (
	"context"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Lister is responsible for enumerating labeled Installations in Inventory.
//
// Each Lister implementation is dedicated to a single Target and exposes the
// target it owns via Target(). If Installations must be enumerated across
// multiple Targets, the CLI side composes the results from multiple Listers.
//
// Implementations live under distribution/<target> and encapsulate the
// Target-specific search scope and Label detection strategy. Entities that do
// not carry a Label must not be included in the result.
type Lister interface {
	// Target returns the distribution target handled by this Lister.
	Target() source.Target

	// List enumerates labeled Installations under the specified Scope.
	//
	// Contract:
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - Every returned Installation must have a non-zero Label.
	//   - Installation.Artifact in the return value may be constructed from
	//     entity metadata only, and the Artifact body itself may not have been
	//     loaded.
	List(ctx context.Context, scope Scope) ([]Installation, error)
}
