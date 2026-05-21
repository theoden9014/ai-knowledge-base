package source

import (
	"errors"
	"path"
	"strings"
)

// ErrInvalidArtifactPath is returned when a string is rejected as an
// ArtifactPath because it violates one of the structural invariants
// (empty, absolute, contains "..", contains NUL, or contains a backslash).
var ErrInvalidArtifactPath = errors.New("invalid artifact path")

// ArtifactPath is a relative path of an Artifact under an Inventory root.
//
// Invariants enforced by NewArtifactPath:
//   - non-empty
//   - not absolute (does not start with "/")
//   - does not contain ".." as a path segment
//   - does not contain NUL or backslash
//
// The zero value is treated as "no path" (IsZero reports true).
type ArtifactPath struct {
	value string
}

// NewArtifactPath constructs an ArtifactPath after validating its invariants.
// The input is rejected in the order: empty -> dangerous bytes -> absolute -> "..".
func NewArtifactPath(s string) (ArtifactPath, error) {
	if s == "" {
		return ArtifactPath{}, ErrInvalidArtifactPath
	}
	if strings.ContainsAny(s, "\x00\\") {
		return ArtifactPath{}, ErrInvalidArtifactPath
	}
	if path.IsAbs(s) {
		return ArtifactPath{}, ErrInvalidArtifactPath
	}
	for seg := range strings.SplitSeq(s, "/") {
		if seg == ".." {
			return ArtifactPath{}, ErrInvalidArtifactPath
		}
	}
	return ArtifactPath{value: s}, nil
}

// String returns the underlying path string. Returns "" for the zero value.
func (p ArtifactPath) String() string { return p.value }

// IsZero reports whether p is the zero value.
func (p ArtifactPath) IsZero() bool { return p.value == "" }

// Equal reports structural equality.
func (p ArtifactPath) Equal(other ArtifactPath) bool { return p.value == other.value }

// TopSegment returns the first path segment (the part before the first "/").
// Returns "" for the zero value.
func (p ArtifactPath) TopSegment() string {
	if p.value == "" {
		return ""
	}
	if i := strings.IndexByte(p.value, '/'); i >= 0 {
		return p.value[:i]
	}
	return p.value
}
