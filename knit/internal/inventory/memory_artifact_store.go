package inventory

import (
	"context"
	"io/fs"
	"strings"
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

// Content returns a copy of the bytes stored at p, or (nil, false) when
// absent. Intended for assertions in tests.
func (s *MemoryArtifactStore) Content(p AbsoluteArtifactPath) ([]byte, bool) {
	if p.IsZero() {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.files[p.String()]
	if !ok {
		return nil, false
	}
	out := make([]byte, len(f.content))
	copy(out, f.content)
	return out, true
}

// PathsUnder returns every stored path that begins with prefix, in
// undefined order. Intended for assertions in tests.
func (s *MemoryArtifactStore) PathsUnder(prefix string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for k := range s.files {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out
}

// Static type check.
var _ ArtifactStore = (*MemoryArtifactStore)(nil)
