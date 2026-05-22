package claude

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// hasClaudeFrontmatter reports whether Entry.Tools[claude.Target].Frontmatter
// is non-empty.
func hasClaudeFrontmatter(e *source.Entry) bool {
	cfg, ok := e.Tools[Target]
	if !ok {
		return false
	}
	return len(cfg.Frontmatter) > 0
}

// mergeClaudeFrontmatter merges keys from Tools[claude.Target].Frontmatter
// into fm with overwrite semantics. Frontmatter keys take precedence over
// the neutral-derived keys.
func mergeClaudeFrontmatter(fm map[string]any, e *source.Entry) {
	cfg, ok := e.Tools[Target]
	if !ok {
		return
	}
	for k, v := range cfg.Frontmatter {
		fm[k] = v
	}
}

// renderWithFrontmatter formats fm as YAML frontmatter and prepends it to
// body. Keys are emitted in alphabetical order so output stays
// deterministic. A trailing newline is appended to body when missing.
func renderWithFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	if len(fm) == 0 {
		return ensureTrailingNewline(body), nil
	}
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("---\n")
	for _, k := range keys {
		single := map[string]any{k: fm[k]}
		line, err := yaml.Marshal(single)
		if err != nil {
			return nil, fmt.Errorf("claude: marshal frontmatter key %q: %w", k, err)
		}
		buf.Write(line)
	}
	buf.WriteString("---\n")
	buf.Write(ensureTrailingNewline(body))
	return buf.Bytes(), nil
}

// ensureTrailingNewline appends a trailing newline to body when missing.
func ensureTrailingNewline(body []byte) []byte {
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

// neutralIDToShortName converts "<pack>.skill.<entry>" into "<pack>-<entry>",
// the form Claude Code expects in an agent frontmatter `skills:` array.
// Inputs that do not contain ".skill." are returned unchanged so unchecked
// inputs do not panic.
func neutralIDToShortName(id string) string {
	const sep = ".skill."
	idx := strings.Index(id, sep)
	if idx < 0 {
		return id
	}
	return id[:idx] + "-" + id[idx+len(sep):]
}
