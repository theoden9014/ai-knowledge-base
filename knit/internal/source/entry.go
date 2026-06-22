package source

import "maps"

// Entry is a single knowledge file loaded from a pack. It carries the neutral
// frontmatter fields defined by knowledge-format together with the raw
// markdown body. Target-specific transformations are the Builder's job; Entry
// itself stays neutral.
//
// Entry is treated as immutable input by every consumer (Validator, Builder).
// Methods that need to query enabled-state live on Pack so the resolution
// rule (per-target enabled flag → DefaultTools fallback) is defined in
// exactly one place.
type Entry struct {
	// ID is the neutral identifier in the form <pack>.<kind>.<name>.
	ID string

	// Kind is the entry kind (skill / agent / rule / prompt).
	Kind Kind

	// Name is the identifier that will be passed to the target tool.
	// Conventionally <pack>-<entry-name>.
	Name string

	// Description is the human-readable summary.
	Description string

	// Tags is the optional classification tag list.
	Tags []string

	// Tools is the per-target build directive map keyed by Target.
	Tools map[Target]ToolConfig

	// Agent carries kind-specific metadata that is only meaningful when
	// Kind is KindAgent. For every other Kind this field is nil. Putting
	// agent-only fields behind a typed sub-struct lets callers express
	// "agent metadata is absent" at the type level instead of relying on
	// runtime checks against Kind.
	Agent *AgentMeta

	// Skill carries kind-specific metadata that is only meaningful when
	// Kind is KindSkill. It pairs the skill root (a pack-relative
	// directory) with the sibling assets collected from that directory.
	// For every other Kind this field is nil. Skill renderers must consume
	// SkillMeta.Root rather than Entry.Path; the latter is kept as a
	// compatibility copy of SkillMeta.Root.
	Skill *SkillMeta

	// Path is the pack-relative location of the source for this entry.
	// For agent / rule / prompt entries it is the entry's markdown file
	// (e.g. "agents/reviewer.md"). For skill entries it is the skill
	// root directory (e.g. "skills/orchestrator"), carrying the same
	// value as Skill.Root() for compatibility with callers that read
	// Entry.Path directly.
	Path string

	// Body is the markdown body of the entry, with the YAML frontmatter
	// stripped. It is byte-exact: every byte from the end of the closing
	// frontmatter delimiter line to EOF is preserved verbatim (including
	// trailing newlines and inner whitespace). Builders can rely on this
	// when round-trip fidelity matters.
	Body []byte
}

// AgentMeta carries the agent-only frontmatter fields. It is referenced by
// Entry.Agent and is non-nil only when Entry.Kind == KindAgent.
type AgentMeta struct {
	// UsesSkills lists dependent skill ids in the form
	// <pack>.skill.<name>. The list is non-empty when present
	// (entry.schema.json enforces minItems: 1).
	UsesSkills []string
}

// ToolConfig is the per-target build directive carried by Entry.Tools.
type ToolConfig struct {
	// Enabled controls whether the entry is built for this target. When
	// nil, the manifest's DefaultTools list is consulted instead.
	Enabled *bool

	// Frontmatter is the target-specific frontmatter merged verbatim into
	// the generated artifact's frontmatter. Same-named fields produced by
	// the neutral conversion are overridden by this map.
	Frontmatter map[string]any
}

// HasFrontmatterFor reports whether the entry declares non-empty
// tools.<target>.frontmatter. Distribution renderers use this to decide
// whether the per-target frontmatter conflicts with a frontmatter-less
// output format (e.g. CLAUDE.md / AGENTS.md / GEMINI.md).
func (e Entry) HasFrontmatterFor(target Target) bool {
	cfg, ok := e.Tools[target]
	return ok && len(cfg.Frontmatter) > 0
}

// FrontmatterFor returns a defensive copy of tools.<target>.frontmatter,
// or nil when the entry does not declare frontmatter for target. The
// copy lets callers merge it into a working map without mutating the
// pack-loaded Entry.
func (e Entry) FrontmatterFor(target Target) map[string]any {
	cfg, ok := e.Tools[target]
	if !ok || len(cfg.Frontmatter) == 0 {
		return nil
	}
	return maps.Clone(cfg.Frontmatter)
}
