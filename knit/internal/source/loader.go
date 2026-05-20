package source

import (
	"context"
	"io/fs"
)

// Loader reads a knowledge pack from a filesystem and returns the loaded
// Pack together with diagnostic provenance (LoadInfo). Implementations must
// be target-agnostic.
//
// Implementations of Loader own a Validator (injected at construction via
// NewLoader) and call it during loading so that JSON Schema violations are
// reported with the rich context (file path, line number) that only the
// loading pass has. Callers must not invoke a Validator separately on the
// same bytes; the contract is that a Pack returned by LoadPack has already
// been schema-validated.
//
// fsys is rooted at the directory containing one or more pack directories
// (typically the repository's knowledge/ directory). packDir is the pack
// directory name (e.g. "structure-behavior-design").
//
// ctx is honored for cancellation: a long-running load (large pack, schema
// validator that touches the network) must abort promptly when ctx is done.
type Loader interface {
	LoadPack(ctx context.Context, fsys fs.FS, packDir string) (*Pack, LoadInfo, error)
}

// LoadInfo carries diagnostic provenance about a completed LoadPack call.
// It exists so that Pack itself stays a pure data model and consumers
// (Builder, CLI) cannot accidentally use a Pack to perform additional I/O.
type LoadInfo struct {
	// PackDir is the pack directory name within the fs.FS supplied to
	// LoadPack. Use only for logging and error messages.
	PackDir string
}

// NewLoader returns a Loader that delegates schema validation to v. The
// Validator is mandatory: there is no constructor that produces a Loader
// without one, which guarantees at the API level that every Pack returned
// by LoadPack has been schema-validated.
func NewLoader(v Validator) Loader {
	return newLoader(v)
}
