package gemini

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"sigs.k8s.io/yaml"
)

// Builder is the Gemini CLI implementation of source.Builder.
//
// Given a Pack, it generates zero or more source.Artifacts according to the
// following rules:
//
//   - source.KindSkill: one skills/<name>/SKILL.md
//     (the YAML frontmatter emits only name / description following the Gemini
//     CLI spec, and keys / values in tools["gemini"].Frontmatter are
//     override-merged with same-name keys taking precedence)
//   - source.KindAgent: one agents/<name>.md
//     (the YAML frontmatter emits name / description, and the body becomes the
//     subagent System Prompt. Gemini CLI-specific subagent fields
//     (kind / tools / mcpServers / model / temperature / max_turns /
//     timeout_mins) may be set via tools["gemini"].Frontmatter. Neutral
//     uses_skills is not reflected in generated output in this release)
//   - source.KindRule: one GEMINI.md containing all rule entries concatenated
//     in manifest order
//     (Gemini CLI does not interpret frontmatter in GEMINI.md, so none is
//     emitted. The pack name becomes H1 and each entry name becomes H2.
//     Neutral description is intentionally not emitted to GEMINI.md in this
//     release)
//   - source.KindPrompt: one commands/<name>.toml
//     (generated in TOML format following the Gemini CLI Custom Commands spec.
//     The required key "prompt" receives the neutral Body, and the optional
//     key "description" receives the neutral description. Keys / values in
//     tools["gemini"].Frontmatter are override-merged as TOML top-level keys
//     with same-name keys taking precedence)
//
// The rule concatenation policy, including ordering and heading insertion, is
// confined to this Builder.
//
// Builder has no side effects and does not touch the filesystem.
// Entry selection is performed via pack.EntriesFor(gemini.Target), following
// the convention that per-target / DefaultTools resolution is centralized in
// one place.
//
// # TOML Encoding Rules for Tools["gemini"].Frontmatter (kind: prompt)
//
// In the neutral format, tools.<target>.frontmatter is map<string, any>
// (knowledge-format.md §tools.<target>). When this Builder generates TOML for
// KindPrompt, it emits TOML values using the following Go-type mapping while
// preserving type information:
//
//   - string                → TOML string
//   - bool                  → TOML bool
//   - int / int64 / uint64  → TOML integer
//   - float64               → TOML float
//   - time.Time             → TOML datetime
//   - []any                 -> TOML array (elements recurse under the same rules)
//   - map[string]any        -> TOML inline table or table (implementation
//     chooses either form; the semantics are equivalent)
//   - nil                   -> omit the key entirely
//
// If a value falls outside the cases above, such as a function value, channel,
// or unsupported struct, ErrUnsupportedFrontmatterValue is returned.
//
// Values received through YAML stay within the supported range above, so
// normal use of the neutral format does not trigger this error.
//
// # Frontmatter Merge Consistency
//
// Keys matching normal neutral fields (name / description, etc.) are allowed
// as overrides, with tools["gemini"].Frontmatter taking precedence. In
// contrast, if frontmatter is specified for a kind that structurally cannot
// have frontmatter, ErrFrontmatterMergeConflict is returned:
//
//   - when a source.KindRule Entry has non-empty Tools["gemini"].Frontmatter
//     (the generated GEMINI.md cannot have frontmatter under the Gemini CLI
//     specification)
type Builder struct{}

// NewBuilder constructs a Builder. This implementation takes no arguments, but
// construction still goes through a constructor so future template settings or
// similar configuration can be introduced without changing the calling style.
func NewBuilder() *Builder {
	return &Builder{}
}

// Target returns the distribution target handled by this Builder. It always
// returns [Target].
func (b *Builder) Target() source.Target {
	return Target
}

