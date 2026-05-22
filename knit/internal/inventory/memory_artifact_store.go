package inventory

import (
	"context"
	"io/fs"
	"sync"
)

// MemoryArtifactStore is an in-memory ArtifactStore for use in unit tests.
// It does not model directory entries explicitly; PruneAncestorsWithin is a
// no-op because the in-memory model has no notion of empty directories.
//
// Files are keyed by their AbsoluteArtifactPath string. Operations are safe
// for concurrent use.
type MemoryArtifactStore struct {
	mu    sync.Mutex
	files map[string]memFile
}

type memFile struct {
	content []byte
	mode    fs.FileMode
}

// NewMemoryArtifactStore returns an empty in-memory ArtifactStore.
func NewMemoryArtifactStore() *MemoryArtifactStore {
	return &MemoryArtifactStore{files: make(map[string]memFile)}
}

// Exists reports whether p has been written.
func (s *MemoryArtifactStore) Exists(_ context.Context, p AbsoluteArtifactPath) (bool, error) {
	if p.IsZero() {
		return false, ErrArtifactPathEscape
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.files[p.String()]
	return ok, nil
}

// Write stores content under p.
func (s *MemoryArtifactStore) Write(_ context.Context, p AbsoluteArtifactPath, content []byte, mode fs.FileMode) error {
	if p.IsZero() {
		return ErrArtifactPathEscape
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]byte, len(content))
	copy(copied, content)
	s.files[p.String()] = memFile{content: copied, mode: mode}
	return nil
}

// Remove deletes p. Missing entries are treated as success.
func (s *MemoryArtifactStore) Remove(_ context.Context, p AbsoluteArtifactPath) error {
	if p.IsZero() {
		return ErrArtifactPathEscape
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, p.String())
	return nil
}

// PruneAncestorsWithin is a no-op for the in-memory backend, except that it
// enforces the boundary contract (returns ErrPruneBoundaryViolation when
// child does not lie within boundary).
func (s *MemoryArtifactStore) PruneAncestorsWithin(_ context.Context, child AbsoluteArtifactPath, boundary InventoryRoot) error {
	if child.IsZero() || boundary.IsZero() {
		return ErrPruneBoundaryViolation
	}
	if !isWithin(boundary.String(), child.String()) {
		return ErrPruneBoundaryViolation
	}
	return nil
}

// Static type check.
var _ ArtifactStore = (*MemoryArtifactStore)(nil)
