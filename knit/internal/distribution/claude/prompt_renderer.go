package claude

import (
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// promptRenderer produces commands/<name>.md from a KindPrompt entry.
// Claude commands do not carry frontmatter, so the body is emitted as-is
// (with a trailing newline) and tools.claude.frontmatter is rejected.
type promptRenderer struct{}

// Kind returns source.KindPrompt.
func (promptRenderer) Kind() source.Kind { return source.KindPrompt }

// Render returns the prompt artifact or ErrFrontmatterMergeConflict when
// the entry asks for frontmatter that the prompt format cannot represent.
func (promptRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
	if e.HasFrontmatterFor(Target) {
		return source.Artifact{}, fmt.Errorf("%w: kind=prompt entry=%s", ErrFrontmatterMergeConflict, e.ID)
	}
	return source.Artifact{
		Target:         Target,
		Path:           "commands/" + e.Name + ".md",
		Content:        source.EnsureTrailingNewline(e.Body),
		SourceEntryIDs: []string{e.ID},
	}, nil
}
