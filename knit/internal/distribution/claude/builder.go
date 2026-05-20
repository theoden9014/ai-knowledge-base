package claude

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Builder is the Claude Code implementation of source.Builder.
//
// Given a Pack, it generates zero or more source.Artifacts using the following rules:
//
//   - source.KindSkill: one skills/<name>/SKILL.md
//     (frontmatter is a merge of name / description and tools.claude.frontmatter)
//   - source.KindAgent: one agents/<name>.md
//     (frontmatter is a merge of name / description / skills derived from
//     uses_skills and tools.claude.frontmatter)
//   - source.KindRule: one CLAUDE.md containing all rule entries concatenated in manifest order
//     (no frontmatter. Concatenates the pack name as H1 and each entry name as H2.
//     Appends a trailing newline to each entry body and inserts one blank line
//     between entries. Returns ErrFrontmatterMergeConflict when
//     Tools["claude"].Frontmatter is non-empty)
//   - source.KindPrompt: one commands/<name>.md
//     (no frontmatter, body only. Returns ErrFrontmatterMergeConflict when
//     Tools["claude"].Frontmatter is non-empty)
//
// For outputs with frontmatter (skill / agent), keys in
// Tools["claude"].Frontmatter are merged with overwrite semantics on top of the
// neutral transformation result (Frontmatter values win for duplicate keys).
// Frontmatter keys are emitted in alphabetical order to keep output ordering
// deterministic.
// If Frontmatter is specified for an output that has no frontmatter
// representation (rule / prompt), it is rejected as
// ErrFrontmatterMergeConflict because it cannot be represented structurally
// (matching the sentinel definition in errors.go).
//
// Rule concatenation policy (ordering, heading insertion, blank lines, and
// trailing newlines) is confined to this Builder.
// Newline rules are aligned across distribution/{claude,codex,gemini}: append
// one trailing newline to each entry and insert one blank line between entries.
//
// Builder is side-effect free and does not touch the filesystem.
// Entry selection is performed via pack.EntriesFor(claude.Target), following
// the convention that per-target / DefaultTools resolution is centralized in one place.
type Builder struct{}

// NewBuilder constructs a Builder. This implementation currently takes no
// arguments, but uses a constructor to preserve room for future template or
// configuration parameters.
func NewBuilder() *Builder {
	return &Builder{}
}

// Target returns the distribution target handled by this Builder. It always returns [Target].
func (b *Builder) Target() source.Target {
	return Target
}

// Build converts the given Pack into artifacts for Claude Code.
func (b *Builder) Build(ctx context.Context, pack *source.Pack) ([]source.Artifact, error) {
	entries := pack.EntriesFor(Target)
	var (
		artifacts []source.Artifact
		ruleBuf   []*source.Entry
	)
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		switch e.Kind {
		case source.KindSkill:
			art, err := buildSkillArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, art)
		case source.KindAgent:
			art, err := buildAgentArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, art)
		case source.KindPrompt:
			art, err := buildPromptArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, art)
		case source.KindRule:
			if hasClaudeFrontmatter(e) {
				return nil, fmt.Errorf("%w: kind=rule entry=%s", ErrFrontmatterMergeConflict, e.ID)
			}
			ruleBuf = append(ruleBuf, e)
		}
	}
	if len(ruleBuf) > 0 {
		artifacts = append(artifacts, buildRuleArtifact(pack.Name, ruleBuf))
	}
	return artifacts, nil
}

// hasClaudeFrontmatter reports whether Entry.tools.claude.frontmatter is
// non-empty (contains at least one key).
func hasClaudeFrontmatter(e *source.Entry) bool {
	cfg, ok := e.Tools[Target]
	if !ok {
		return false
	}
	return len(cfg.Frontmatter) > 0
}

// buildSkillArtifact builds skills/<name>/SKILL.md from a KindSkill entry.
func buildSkillArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	mergeClaudeFrontmatter(fm, e)
	content, err := renderWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           "skills/" + e.Name + "/SKILL.md",
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

