package gemini

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// frontmatterRenderer is the shared MarkdownFrontmatter configuration
// used by Gemini's skill and agent renderers. Gemini does not insert a
// blank line between the closing "---" and the body.
var frontmatterRenderer = source.MarkdownFrontmatter{}
