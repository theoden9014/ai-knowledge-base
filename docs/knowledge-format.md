# Knowledge Format Specification

The formal specification of the neutral format stored under `knowledge/`. `knit` reads this format as input and converts it into each AI tool's specific format and directory structure.

The "Abstract knowledge model" and "Knowledge packs" sections in the root [README.md](../README.md) define the conceptual model. This document defines the schema of the actual files that satisfy that model.

---

## File Structure

```text
knowledge/
└── <pack-name>/
    ├── manifest.yaml          # Pack definition (required)
    ├── skills/<name>.md       # kind: skill
    ├── agents/<name>.md       # kind: agent
    ├── rules/<name>.md        # kind: rule
    └── prompts/<name>.md      # kind: prompt
```

- `<pack-name>` / `<name>` use kebab-case
- Subdirectories other than `manifest.yaml` are created only when at least one entry of that kind exists (do not keep unnecessary directories)

---

## `id` Naming Convention

Each knowledge file has a neutral ID in the following format:

```text
<pack-name>.<kind>.<entry-name>
```

- Segments are separated by periods `.`
- Each segment uses kebab-case
- `<kind>` is one of `skill` / `agent` / `rule` / `prompt`
- Unique within a pack

Examples:

```text
structure-behavior-design.skill.orchestrator
structure-behavior-design.agent.solid-reviewer
```

---

## Common Frontmatter Schema

All knowledge files use YAML frontmatter followed by a Markdown body.

| field | type | Required | Description |
|---|---|:---:|---|
| `id` | string | ✓ | Neutral ID following the naming convention above |
| `kind` | enum (`skill` / `agent` / `rule` / `prompt`) | ✓ | Knowledge type |
| `name` | string | ✓ | Identifier passed to the target tool. By convention, `<pack-name>-<entry-name>` |
| `description` | string | ✓ | Summary. Multi-line values may wrap with `\|` |
| `tags` | string[] | - | Tags for classification |
| `tools` | map<target, ToolConfig> | - | Target-specific settings (described below) |
| `uses_skills` | string[] | - | Array of neutral IDs for dependent skills. Only meaningful for `kind: agent` |

### `tools.<target>` (ToolConfig)

Build instructions for each target.

| field | type | Required | Description |
|---|---|:---:|---|
| `enabled` | bool | - | When `true`, include the entry in the build target. If omitted, follow `default_tools` in `manifest.yaml` |
| `frontmatter` | map<string, any> | - | Additional metadata expanded directly into the generated frontmatter as target-specific frontmatter |

---

## Additional Rules by `kind`

### `kind: skill`

- No additional fields
- The body defines procedures, viewpoints, and output formats

### `kind: agent`

- Dependencies can be declared in `uses_skills` using neutral skill IDs
- The body defines the specialist's role, review perspectives, and output format

Builders convert the neutral IDs in `uses_skills` into the target tool's reference format (for example, for Claude Code, extract `<name>` and expand it into the `skills:` array).

### `kind: rule`

- No additional fields are currently defined (reserved for future extension)
- The body defines always-on instructions and assumptions

The rules for combining multiple rule entries into one file, including ordering and heading conventions, are defined separately.

### `kind: prompt`

- No additional fields are currently defined (reserved for future extension)
- The body contains reusable prompts and slash command content

---

## `manifest.yaml` Schema

The definition file for a pack, located at `knowledge/<pack-name>/manifest.yaml`.

| field | type | Required | Description |
|---|---|:---:|---|
| `pack` | string | ✓ | Pack name in kebab-case. Must match the directory name |
| `version` | string | ✓ | semver |
| `description` | string | ✓ | Pack overview |
| `default_tools` | string[] | - | List of targets enabled when `tools.<target>.enabled` is omitted on an entry |
| `entries` | Entry[] | ✓ | List of entries included in the pack |

### Entry

| field | type | Required | Description |
|---|---|:---:|---|
| `id` | string | ✓ | Must match the file's frontmatter `id` |
| `path` | string | ✓ | Relative path from the pack root (for example, `skills/orchestrator.md`) |

The manifest and each file's frontmatter are intentionally redundant so the repository preserves both pack-wide discoverability and self-description of individual files. Consistency checks are performed during `knit` builds.

---

## Propagation Rules for `tools.<target>`

Target-specific builders in `knit` generate artifacts using the following rules:

1. Include only entries where `tools.<target>.enabled` is `true` or the target is included in `default_tools`
2. Convert neutral fields such as `name`, `description`, and `uses_skills` according to the target's conventions and write them into the generated frontmatter
3. Merge keys and values from `tools.<target>.frontmatter` directly into the generated frontmatter
4. If the same field exists in both the neutral conversion result and `tools.<target>.frontmatter`, prioritize `tools.<target>.frontmatter` as an explicit override

---

## Handling the Body (Markdown)

- Copy the Markdown body as-is into the generated artifact body
- In the current scope, do not perform template expansion, variable substitution, or link resolution to other entries
- Future support for cross references such as `[[id]]` or variable expansion remains possible, but is not part of this specification

---

## Validation

The neutral format is validated using [JSON Schema](https://json-schema.org/). Schema files are placed directly under `schema/`.

| Target | Schema file |
|---|---|
| Pack definition (`manifest.yaml`) | [`schema/manifest.schema.json`](../schema/manifest.schema.json) |
| Entry frontmatter (top of each `.md`) | [`schema/entry.schema.json`](../schema/entry.schema.json) |

The same schemas are referenced both by build-time validation in `knit` and by real-time checks in editors such as YAML Language Server.

### Main constraints enforced by the schemas

- `id`, `name`, `pack`, and similar fields use kebab-case
- `version` uses semver
- `id` must follow the format `<pack>.<kind>.<entry>`, and each `uses_skills` element must follow `<pack>.skill.<entry>`
- Using `uses_skills` outside `kind: agent` is an error
- Unknown fields such as typos are rejected through `additionalProperties: false`

---

## Examples

See `knowledge/structure-behavior-design/` for examples.

- Skill example: [`knowledge/structure-behavior-design/skills/orchestrator.md`](../knowledge/structure-behavior-design/skills/orchestrator.md)
- Agent example: [`knowledge/structure-behavior-design/agents/solid-reviewer.md`](../knowledge/structure-behavior-design/agents/solid-reviewer.md)
- Manifest example: [`knowledge/structure-behavior-design/manifest.yaml`](../knowledge/structure-behavior-design/manifest.yaml)