// Build converts the given Pack into Gemini CLI Artifacts.
//
// Contract:
//   - Only Entries returned by pack.EntriesFor(gemini.Target) are processed.
//     Entries that are not enabled are excluded from Build.
//   - Every returned Artifact satisfies Artifact.Target == [Target].
//   - Artifact.Path is relative to the Inventory root:
//     "skills/<name>/SKILL.md", "agents/<name>.md", "GEMINI.md",
//     or "commands/<name>.toml".
//   - Artifact.SourceEntryIDs includes every Entry.ID that contributed to that
//     Artifact. For GEMINI.md, which folds multiple Entries together, it
//     contains the IDs of all rule entries.
//   - Build is idempotent, returning the same Artifact list for the same Pack
//     input.
//   - Frontmatter specified for a rule returns ErrFrontmatterMergeConflict.
//   - If prompt Frontmatter contains values that cannot be TOML-encoded,
//     ErrUnsupportedFrontmatterValue is returned.
//   - ctx cancellation is respected.
//   - If there are zero target Entries, it returns an empty slice and nil
//     error.
func (b *Builder) Build(ctx context.Context, pack *source.Pack) ([]source.Artifact, error) {
	entries := pack.EntriesFor(Target)
	out := make([]source.Artifact, 0, len(entries))
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
			out = append(out, a)
		case source.KindAgent:
			a, err := buildAgentArtifact(e)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		case source.KindRule:
			if cfg, ok := e.Tools[Target]; ok && len(cfg.Frontmatter) > 0 {
				return nil, fmt.Errorf("%w: rule %q cannot accept frontmatter (GEMINI.md has no frontmatter)", ErrFrontmatterMergeConflict, e.ID)
			}
			ruleEntries = append(ruleEntries, e)
		case source.KindPrompt:
			a, err := buildPromptArtifact(e)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
	}

	if len(ruleEntries) > 0 {
		out = append(out, buildRuleArtifact(pack.Name, ruleEntries))
	}
	return out, nil
}

// buildSkillArtifact builds a SKILL.md Artifact from a KindSkill Entry.
func buildSkillArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	if cfg, ok := e.Tools[Target]; ok {
		for k, v := range cfg.Frontmatter {
			fm[k] = v
		}
	}
	content, err := composeYAMLFrontmatter(fm, e.Body)
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

// buildAgentArtifact builds an agents/<name>.md Artifact from a KindAgent
// Entry. Neutral uses_skills is not reflected in generated output in this
// release.
func buildAgentArtifact(e *source.Entry) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	if cfg, ok := e.Tools[Target]; ok {
		for k, v := range cfg.Frontmatter {
			fm[k] = v
		}
	}
	content, err := composeYAMLFrontmatter(fm, e.Body)
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

// buildRuleArtifact collapses KindRule Entries into a single GEMINI.md.
// No frontmatter is added; the content is concatenated in the order
// H1=packName, H2=entry name, then body.
func buildRuleArtifact(packName string, entries []*source.Entry) source.Artifact {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", packName)
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
	}
}

// buildPromptArtifact builds a commands/<name>.toml Artifact from a KindPrompt
// Entry. The neutral Body is embedded under the "prompt" key, the neutral
// description under the "description" key, and Tools[Target].Frontmatter is
// override-merged as TOML top-level keys.
func buildPromptArtifact(e *source.Entry) (source.Artifact, error) {
	v := map[string]any{
		"prompt": string(e.Body),
	}
	if e.Description != "" {
		v["description"] = e.Description
	}
	if cfg, ok := e.Tools[Target]; ok {
		for k, val := range cfg.Frontmatter {
			if val == nil {
				delete(v, k)
				continue
			}
			if !isTOMLEncodable(val) {
				return source.Artifact{}, fmt.Errorf("%w: prompt %q: unsupported value type %T at key %q", ErrUnsupportedFrontmatterValue, e.ID, val, k)
			}
			v[k] = val
		}
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return source.Artifact{}, fmt.Errorf("%w: prompt %q: %v", ErrUnsupportedFrontmatterValue, e.ID, err)
	}
	return source.Artifact{
		Target:         Target,
		Path:           "commands/" + e.Name + ".toml",
		Content:        buf.Bytes(),
		SourceEntryIDs: []string{e.ID},
	}, nil
}

// composeYAMLFrontmatter combines frontmatter (YAML) and a Markdown body.
// Output format: "---\n<yaml>---\n<body>".
func composeYAMLFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("yaml.Marshal failed: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlBytes)
	buf.WriteString("---\n")
	buf.Write(body)
	return buf.Bytes(), nil
}

// isTOMLEncodable recursively reports whether val is a Go type that can be
// encoded as TOML. See doc.go "TOML Encoding Rules for
// Tools[\"gemini\"].Frontmatter" for the allowed types.
func isTOMLEncodable(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, time.Time:
		return true
	case []any:
		for _, el := range v {
			if !isTOMLEncodable(el) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, el := range v {
			if !isTOMLEncodable(el) {
				return false
			}
		}
		return true
	}
	// Fall back to reflect for named types and similar cases.
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
