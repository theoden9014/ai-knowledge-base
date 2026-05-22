package claude

import (
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// frontmatterRenderer is the shared MarkdownFrontmatter configuration
// used by Claude's skill / agent / prompt renderers. Claude does not
// insert a blank line between the closing "---" and the body.
var frontmatterRenderer = source.MarkdownFrontmatter{}

// neutralIDToShortName converts "<pack>.skill.<entry>" into
// "<pack>-<entry>", the form Claude Code expects in an agent frontmatter
// `skills:` array. Inputs that do not contain ".skill." are returned
// unchanged so unchecked inputs do not panic.
func neutralIDToShortName(id string) string {
	const sep = ".skill."
	idx := strings.Index(id, sep)
	if idx < 0 {
		return id
	}
	return id[:idx] + "-" + id[idx+len(sep):]
}
