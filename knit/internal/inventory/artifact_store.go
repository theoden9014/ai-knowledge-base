package inventory

import (
	"context"
	"io/fs"
)

// ArtifactReader exposes the read-side of an artifact-storage backend.
// TransactionalLister and the preflight phase of TransactionalInstaller
// depend on this interface, never on ArtifactWriter, so list-only paths
// cannot accidentally mutate the filesystem.
type ArtifactReader interface {
	// Exists reports whether the artifact at p exists on the storage
	// backend. A missing file returns (false, nil). I/O errors other than
	// "not exist" are surfaced verbatim.
	Exists(ctx context.Context, p AbsoluteArtifactPath) (bool, error)
}

// ArtifactWriter exposes the write-side of an artifact-storage backend.
// PruneAncestorsWithin receives the InventoryRoot as a boundary parameter
// so the contract that "do not delete anything at or above the root" can
// be enforced structurally rather than by string convention.
type ArtifactWriter interface {
	// Write places the given content at p with the requested mode. Parent
	// directories are created as needed. Existing content is overwritten.
	Write(ctx context.Context, p AbsoluteArtifactPath, content []byte, mode fs.FileMode) error

	// Remove deletes the artifact at p. A missing file is treated as
	// success (idempotent).
	Remove(ctx context.Context, p AbsoluteArtifactPath) error

	// PruneAncestorsWithin removes empty directories starting at the
	// parent of child and walking upward, stopping strictly before
	// boundary. It must never delete boundary itself or anything outside
	// it. Returns ErrPruneBoundaryViolation if child does not lie within
	// boundary.
	PruneAncestorsWithin(ctx context.Context, child AbsoluteArtifactPath, boundary InventoryRoot) error
}

// ArtifactStore is the convenience composition of ArtifactReader and
// ArtifactWriter. Concrete backends typically implement both, but
// consumers should depend on the narrower interface whenever possible
// (Lister depends on ArtifactReader only).
type ArtifactStore interface {
	ArtifactReader
	ArtifactWriter
}
