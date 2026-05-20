// Package inventory provides the distribution-target-side framework for knit.
//
// It defines target-neutral data types (Installation, Label, Provenance,
// Scope) and operation interfaces (Installer, Uninstaller, Lister) for
// working with Installations placed on a distribution target (Target).
//
// Implementations specific to individual targets (claude, codex, gemini,
// etc.) live under internal/distribution/<target> and implement the
// interfaces defined by this package. Each implementation is dedicated to a
// single Target and exposes the target it owns via the Target() method.
//
// This package does not own its own state file or database. The Inventory
// itself (the filesystem) is the source of truth, and Installations are
// identified by metadata labels called Label. The concrete way a Label is
// attached to the underlying entity (extended attributes, marker files, etc.)
// is intentionally left unspecified here and delegated to the distribution
// implementation.
//
// # Scope handling
//
// Scope is taken explicitly as an argument by Installer and Lister.
// Uninstaller accepts an already identified Installation, so it treats
// Installation.Label.Scope as authoritative. This is a deliberate design
// choice that separates how Scope is obtained for operations on not-yet-placed
// targets versus operations on already-placed targets.
//
// # Error contract
//
// The sentinel errors defined by this package (ErrInvalidScope /
// ErrTargetMismatch / ErrAlreadyInstalled / ErrInstallationNotFound) are
// centralized in errors.go. Distribution implementations may wrap them, but
// whenever an error condition is specified by contract, the corresponding
// sentinel must still be detectable via errors.Is.
package inventory
