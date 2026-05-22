package source

// SourceRef identifies where a Pack was loaded from for future refreshes.
//
// It is intentionally small and serializable because inventory labels persist
// it as provenance. The CLI owns interpretation of Kind/Value; builders and
// distribution packages should treat it as trace metadata.
type SourceRef struct {
	Kind  SourceRefKind
	Value string
}

// SourceRefKind identifies the source-reference namespace.
type SourceRefKind string

const (
	// SourceRefLocalPath means Value is an absolute path to a local pack
	// directory.
	SourceRefLocalPath SourceRefKind = "local-path"

	// SourceRefRemoteURL means Value is a canonical remote locator accepted by
	// the CLI, for example "github.com/owner/repo/path/to/pack".
	SourceRefRemoteURL SourceRefKind = "remote-url"
)

// IsZero reports whether no source reference is present.
func (r SourceRef) IsZero() bool {
	return r.Kind == "" && r.Value == ""
}
