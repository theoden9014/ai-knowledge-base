package inventory

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// InstallationID uniquely identifies an Installation within Inventory.
// This package treats it as an opaque string; interpretation and construction
// are left to the distribution implementation.
//
// Note: distribution implementations are expected to use something equivalent
// to the placement path as the ID, but that is not a formal requirement.
type InstallationID string

// String returns the string representation of InstallationID.
func (id InstallationID) String() string {
	return string(id)
}

// Provenance is traceability information that records how an Installation was
// produced, including which Entries it was derived from. It is separated from
// Label because its concern differs from identification, and it may vary when
// the Entry folding strategy changes.
type Provenance struct {
	// SourceEntryIDs holds the list of target-neutral Entry IDs
	// (<pack>.<kind>.<entry>) that contributed to the Installation.
	// It is stored as a slice to support cases where a single Artifact folds
	// together multiple Entries, such as merged rules.
	SourceEntryIDs []string
}

// Installation represents a single entity placed in the Inventory for a given
// (Scope, Target). It carries the Artifact contents, a Label indicating that
// it is managed by knit, and Provenance describing its origin.
//
// Installation is a value object describing something that already exists on
// the filesystem. It has no persistence responsibility of its own. Placement
// and removal are handled by Installer and Uninstaller; enumeration is handled
// by Lister.
//
// Invariant: Installation always has a non-zero Label.
// Installer must return only Installations with non-zero Labels from Install,
// and Lister must return only Installations with non-zero Labels from List.
// Uninstaller accepts only Installations with non-zero Labels as input; the
// behavior for a zero-Label Installation is undefined.
type Installation struct {
	// ID is the identifier within Inventory.
	ID InstallationID

	// Label is metadata indicating that this entity is managed by knit.
	// It must always be non-zero.
	Label Label

	// Provenance is trace information describing where this entity came from.
	// During enumeration (Lister), the persistence format may not allow it to
	// be restored, in which case SourceEntryIDs may be empty.
	Provenance Provenance

	// Artifact references the placed intermediate representation.
	// During enumeration (Lister), only entity metadata may be loaded without
	// reading the Artifact body, in which case Artifact may be the zero value.
	Artifact source.Artifact
}
