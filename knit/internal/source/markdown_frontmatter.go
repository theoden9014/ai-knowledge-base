package source

import (
	"bytes"
	"fmt"
	"sort"

	"sigs.k8s.io/yaml"
)

// MarkdownFrontmatter renders YAML frontmatter followed by a Markdown
// body. The output shape is always:
//
//	---\n<keys-sorted alphabetically>---\n[optional blank line]<body+\n>
//
// Each frontmatter key is marshaled separately so the alphabetical
// ordering is enforced regardless of Go map iteration order (which the
// underlying yaml library does not sort).
//
// Distribution targets share this renderer; they configure it through
// BlankLineBeforeBody to reflect their per-target convention.
type MarkdownFrontmatter struct {
	// BlankLineBeforeBody inserts a blank line between the closing
	// "---" line and the body. Codex emits the separator; Claude and
	// Gemini do not.
	BlankLineBeforeBody bool
}

// Render returns the assembled bytes. An empty fm returns just the body
// with a trailing newline ensured, so callers can use the same helper
// for kinds that opt out of frontmatter (Claude prompts, etc.).
func (m MarkdownFrontmatter) Render(fm map[string]any, body []byte) ([]byte, error) {
	if len(fm) == 0 {
		return EnsureTrailingNewline(body), nil
	}
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("---\n")
	for _, k := range keys {
		line, err := yaml.Marshal(map[string]any{k: fm[k]})
		if err != nil {
			return nil, fmt.Errorf("source: marshal frontmatter key %q: %w", k, err)
		}
		buf.Write(line)
	}
	buf.WriteString("---\n")
	if m.BlankLineBeforeBody && len(body) > 0 {
		buf.WriteString("\n")
	}
	buf.Write(EnsureTrailingNewline(body))
	return buf.Bytes(), nil
}

// EnsureTrailingNewline appends a trailing newline to body when missing,
// returning an empty-body input as a single newline. Distribution rule
// aggregators and renderers reuse it instead of each defining their own.
func EnsureTrailingNewline(body []byte) []byte {
	if len(body) == 0 {
		return []byte("\n")
	}
	if body[len(body)-1] == '\n' {
		return body
	}
	out := make([]byte, len(body)+1)
	copy(out, body)
	out[len(body)] = '\n'
	return out
}
