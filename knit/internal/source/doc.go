// Package source defines the neutral data model and source-side interfaces
// that knit uses to read knowledge packs and convert them into target-specific
// artifacts.
//
// The package owns the framework types (Pack, Entry, Kind, Target, Artifact)
// and the source-side roles (Loader, Validator, Builder). It is intentionally
// target-agnostic: target-specific implementations of Builder live under
// internal/distribution/<target>.
package source
