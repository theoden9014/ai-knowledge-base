package cli

import (
	"path/filepath"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// ArgKind classifies the shape of the <pack-or-path-or-url> positional
// argument accepted by install / update / build.
//
// The triage layer in [TriageArg] only assigns a Kind based on the
// argument's syntactic shape. Loading, validation, and remote dispatch
// are performed afterwards by [loadPackFromArg].
type ArgKind int

const (
	// ArgKindPackName means the argument is a kebab-case local pack name
	// to be resolved against the auto-detected knowledge/ directory.
	// Example: "structure-behavior-design".
	ArgKindPackName ArgKind = iota

	// ArgKindLocalPath means the argument is a local filesystem path
	// (absolute or relative) pointing directly at a pack directory that
	// contains manifest.yaml.
	// Examples:
	//   "/abs/path/to/pack"
	//   "./knowledge/structure-behavior-design"
	//   "../packs/my-pack"
	//   "."  ".."
	//   "knowledge/structure-behavior-design"  (contains "/")
	ArgKindLocalPath

	// ArgKindRemoteURL means the argument is a remote git reference
	// (host-like first segment + "/owner/repo[/subpath]"). The http(s)://
	// scheme prefix, if present, has been stripped by [TriageArg] before
	// the classification.
	// Example: "github.com/theoden9014/ai-knowledge-base/knowledge/pack".
	ArgKindRemoteURL
)

// String returns a stable kebab-case label for the Kind. Used only for
// error messages and debug output.
func (k ArgKind) String() string {
	switch k {
	case ArgKindPackName:
		return "pack-name"
	case ArgKindLocalPath:
		return "local-path"
	case ArgKindRemoteURL:
		return "remote-url"
	default:
		return "unknown"
	}
}

// TriagedArg is the output of [TriageArg]: a classified argument plus a
// "Cleaned" form ready for the corresponding loader.
//
// Contract:
//   - For ArgKindRemoteURL, Cleaned is the input with any http(s)://
//     prefix stripped. It still needs [remote.Parse] before fetch.
//   - For ArgKindLocalPath, Cleaned is the result of [filepath.Clean]
//     applied to the original input. Absolute-path normalization (via
//     [filepath.Abs]) is the responsibility of [loadPackFromArg], not of
//     the triage layer.
//   - For ArgKindPackName, Cleaned equals the input verbatim.
type TriagedArg struct {
	Kind    ArgKind
	Cleaned string
}

// TriageArg classifies arg into one of the three [ArgKind] forms. The
// algorithm is purely syntactic and does no I/O.
//
// Order of checks:
//  1. Strip an optional "http://" / "https://" prefix. The stripped
//     form is what the rest of the rules operate on.
//  2. Path-like prefixes ("/", "./", "../", "." / "..") -> LocalPath.
//     These shapes are unambiguous filesystem paths and must not be
//     re-interpreted as host-like even if they happen to contain "."s.
//  3. Host-like first segment (a "<word>.<word>(.<word>)*" label
//     followed by "/"). This is delegated to [remote.IsRemoteArg] so
//     the host-shape definition stays in one place.
//     -> RemoteURL.
//  4. Any remaining argument that contains "/" -> LocalPath. This
//     catches implicit relative paths like "knowledge/pack" once URLs
//     have been ruled out above.
//  5. Otherwise -> PackName.
//
// Ambiguous shapes that contain a "." but no "/" (for example
// "foo.bar") are intentionally classified as ArgKindPackName here, and
// it is [loadPackFromArg]'s responsibility to reject them with
// ErrUsage. The triage layer does not produce errors of its own —
// every input maps to exactly one Kind.
func TriageArg(arg string) TriagedArg {
	stripped := stripURLScheme(arg)
	if stripped == "" {
		return TriagedArg{Kind: ArgKindPackName, Cleaned: stripped}
	}

	// Path-like prefixes always win over the host-shape check so that
	// "./..." and "../..." cannot be mistaken for hosts.
	switch {
	case stripped == "." || stripped == "..":
		return TriagedArg{Kind: ArgKindLocalPath, Cleaned: filepath.Clean(stripped)}
	case strings.HasPrefix(stripped, "/"),
		strings.HasPrefix(stripped, "./"),
		strings.HasPrefix(stripped, "../"):
		return TriagedArg{Kind: ArgKindLocalPath, Cleaned: filepath.Clean(stripped)}
	}

	if remote.IsRemoteArg(stripped) {
		return TriagedArg{Kind: ArgKindRemoteURL, Cleaned: stripped}
	}

	if strings.Contains(stripped, "/") {
		return TriagedArg{Kind: ArgKindLocalPath, Cleaned: filepath.Clean(stripped)}
	}

	return TriagedArg{Kind: ArgKindPackName, Cleaned: stripped}
}
