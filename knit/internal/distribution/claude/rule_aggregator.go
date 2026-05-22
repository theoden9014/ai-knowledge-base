package claude

import (
	"bytes"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// ruleAggregator folds every KindRule entry in a pack into a single
// CLAUDE.md. Layout:
//   - "# <pack>" header
//   - For each entry: blank line, "## <name>", blank line, body with a
//     trailing newline ensured.
//
// tools.claude.frontmatter is unsupported for rule entries and produces
// ErrFrontmatterMergeConflict.
type ruleAggregator struct{}

// Aggregate concatenates entries in manifest order into CLAUDE.md.
func (ruleAggregator) Aggregate(entries []*source.Entry, pack *source.Pack) (source.Artifact, error) {
	var buf bytes.Buffer
	buf.WriteString("# ")
	buf.WriteString(pack.Name)
	buf.WriteString("\n")
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.HasFrontmatterFor(Target) {
			return source.Artifact{}, fmt.Errorf("%w: kind=rule entry=%s", ErrFrontmatterMergeConflict, e.ID)
		}
		ids = append(ids, e.ID)
		buf.WriteString("\n## ")
		buf.WriteString(e.Name)
		buf.WriteString("\n\n")
		buf.Write(ensureTrailingNewline(e.Body))
	}
	return source.Artifact{
		Target:         Target,
		Path:           "CLAUDE.md",
		Content:        buf.Bytes(),
		SourceEntryIDs: ids,
	}, nil
}
