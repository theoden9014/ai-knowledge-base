package codex

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// Target is the source.Target constant for the distribution handled by this
// package. Its value is the kebab-case string "codex", matching the key used
// in Entry.Tools and the values that appear in Pack.DefaultTools.
//
// The Target method on this package's Builder, Installer, Uninstaller, and
// Lister implementations always returns this value.
const Target source.Target = "codex"
