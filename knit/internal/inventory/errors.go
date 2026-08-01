package inventory

import "errors"

// ErrInvalidScope is returned when the Scope value matches neither of the
// allowed constants (ScopeUser / ScopeProject).
var ErrInvalidScope = errors.New("inventory: invalid scope")

// ErrTargetMismatch is returned when the Target handled by Installer,
// Uninstaller, or Lister does not match the Artifact.Target or
// Installation.Label.Target passed in.
var ErrTargetMismatch = errors.New("inventory: target mismatch")

// ErrAlreadyInstalled is returned by Installer.Install when an Installation
// with the same Label (Target / Scope) and the same placement destination
// (equivalent to InstallationID) already exists in Inventory and a new
// placement cannot be created. If reinstallation is needed, the caller must
// first remove the existing Installation via Uninstaller and then call Install
// again.
var ErrAlreadyInstalled = errors.New("inventory: already installed")

// ErrInstallationNotFound is returned by Uninstaller.Uninstall when the
// underlying entity corresponding to the given Installation cannot be found in
// Inventory. Callers that need idempotent deletion may tolerate this error
// via errors.Is.
//
// LabelStore.Get and LabelStore.Delete also return this error when the target
// Label is not found, because LabelStore shares the same "absence" concept as
// the persistence layer for Installations.
var ErrInstallationNotFound = errors.New("inventory: installation not found")

// ErrLabelAlreadyExists is returned by LabelStore.Set when a Label record with
// the same (scope, id) already exists at the storage location and an overwrite
// is not performed.
//
// This sentinel is conceptually distinct from ErrAlreadyInstalled:
//   - ErrAlreadyInstalled is returned by Installer's preflight (which
//     branches between ErrAlreadyInstalled and ErrUnmanagedArtifactExists based
//     on whether a sidecar exists) to mean "a knit-originated Installation
//     already exists in Inventory."
//   - ErrLabelAlreadyExists is the LabelStore-layer signal that "the Label
//     record itself already exists." Installer is responsible for translating
//     ErrLabelAlreadyExists from LabelStore.Set into ErrAlreadyInstalled based
//     on its own preflight result.
//
// Keeping these separate ensures that ErrAlreadyInstalled has a single
// responsible producer (Installer) and that LabelStore tests can verify "label
// collision" as an independent contract.
var ErrLabelAlreadyExists = errors.New("inventory: label already exists")

// ErrLabelsRootNotConfigured is returned by every LabelStore operation when
// the storage root for Labels has not been configured, for example when an
// empty labelsRoot is passed to SidecarLabelStore.
//
// This error is distinct from ErrInvalidScope. The Scope value itself may be
// valid; the problem is that the root corresponding to that Scope was not
// provided from the CLI layer. At the LabelStore layer, this plays the same
// role as distribution-side errors such as claude's
// ErrProjectRootNotConfigured.
var ErrLabelsRootNotConfigured = errors.New("inventory: labels root not configured")

// ErrInvalidInventoryRoot is returned when a string is rejected as an
// InventoryRoot because it is empty or not an absolute path.
var ErrInvalidInventoryRoot = errors.New("inventory: invalid inventory root")

// ErrProjectRootNotConfigured is returned by InventoryRoots.For when the
// caller requests the project root but no project root was configured at
// construction time.
var ErrProjectRootNotConfigured = errors.New("inventory: project root not configured")

// ErrArtifactPathEscape is returned when joining an InventoryRoot with an
// ArtifactPath yields a path that resolves outside the root after lexical
// cleaning.
var ErrArtifactPathEscape = errors.New("inventory: artifact path escapes inventory root")

// ErrInvalidInstallationID is returned when constructing an InstallationID
// from inputs that do not satisfy the construction contract (for example, an
// empty ArtifactPath).
var ErrInvalidInstallationID = errors.New("inventory: invalid installation id")

// ErrInstallationIdentityMismatch is returned when persisted LabelData points
// to an artifact path whose derived ID differs from the requested
// InstallationID. No filesystem mutation is performed.
var ErrInstallationIdentityMismatch = errors.New("inventory: installation identity mismatch")

// ErrPruneBoundaryViolation is returned by ArtifactWriter.PruneAncestorsWithin
// when the requested child path does not lie within the boundary.
var ErrPruneBoundaryViolation = errors.New("inventory: prune boundary violation")

// ErrUnmanagedArtifactExists is returned by TransactionalInstaller preflight
// when an artifact already exists at the destination but no knit-managed
// Label is present, indicating that an external (non-knit) file occupies
// the slot. Callers must remove the unmanaged file before installing.
var ErrUnmanagedArtifactExists = errors.New("inventory: unmanaged artifact exists")