// buildAgentArtifact builds agents/<name>.md from a KindAgent entry.
// uses_skills is merged as a skills: array by extracting the "<entry>" portion
// from "<pack>.skill.<entry>" (per knowledge-format.md).
func buildAgentArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	if e.Agent != nil && len(e.Agent.UsesSkills) > 0 {
		skills := make([]string, 0, len(e.Agent.UsesSkills))
		for _, id := range e.Agent.UsesSkills {
			skills = append(skills, neutralIDToShortName(id))
		}
		fm["skills"] = skills
	}
	mergeClaudeFrontmatter(fm, e)
	content, err := renderWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           "agents/" + e.Name + ".md",
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

// buildPromptArtifact builds commands/<name>.md from a KindPrompt entry.
// In the current spec, prompts do not have frontmatter.
func buildPromptArtifact(e *source.Entry) (source.Artifact, error) {
	if hasClaudeFrontmatter(e) {
		return source.Artifact{}, fmt.Errorf("%w: kind=prompt entry=%s", ErrFrontmatterMergeConflict, e.ID)
	}
	body := ensureTrailingNewline(e.Body)
	return source.Artifact{
		Target:         Target,
		Path:           "commands/" + e.Name + ".md",
		Content:        body,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

// buildRuleArtifact concatenates KindRule entries into a single CLAUDE.md.
// Formatting rules shared by distribution/{claude,codex,gemini}:
//   - One H1 heading with the pack name at the top
//   - Each entry is emitted under an H2 heading using its name, with one blank
//     line between the H2 and the body
//   - A trailing newline is appended to each entry body
//   - Entries are separated by one blank line between the previous body's
//     trailing newline and the next H2
func buildRuleArtifact(packName string, entries []*source.Entry) source.Artifact {
	var buf bytes.Buffer
	buf.WriteString("# ")
	buf.WriteString(packName)
	buf.WriteString("\n")
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
		buf.WriteString("\n")
		buf.WriteString("## ")
		buf.WriteString(e.Name)
		buf.WriteString("\n\n")
		buf.Write(ensureTrailingNewline(e.Body))
	}
	return source.Artifact{
		Target:         Target,
		Path:           "CLAUDE.md",
		Content:        buf.Bytes(),
		SourceEntryIDs: ids,
	}
}

// ensureTrailingNewline appends a trailing newline to body if it does not already have one.
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

// mergeClaudeFrontmatter merges keys from Tools[claude.Target].Frontmatter into
// fm with overwrite semantics. Frontmatter keys take precedence.
func mergeClaudeFrontmatter(fm map[string]any, e *source.Entry) {
	cfg, ok := e.Tools[Target]
	if !ok {
		return
	}
	for k, v := range cfg.Frontmatter {
		fm[k] = v
	}
}

// renderWithFrontmatter builds YAML frontmatter plus a Markdown body from a
// frontmatter map and body bytes. Frontmatter keys are emitted in alphabetical
// order to keep ordering deterministic.
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

// neutralIDToShortName converts a neutral skill ID ("<pack>.skill.<entry>")
// into the Claude Code-style skills array element ("<pack>-<entry>").
//
// Example: "p.skill.s1" -> "p-s1"
//
//	"structure-behavior-design.skill.solid-responsibility"
//	→ "structure-behavior-design-solid-responsibility"
//
// Fallback: if id does not follow the expected format, return the original
// string unchanged.
// knowledge-format's entry.schema.json enforces the
// "<pack>.skill.<entry>" form for uses_skills elements, so this fallback path
// is unreachable as long as the input Pack has passed through Loader / Validator.
// It remains as a defensive guard so unchecked inputs still do not panic.
func neutralIDToShortName(id string) string {
	const sep = ".skill."
	idx := strings.Index(id, sep)
	if idx < 0 {
		return id
	}
	return id[:idx] + "-" + id[idx+len(sep):]
}

// Builder interface conformance check for early detection of signature changes.
var _ source.Builder = (*Builder)(nil)
