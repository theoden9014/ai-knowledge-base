package inventory

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// Label is metadata used to identify that an Installation was placed by knit.
//
// The responsibility of Label is intentionally limited to identifying that the
// Installation is knit-managed and declaring which Inventory (Target / Scope)
// it belongs to. Traceability such as which Entry it was derived from belongs
// to Installation.Provenance instead, because it changes for different
// reasons.
//
// Label specifies only what it represents, not how it is attached to the
// underlying entity (extended attributes, marker files, adjacent sidecars,
// etc.). The concrete persistence mechanism is left to the distribution
// implementation.
type Label struct {
	// Target stores the distribution target that owns the Installation.
	Target source.Target

	// Scope stores the scope where the Installation was placed.
	Scope Scope
}

// IsZero reports whether Label is unset (the zero value).
// It is used to distinguish unlabeled entities in Lister and similar code.
func (l Label) IsZero() bool {
	return l == Label{}
}

// Matches reports whether l and other identify the same (Target, Scope).
// Transactional operations use it to detect ErrTargetMismatch /
// ErrInvalidScope conditions on caller-supplied Installations.
func (l Label) Matches(other Label) bool {
	return l == other
}
