package gemini

import (
	"bytes"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type ruleAggregator struct{}

// Aggregate concatenates KindRule entries into a single GEMINI.md with no
// frontmatter (GEMINI.md does not interpret frontmatter). Returns
// ErrFrontmatterMergeConflict when any rule entry declares
// tools.gemini.frontmatter.
func (ruleAggregator) Aggregate(entries []*source.Entry, pack *source.Pack) (source.Artifact, error) {
	for _, e := range entries {
		if e.HasFrontmatterFor(Target) {
			return source.Artifact{}, fmt.Errorf("%w: rule %q cannot accept frontmatter (GEMINI.md has no frontmatter)", ErrFrontmatterMergeConflict, e.ID)
		}
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", pack.Name)
	ids := make([]string, 0, len(entries))
	for i, e := range entries {
		fmt.Fprintf(&buf, "## %s\n\n", e.Name)
		buf.Write(e.Body)
		if !bytes.HasSuffix(e.Body, []byte("\n")) {
			buf.WriteByte('\n')
		}
		if i < len(entries)-1 {
			buf.WriteByte('\n')
		}
		ids = append(ids, e.ID)
	}
	return source.Artifact{
		Target:         Target,
		Path:           "GEMINI.md",
		Content:        buf.Bytes(),
		SourceEntryIDs: ids,
	}, nil
}
