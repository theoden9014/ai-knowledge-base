package cli

import (
	"path/filepath"
	"strings"
)

// FsPath is a path inside the fs.FS hierarchy rooted at Runtime.Fsys.
// fs.FS demands paths that do not begin with "/" and uses "." for the
// root; FsPath wraps those conventions so callers can join, walk to a
// parent, and round-trip to absolute paths without re-deriving the
// "/-vs-." handling in every call site.
//
// The zero value is the fs root (".").
type FsPath struct {
	// value is normalized to use forward slashes and never begins with
	// "/". The fs root is stored as ".".
	value string
}

// FsPathFromAbs converts an absolute filesystem path into an FsPath. The
// filesystem root ("/") maps to FsPath{"."}. Relative inputs are
// accepted (treated as already-fs paths) so tests can pass them
// directly. The empty string maps to the fs root.
func FsPathFromAbs(p string) FsPath {
	slash := filepath.ToSlash(p)
	if slash == "" {
		return FsPath{value: "."}
	}
	if !strings.HasPrefix(slash, "/") {
		return FsPath{value: slash}
	}
	trimmed := strings.TrimPrefix(slash, "/")
	if trimmed == "" {
		return FsPath{value: "."}
	}
	return FsPath{value: trimmed}
}

// String returns the fs.FS-style representation ("." for the root,
// "foo/bar" otherwise).
func (p FsPath) String() string {
	if p.value == "" {
		return "."
	}
	return p.value
}

// Abs returns the absolute filesystem path equivalent. The fs root maps
// to "/"; every other path receives a leading "/".
func (p FsPath) Abs() string {
	if p.value == "" || p.value == "." {
		return "/"
	}
	return "/" + p.value
}

// IsRoot reports whether p is the fs root.
func (p FsPath) IsRoot() bool { return p.value == "" || p.value == "." }

// Join appends name as a child segment. It avoids the "./" leading
// prefix that path.Join would emit when joining onto the root.
func (p FsPath) Join(name string) FsPath {
	if p.IsRoot() {
		return FsPath{value: name}
	}
	return FsPath{value: p.value + "/" + name}
}

// Parent returns the parent directory. If p is already the root, the
// second return is false; callers use that to terminate upward walks.
func (p FsPath) Parent() (FsPath, bool) {
	if p.IsRoot() {
		return p, false
	}
	idx := strings.LastIndexByte(p.value, '/')
	if idx < 0 {
		return FsPath{value: "."}, true
	}
	return FsPath{value: p.value[:idx]}, true
}
