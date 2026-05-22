package codex

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// frontmatterRenderer is the shared MarkdownFrontmatter configuration
// used by Codex's skill and prompt renderers. Codex inserts a blank line
// between the closing "---" and the body, unlike Claude/Gemini.
var frontmatterRenderer = source.MarkdownFrontmatter{BlankLineBeforeBody: true}
