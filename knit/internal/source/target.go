package source

// Target identifies a distribution target (an AI tool such as claude, codex,
// gemini). The value is the kebab-case target name used as a key under
// Entry.Tools and as the manifest's default_tools element.
//
// The set of valid targets is open: each distribution package is expected to
// expose its own Target constant. The source package deliberately does not
// enumerate the known targets so that adding a new target does not require
// modifying this package.
type Target string

// String returns the string form of the Target.
func (t Target) String() string {
	return string(t)
}
