# Knowledge Format Specification

`knowledge/` is the canonical, tool-neutral source consumed by `knit`.

## File structure

```text
knowledge/
└── <pack-name>/
    ├── manifest.yaml
    ├── skills/
    │   └── <name>/
    │       ├── SKILL.md
    │       └── ... optional sibling assets
    └── agents/
        └── <name>.md
```

Names use kebab-case. Only `skill` and `agent` are canonical kinds.

Custom commands and reusable prompts are represented as skills with
`invocation: manual`. Always-on repository instructions belong in the
repository's native instruction files (`AGENTS.md`, `CLAUDE.md`, or
`GEMINI.md`), outside knowledge packs.

## Entry frontmatter

Every `SKILL.md` and agent Markdown file starts with YAML frontmatter.

| field | type | required | applies to | meaning |
|---|---|:---:|---|---|
| `id` | string | yes | both | `<pack>.<kind>.<entry>` |
| `kind` | `skill` or `agent` | yes | both | Canonical entry kind |
| `name` | string | yes | both | Target-facing name |
| `description` | string | yes | both | Discovery summary |
| `tags` | string[] | no | both | Classification |
| `tools` | map | no | both | Target-specific enablement and metadata |
| `invocation` | `both` or `manual` | no | skill | Invocation policy; defaults to `both` |
| `uses_skills` | string[] | no | agent | Neutral IDs of dependent skills |

`invocation: both` is the default and adds no neutral restriction on explicit
or implicit selection. Target-specific metadata may narrow it. `manual` is an
authoritative requirement for explicit invocation only.

### Target configuration

`tools.<target>` accepts:

| field | type | meaning |
|---|---|---|
| `enabled` | bool | Include or exclude the entry for this target. If omitted, use `default_tools` |
| `frontmatter` | map<string, any> | Target-specific metadata merged over generated fields |

Target-specific values override neutral generated values on key collisions.
Target-required policy fields derived from neutral semantics remain
authoritative; for example, a manual Claude skill always emits
`disable-model-invocation: true`.

## Skill directories

A skill's manifest path names its directory, not its `SKILL.md` file:

```yaml
- id: example.skill.cleanup-devenv
  path: skills/cleanup-devenv
```

`knit` loads `skills/cleanup-devenv/SKILL.md` as the entry body and preserves
all sibling assets recursively. `SKILL.md` cannot also appear as a sibling
asset.

## Manifest

`knowledge/<pack>/manifest.yaml` contains:

| field | type | required | meaning |
|---|---|:---:|---|
| `pack` | string | yes | Pack name; must match the directory |
| `version` | semver string | yes | Pack version |
| `description` | string | yes | Pack summary |
| `default_tools` | string[] | no | Targets enabled by default |
| `entries` | Entry[] | yes | Entry IDs and pack-relative paths |

Example:

```yaml
pack: example
version: 0.1.0
description: Example pack
default_tools: [claude, codex]
entries:
  - id: example.skill.cleanup-devenv
    path: skills/cleanup-devenv
  - id: example.agent.reviewer
    path: agents/reviewer.md
```

The manifest and file frontmatter deliberately repeat the entry ID. The loader
rejects mismatches and duplicate IDs.

## Invocation mapping

| neutral value | Claude Code | Codex | Gemini CLI |
|---|---|---|---|
| omitted / `both` | Normal skill | Normal skill | Normal skill |
| `manual` | `disable-model-invocation: true` in `SKILL.md` | `policy.allow_implicit_invocation: false` in `agents/openai.yaml` | Unsupported; build fails clearly |

For Codex, an existing `agents/openai.yaml` sibling is preserved and merged;
the generated manual-invocation policy is authoritative.

## Output locations

| target | user skills | project skills | user agents | project agents |
|---|---|---|---|---|
| Claude Code | `~/.claude/skills/` | `<repo>/.claude/skills/` | `~/.claude/agents/` | `<repo>/.claude/agents/` |
| Codex | `~/.agents/skills/` | `<repo>/.agents/skills/` | `$CODEX_HOME/agents/` (normally `~/.codex/agents/`) | `<repo>/.codex/agents/` |
| Gemini CLI | `~/.gemini/skills/` | `<repo>/.gemini/skills/` | `~/.gemini/agents/` | `<repo>/.gemini/agents/` |

## Validation and migration

The schemas are [manifest.schema.json](../schema/manifest.schema.json) and
[entry.schema.json](../schema/entry.schema.json). They reject unknown fields,
invalid kind/path combinations, `invocation` on agents, and `uses_skills` on
skills.

Entry identity is coherent: the pack directory and `manifest.pack` agree; each
entry ID uses that pack and the same kind as its frontmatter; and the manifest
path basename matches the ID's entry-name segment.

Legacy `kind: rule` and `kind: prompt` entries are intentionally rejected:

- Move always-on content to the appropriate repository instruction file.
- Convert reusable prompts or commands into `kind: skill` with
  `invocation: manual`.

This is a breaking format and layout migration. Existing legacy installations
are not migrated automatically. Before upgrading, uninstall them with the
previous `knit` version. If already upgraded, remove the old knit labels and
their corresponding legacy artifacts manually, including Codex skills under
`.codex/skills`; reinstall current packs afterward.

See `knowledge/structure-behavior-design/` and `knowledge/git-pr-workflow/` for
working examples.
