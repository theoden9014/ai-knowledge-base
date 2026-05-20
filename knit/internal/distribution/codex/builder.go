package codex

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/pelletier/go-toml/v2"
	"sigs.k8s.io/yaml"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Builder is the Codex CLI implementation of source.Builder.
//
// Given a Pack, it produces zero or more source.Artifact values according to
// these rules:
//
//   - source.KindSkill: one `skills/<name>/SKILL.md`
//     (YAML frontmatter containing name / description merged with
//     tools.codex.frontmatter)
//   - source.KindAgent: one `agents/<name>.toml`
//     (TOML output generated from neutral name / description /
//     developer_instructions, with tools.codex.frontmatter merged into the TOML
//     table; uses_skills is ignored in this phase)
//   - source.KindPrompt: one `prompts/<name>.md`
//     (YAML frontmatter containing description merged with
//     tools.codex.frontmatter; prompts/ must stay flat because Codex does not
//     allow subdirectories)
//   - source.KindRule: one `AGENTS.md` built by concatenating all rule entries
//     in manifest order
//     (no frontmatter; the pack name becomes H1 and each entry name becomes H2;
//     if tools.codex.frontmatter is specified, ErrFrontmatterMergeConflict is
//     returned)
//
// The rule concatenation policy, including ordering and heading insertion, is
// intentionally contained within this Builder.
//
// Agent output is the only kind that uses **TOML**. The neutral body is copied
// into `developer_instructions` as a TOML multiline string.
//
// If the neutral body contains the multiline delimiter (`"""`), the contract is
// that Builder absorbs the collision via escaping rather than returning an
// error or introducing a new sentinel error. Callers therefore do not need
// special error handling for delimiter collisions. The implementation delegates
// to `pelletier/go-toml/v2` Marshal and relies on the library to choose the
// required escaping.
//
// Builder is side-effect free and does not touch the filesystem. Entry
// selection always goes through `pack.EntriesFor(codex.Target)`, following the
// convention that per-target and DefaultTools resolution is centralized in one
// place.
type Builder struct{}

// NewBuilder constructs a Builder. The current implementation takes no
// arguments, but the constructor is kept so future options such as templating
// can be added without changing the call pattern.
func NewBuilder() *Builder {
	return &Builder{}
}

// Target returns the distribution target handled by this Builder. It always
// returns [Target].
func (b *Builder) Target() source.Target {
	return Target
}

// Build converts the given Pack into a set of Codex CLI artifacts.
//
// Contract:
//   - Only entries returned by `pack.EntriesFor(codex.Target)` are considered.
//     Entries that are not enabled are excluded.
//   - Every returned artifact satisfies `Artifact.Target == [Target]`.
//   - Artifact.Path is relative to the inventory root, for example
//     "skills/orchestrator/SKILL.md", "agents/solid-reviewer.toml",
//     "prompts/review.md", or "AGENTS.md".
//   - Artifact.SourceEntryIDs contains all Entry.ID values that contributed to
//     that artifact. For AGENTS.md, that means every contributing rule entry ID.
//   - Build is idempotent: the same Pack input produces the same artifact set.
//   - Structurally impossible merge requests, such as tools.codex.frontmatter on
//     kind: rule, return ErrFrontmatterMergeConflict.
func (b *Builder) Build(ctx context.Context, pack *source.Pack) ([]source.Artifact, error) {
	if pack == nil {
		return nil, nil
	}
	entries := pack.EntriesFor(Target)
	if len(entries) == 0 {
		return nil, nil
	}

	var artifacts []source.Artifact
	var ruleEntries []*source.Entry

	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		switch e.Kind {
		case source.KindSkill:
			a, err := buildSkillArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, a)
		case source.KindAgent:
			a, err := buildAgentArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, a)
		case source.KindPrompt:
			a, err := buildPromptArtifact(e)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, a)
		case source.KindRule:
			if cfg, ok := e.Tools[Target]; ok && len(cfg.Frontmatter) > 0 {
				return nil, fmt.Errorf("%w: kind=rule entry=%s does not support per-target frontmatter", ErrFrontmatterMergeConflict, e.ID)
			}
			ruleEntries = append(ruleEntries, e)
		default:
			// Ignore unknown kinds to preserve forward compatibility.
		}
	}

	if len(ruleEntries) > 0 {
		artifacts = append(artifacts, buildRuleArtifact(pack.Name, ruleEntries))
	}

	return artifacts, nil
}

func buildSkillArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	if cfg, ok := e.Tools[Target]; ok {
		mergeFrontmatter(fm, cfg.Frontmatter)
	}
	content, err := writeMarkdownWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           fmt.Sprintf("skills/%s/SKILL.md", e.Name),
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

func buildPromptArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"description": e.Description,
	}
	if cfg, ok := e.Tools[Target]; ok {
		mergeFrontmatter(fm, cfg.Frontmatter)
	}
	content, err := writeMarkdownWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           fmt.Sprintf("prompts/%s.md", e.Name),
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

func buildAgentArtifact(e *source.Entry) (source.Artifact, error) {
	// Reserved fields produced by the neutral transformation.
	table := map[string]any{
		"name":                   e.Name,
		"description":            e.Description,
		"developer_instructions": string(e.Body),
	}
	if cfg, ok := e.Tools[Target]; ok {
		mergeFrontmatter(table, cfg.Frontmatter)
	}
	// pelletier/go-toml v2 sorts map keys, so output ordering is alphabetical,
	// which helps guarantee idempotent output for identical input.
	buf, err := toml.Marshal(table)
	if err != nil {
		return source.Artifact{}, fmt.Errorf("codex: marshal agent toml for %s: %w", e.ID, err)
	}
	return source.Artifact{
		Target:         Target,
		Path:           fmt.Sprintf("agents/%s.toml", e.Name),
		Content:        buf,
		SourceEntryIDs: []string{e.ID},
	}, nil
}

func buildRuleArtifact(packName string, entries []*source.Entry) source.Artifact {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", packName)
	ids := make([]string, 0, len(entries))
	for i, e := range entries {
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
	}
}

// mergeFrontmatter overwrites dst with the keys and values from src. When a key
// exists in both maps, src wins, per the knowledge-format propagation rules for
// tools.<target>.
func mergeFrontmatter(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

// writeMarkdownWithFrontmatter returns YAML frontmatter plus a Markdown body as
// bytes. Keys are emitted in alphabetical order to keep output idempotent.
func writeMarkdownWithFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString("---\n")
	for _, k := range keys {
		// sigs.k8s.io/yaml encodes through JSON, so both strings and scalar values
		// are emitted safely. Writing one single-key map at a time preserves a
		// deterministic order.
		oneKey := map[string]any{k: fm[k]}
		out, err := yaml.Marshal(oneKey)
		if err != nil {
			return nil, fmt.Errorf("codex: marshal frontmatter key %q: %w", k, err)
		}
		// yaml.Marshal appends a trailing newline. Append it directly to avoid
		// introducing double newlines during concatenation.
		buf.Write(out)
	}
	// YAML output already ends with a newline, so there is no duplicate newline
	// before the closing delimiter.
	buf.WriteString("---\n")
	// Do not inject a leading newline into body content. The caller is expected
	// to pass body data that already carries its intended trailing newline.
	if len(body) > 0 {
		// Insert exactly one separator newline between frontmatter and body.
		buf.WriteString("\n")
		buf.Write(body)
		if body[len(body)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}

// Static interface check for early detection of signature drift.
var _ source.Builder = (*Builder)(nil)
