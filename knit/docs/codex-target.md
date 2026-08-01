# Codex target

This document records the Codex-specific mapping implemented by `knit`.

## Supported mappings

| neutral kind | artifact | physical location |
|---|---|---|
| `skill` | `skills/<name>/SKILL.md` plus sibling assets | user: `~/.agents/`; project: `<repo>/.agents/` |
| `agent` | `agents/<name>.toml` | user: `$CODEX_HOME` or `~/.codex/`; project: `<repo>/.codex/` |

Artifact paths remain logical (`skills/...` or `agents/...`). Codex's resolver
selects the physical root by top-level artifact family, so labels and
installation IDs remain stable.

## Skill rendering

The neutral `name` and `description` become YAML frontmatter. Target-specific
frontmatter is merged over those fields. Sibling assets are copied recursively.

For `invocation: manual`, `knit` creates or merges:

```yaml
policy:
  allow_implicit_invocation: false
```

at `agents/openai.yaml` inside the skill directory. Existing metadata such as
`interface` is preserved; the neutral manual policy is authoritative. Invalid
YAML or a non-mapping `policy` returns `ErrInvalidSkillMetadata`.

## Agent rendering

Agents are emitted as TOML with:

- `name`
- `description`
- `developer_instructions`
- additional target fields from `tools.codex.frontmatter`

`uses_skills` is currently not emitted because Codex's concrete skill-path
configuration does not map directly from a neutral ID.

## Scope and metadata

Labels are kept outside Codex-owned directories under
`~/.knit/labels/codex/...` or `<repo>/.knit/labels/codex/...`. This allows one
logical inventory to manage both `.agents` and `.codex` physical roots.

`AGENTS.md` is intentionally not generated from packs. It is repository
governance and should point to more specific repository documentation or
workflows when needed.

## References

- [Agent Skills](https://developers.openai.com/codex/skills)
- [Custom agents](https://developers.openai.com/codex/subagents)
- [AGENTS.md guidance](https://developers.openai.com/codex/guides/agents-md)
