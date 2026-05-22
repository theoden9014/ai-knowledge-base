package codex

import (
	"bytes"
	"fmt"
	"sort"

	"sigs.k8s.io/yaml"
)

// writeMarkdownWithFrontmatter returns YAML frontmatter plus a Markdown
// body as a single byte slice. Keys are emitted in alphabetical order so
// repeated runs over the same input produce identical bytes.
func writeMarkdownWithFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString("---\n")
	for _, k := range keys {
		oneKey := map[string]any{k: fm[k]}
		out, err := yaml.Marshal(oneKey)
		if err != nil {
			return nil, fmt.Errorf("codex: marshal frontmatter key %q: %w", k, err)
		}
		buf.Write(out)
	}
	buf.WriteString("---\n")
	if len(body) > 0 {
		buf.WriteString("\n")
		buf.Write(body)
		if body[len(body)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}
