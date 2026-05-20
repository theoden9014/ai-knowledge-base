package inventory

import "context"

// LabelData is the target-neutral data representation of the Label payload
// persisted by LabelStore.
//
// Label (Target / Scope) identifies that something is knit-managed, while
// LabelData carries the additional information needed to bind that label to
// the underlying entity: which placement path it points to, which Entries it
// was derived from, and the schema version for forward compatibility. It is
// effectively a target-neutral lift of the current sidecar implementation
// payload (distribution/claude/sidecar.go: sidecarPayload).
//
// LabelData is the payload corresponding to the Label of a specific
// Installation and is used as the input and output of LabelStore.Set/Get.
// Label itself (Target/Scope) is already determined by the LabelStore scope
// (such as labelsRoot passed to NewSidecarLabelStore and the scope argument of
// each operation), so it is not included in LabelData.
//
// Even if the implementation is later replaced with xattr storage, each field
// in this struct should remain granular enough to map directly to target-
// neutral xattr key/value metadata.
type LabelData struct {
	// SchemaVersion is the format version of LabelData.
	// A LabelStore implementation may tolerate unknown versions on read, but on
	// write it should emit the latest version it understands.
	SchemaVersion int

	// ArtifactPath is the path of the Artifact associated with the Label,
	// relative to the Inventory root. LabelStore treats it as an opaque string;
	// validation such as convention checks or path traversal protection belongs
	// to the distribution implementation.
	ArtifactPath string

	// SourceEntryIDs is the list of target-neutral Entry IDs that contributed to
	// the Installation referenced by the Label. It is stored as a slice to
	// preserve the correspondence with Provenance.
	SourceEntryIDs []string
}

// LabelStore abstracts persistence for Labels associated with a (Target,
// Scope).
//
// This interface only covers label storage. Reading and writing the Artifact
// entity itself (the installed file contents) is out of scope. Installer,
// Uninstaller, and Lister delegate label persistence to LabelStore and handle
// the Artifact entity on their own.
//
// Scope:
//   - A single LabelStore instance is scoped to a specific combination of
//     (Target, Scope, labelsRoot). Target is fixed by the implementation
//     constructor, while Scope is passed to each method. Callers that need to
//     span multiple Targets or Scopes should compose multiple LabelStores.
//
// Replaceability:
//   - The default implementation is [SidecarLabelStore], which stores JSON
//     files under the knit root. If it is later replaced by an
//     XattrLabelStore backed by macOS extended attributes or Linux xattr,
//     callers such as Installer can remain unchanged as long as the contract
//     of this interface is preserved.
//
// Concurrency:
//   - Concurrent Set/Delete operations against the same (Target, Scope, ID)
//     are not guaranteed by this interface. Callers are expected to execute
//     them sequentially within a single process, which is sufficient for the
//     knit CLI model of one command per process.
type LabelStore interface {
	// Set stores a new LabelData for the given (scope, id).
	//
	// Contract:
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - If a Label for the same (scope, id) already exists, return
	//     ErrLabelAlreadyExists and do not overwrite it. A caller that needs to
	//     store it again must Delete first and then call Set.
	//
	//     Note: ErrLabelAlreadyExists is distinct from ErrAlreadyInstalled.
	//     Installer is responsible for translating ErrLabelAlreadyExists
	//     observed from LabelStore.Set into ErrAlreadyInstalled together with
	//     its own preflight (which also decides ErrUnmanagedArtifactExists).
	//     This keeps "produces ErrAlreadyInstalled" a single-Installer
	//     responsibility and lets LabelStore tests verify label collision in
	//     isolation.
	//   - Writes must be atomic so that a partially written, corrupted Label is
	//     never left behind. The concrete mechanism is left to the
	//     implementation.
	Set(ctx context.Context, scope Scope, id InstallationID, data LabelData) error

	// Get returns the LabelData for the given (scope, id).
	//
	// Contract:
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - If the target does not exist, return ErrInstallationNotFound.
	//
	// As of Wave5, Label is treated as the single source of truth for an
	// Installation, so label absence is expressed as ErrInstallationNotFound.
	// If a future backend (for example xattr) lets Label presence and
	// Installation presence diverge, a separate ErrLabelNotFound sentinel may
	// be introduced.
	Get(ctx context.Context, scope Scope, id InstallationID) (LabelData, error)

	// Delete removes the LabelData for the given (scope, id).
	//
	// Contract:
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - If the target does not exist, return ErrInstallationNotFound.
	//     Callers that need idempotent deletion may tolerate this error via
	//     errors.Is.
	//
	// As of Wave5, Label is treated as the single source of truth for an
	// Installation, so label absence is expressed as ErrInstallationNotFound.
	// If a future backend (for example xattr) lets Label presence and
	// Installation presence diverge, a separate ErrLabelNotFound sentinel may
	// be introduced.
	Delete(ctx context.Context, scope Scope, id InstallationID) error

	// List enumerates all Labels under scope as (id, data) pairs.
	//
	// Contract:
	//   - If scope is not an allowed value, return ErrInvalidScope.
	//   - If the corresponding storage location does not exist yet, meaning
	//     nothing has been stored, return nil or an empty slice without error.
	//   - The order of the return value must be stable within each
	//     implementation for testability. The exact ordering rule is left to
	//     the implementation (the sidecar implementation uses file name order).
	//   - Corrupted individual Labels may be silently skipped by this interface
	//     for now, until dedicated detection/reporting is introduced.
	List(ctx context.Context, scope Scope) ([]LabelEntry, error)
}

// LabelEntry is the (id, data) pair returned by LabelStore.List.
//
// Label itself (Target/Scope) is omitted because it is already determined by
// the LabelStore scope. The caller (a distribution Lister) combines this entry
// with its own Target and Scope to reconstruct inventory.Installation.
type LabelEntry struct {
	ID   InstallationID
	Data LabelData
}
