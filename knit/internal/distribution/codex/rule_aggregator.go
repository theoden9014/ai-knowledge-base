package codex

import (
	"bytes"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type ruleAggregator struct{}

// Aggregate folds every KindRule entry into AGENTS.md. Returns
// ErrFrontmatterMergeConflict when any rule entry declares
// tools.codex.frontmatter (AGENTS.md cannot carry frontmatter).
func (ruleAggregator) Aggregate(entries []*source.Entry, pack *source.Pack) (source.Artifact, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", pack.Name)
	ids := make([]string, 0, len(entries))
	for i, e := range entries {
		if e.HasFrontmatterFor(Target) {
			return source.Artifact{}, fmt.Errorf("%w: kind=rule entry=%s does not support per-target frontmatter", ErrFrontmatterMergeConflict, e.ID)
		}
		if i > 0 {
			buf.WriteString("\n")
		}
		fmt.Fprintf(&buf, "## %s\n\n", e.Name)
		buf.Write(e.Body)
		if len(e.Body) == 0 || e.Body[len(e.Body)-1] != '\n' {
			buf.WriteByte('\n')
		}
		ids = append(ids, e.ID)
	}
	return source.Artifact{
		Target:         Target,
		Path:           "AGENTS.md",
		Content:        buf.Bytes(),
		SourceEntryIDs: ids,
	}, nil
}
