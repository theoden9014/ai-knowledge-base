package inventory

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// FsArtifactStore is the production ArtifactStore implementation backed by
// the host filesystem via the os package.
type FsArtifactStore struct{}

// NewFsArtifactStore returns a stateless ArtifactStore that reads and writes
// against the host filesystem.
func NewFsArtifactStore() *FsArtifactStore { return &FsArtifactStore{} }

// Exists reports whether the artifact at p exists. Non-existence returns
// (false, nil); other I/O errors are returned verbatim.
func (s *FsArtifactStore) Exists(ctx context.Context, p AbsoluteArtifactPath) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if p.IsZero() {
		return false, ErrArtifactPathEscape
	}
	_, err := os.Stat(p.String())
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Write creates intermediate directories with 0o755 and writes content with
// the requested mode. A zero mode is treated as 0o644.
func (s *FsArtifactStore) Write(ctx context.Context, p AbsoluteArtifactPath, content []byte, mode fs.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.IsZero() {
		return ErrArtifactPathEscape
	}
	if mode == 0 {
		mode = 0o644
	}
	dir := filepath.Dir(p.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.String(), content, mode)
}

// Remove deletes the artifact. Missing files are treated as success.
func (s *FsArtifactStore) Remove(ctx context.Context, p AbsoluteArtifactPath) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.IsZero() {
		return ErrArtifactPathEscape
	}
	err := os.Remove(p.String())
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// PruneAncestorsWithin walks upward from filepath.Dir(child) and removes
// each directory that is empty, stopping strictly before boundary. The
// boundary directory itself is never removed.
func (s *FsArtifactStore) PruneAncestorsWithin(ctx context.Context, child AbsoluteArtifactPath, boundary InventoryRoot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if child.IsZero() || boundary.IsZero() {
		return ErrPruneBoundaryViolation
	}
	boundaryAbs := boundary.String()
	if !isWithin(boundaryAbs, child.String()) {
		return ErrPruneBoundaryViolation
	}
	dir := filepath.Dir(child.String())
	for dir != boundaryAbs {
		if !isWithin(boundaryAbs, dir) {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				dir = filepath.Dir(dir)
				continue
			}
			// Non-empty directories return ENOTEMPTY; treat that as a
			// successful stop, not an error.
			return nil
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

// Static type check.
var _ ArtifactStore = (*FsArtifactStore)(nil)
