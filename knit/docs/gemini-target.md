# Gemini CLI Target Specification Research

This document summarizes the directory conventions, file formats, and frontmatter conventions for Target `"gemini"` (Gemini CLI), which is handled by `internal/distribution/gemini`. It also serves as the rationale document for the conversion rules from the neutral format ([knowledge-format.md](../../docs/knowledge-format.md)) to Gemini CLI format.

## Research Sources

Main references confirmed as of 2026-05:

- Official documentation (HTML): <https://google-gemini.github.io/gemini-cli/>
- Mirror site (detailed reference): <https://geminicli.com/docs/>
- GitHub (source): <https://github.com/google-gemini/gemini-cli>
- Official blog post on Subagents: <https://developers.googleblog.com/subagents-have-arrived-in-gemini-cli/>

## Overall Structure (Inventory Root)

Gemini CLI has two layers: user-level and project-level. These map directly to `knit` `Scope` values (`user` / `project`).

| Scope | Inventory root |
|---|---|
| `user` | `$HOME/.gemini/` |
| `project` | `<project root>/.gemini/` |

The structure under the root is the same for both scopes:

```text
.gemini/
├── GEMINI.md                  ← Kind: rule (concatenated)
├── skills/<name>/SKILL.md     ← Kind: skill
├── agents/<name>.md           ← Kind: agent
└── commands/<name>.toml       ← Kind: prompt
```

Under `commands/`, namespaces can be expressed with subdirectories such as `git/commit.toml`, which are invoked as namespaced commands like `/git:commit`. However, `knit` places them flat as `<entry-name>.toml`.

## Mapping Table by Kind

| Neutral Kind | Gemini CLI concept | Destination | File format | frontmatter |
|---|---|---|---|---|
| `skill` | Agent Skills | `skills/<name>/SKILL.md` | Markdown | YAML `name`, `description` only |
| `agent` | (Custom) Subagents | `agents/<name>.md` | Markdown | YAML `name`, `description`, and optionally `kind`, `tools`, `mcpServers`, `model`, `temperature`, `max_turns`, `timeout_mins` |
| `rule` | GEMINI.md (hierarchical memory) | `GEMINI.md` | Markdown | None (frontmatter not supported) |
| `prompt` | Custom Commands | `commands/<name>.toml` | **TOML** | TOML itself as key-value pairs (`description`, `prompt`) |

### Details

#### skill (`kind: skill`)

- Official documentation: <https://geminicli.com/docs/cli/skills/>, <https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/skills/builtin/skill-creator/SKILL.md>
- Destination: `skills/<name>/SKILL.md` (each skill has its own directory; bundled assets can coexist, but `knit` initially generates only the Markdown file)
- Frontmatter: only `name` and `description` (`Do not include any other fields in the YAML frontmatter.`)
- `description` is used as the trigger for `activate_skill`, so a one-line description that covers what it does and when to use it is recommended
- The Markdown body is the procedure guide injected into conversation context after activation

#### agent (`kind: agent`)

- Official documentation: <https://geminicli.com/docs/core/subagents/>, <https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md>
- Destination: `agents/<name>.md` (single file)
- Frontmatter (YAML):
  - Required: `name`, `description`
  - Optional: `kind` (`local`/`remote`), `tools` (array), `mcpServers` (object), `model` (string), `temperature` (0.0–2.0), `max_turns` (int), `timeout_mins` (int)
- The Markdown body becomes the System Prompt
- Neutral `uses_skills` (agent-only) has no corresponding field in the Gemini CLI subagent format. Since Skills are independent from subagents, `uses_skills` is **not written into generated artifacts** in this release. Referencing skills from the body can be considered later. An alternative approach might be listing skill activation tools in `tools`, but because that is not established in the official Gemini CLI spec, it is treated as a noop in this release

#### rule (`kind: rule`)

- Official documentation: <https://geminicli.com/docs/cli/gemini-md/>
- Destination: `GEMINI.md` (single file)
- Frontmatter: **not supported** (only the Markdown body is loaded into context)
- When multiple rule entries exist, they are concatenated into one file. The concatenation convention follows the Claude target: the pack name is rendered as H1 and each entry `name` as H2, in manifest order

#### prompt (`kind: prompt`)

- Official documentation: <https://geminicli.com/docs/cli/custom-commands/>, <https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/custom-commands.md>
- Destination: `commands/<name>.toml` (extension `.toml`)
- The **file format is TOML**, not Markdown. This is a Gemini CLI-specific constraint
- Fields (TOML key-value):
  - Required: `prompt` (string, may be multiline)
  - Optional: `description` (one line shown in `/help`)
- Embed the neutral-format Markdown body as the `prompt` value and map neutral `description` into the `description` field
- `knit` does not inject Gemini CLI-specific placeholders such as `{{args}}`, `!{...}`, or `@{...}` into generated `prompt` values; it only embeds the body byte-exactly

## Neutral Field Conversion Rules

| Neutral frontmatter | skill (SKILL.md) | agent (agents/*.md) | rule (GEMINI.md) | prompt (*.toml) |
|---|---|---|---|---|
| `id` | Not written | Not written | Not written | Not written |
| `kind` | Not written (implied by placement) | Not written | Not written | Not written |
| `name` | `name:` | `name:` | (used as H2 when concatenated) | Derived from filename |
| `description` | `description:` | `description:` | **Not written (choice for this release)** | `description = "..."` |
| `tags` | Not written | Not written | Not written | Not written |
| `tools.gemini.frontmatter` | Merged into YAML frontmatter | Merged into YAML frontmatter | **If non-empty, return ErrFrontmatterMergeConflict** | Merged into TOML key-value pairs (type preservation rules below) |
| `uses_skills` | Not written | Not written (noop in this release) | Not written | Not written |

The same-name key precedence rule for `tools.<target>.frontmatter` from [knowledge-format.md section `tools.<target>` propagation rules](../../docs/knowledge-format.md) is preserved for this target as well.

## TOML Type Preservation Rules (`kind: prompt` only)

`tools.<target>.frontmatter` in the neutral format is `map<string, any>` coming from YAML. For TOML output of `kind: prompt`, values are written while preserving types according to these Go type -> TOML type rules:

| Go type (from YAML) | TOML output type |
|---|---|
| `string` | string |
| `bool` | bool |
| `int` / `int64` / `uint64` | integer |
| `float64` | float |
| `time.Time` | datetime |
| `[]any` | array (elements recursively follow the same rules) |
| `map[string]any` | inline table or table (implementation choice; semantics are equivalent) |
| `nil` | omit the key entirely |

If a value outside these supported forms is provided, such as a function value, channel, or unsupported struct, the Builder returns `ErrUnsupportedFrontmatterValue`. In normal use of the neutral format this should not occur, because YAML-loaded values stay within the supported range.

## Error Contract (sentinels exported by this package)

The following should be exported from `internal/distribution/gemini/errors.go`:

| Error | Condition |
|---|---|
| `ErrProjectRootNotConfigured` | An operation uses `ScopeProject` but `projectRoot` is empty |
| `ErrInvalidArtifactPath` | `Artifact.Path` is empty, absolute, escapes root via `..`, or starts with an unsupported segment |
| `ErrFrontmatterMergeConflict` | `tools["gemini"].Frontmatter` is non-empty for `KindRule` |
| `ErrUnmanagedArtifactExists` | A destination already exists but has no matching sidecar, meaning it is not managed by `knit` |
| `ErrUnsupportedFrontmatterValue` | A value with an unsupported type is detected while TOML-encoding `KindPrompt` |

`inventory.ErrTargetMismatch`, `inventory.ErrInvalidScope`, `inventory.ErrAlreadyInstalled`, and `inventory.ErrInstallationNotFound` are reused directly from the `inventory` package.

## Label Persistence Strategy

Use the same sidecar strategy as the Claude target. knit-managed metadata is stored under a dedicated `.knit/` root, kept separate from the Gemini CLI's own home directory (`~/.gemini/`) so users can manage `~/.gemini/` as plain Gemini configuration:

```text
<knitRoot>/labels/<target>/<scope>/<installation id>.json
```

- Example (user scope): `$HOME/.knit/labels/gemini/user/<installation id>.json`
- Example (project scope): `<projectRoot>/.knit/labels/gemini/project/<installation id>.json`
- Gemini CLI does not inspect `<knitRoot>/`, so the two trees coexist without interference
- The same sidecar stores the Label, Provenance, and corresponding Inventory-relative path

## `Artifact.Path` Convention

`source.Artifact.Path` returned by the Builder is **relative to the Inventory root**:

- `skills/orchestrator/SKILL.md`
- `agents/solid-reviewer.md`
- `GEMINI.md`
- `commands/review.toml`

Keeping it Scope-independent centralizes resolution from `(Scope, root)` to an absolute path in Installer / Uninstaller / Lister, matching the Claude design.

## Open / Temporary Areas

| Item | Confidence | Temporary policy |
|---|---|---|
| Stability of the Skills feature (still relatively new) | Officially present in Gemini CLI v0.36+ | Adopt as-is |
| Notation for pointing to skills from the subagent `tools` array | Not finalized in the official spec | Do not reflect `uses_skills` into artifacts in this release |
| `name` field in `commands/*.toml` | Not part of the TOML spec because the command name comes from the filename | Do not write it into TOML; represent it with filename = `<entry-name>.toml` |
| Frontmatter support for `GEMINI.md` | No official frontmatter specification; only Markdown body is loaded | Do not add frontmatter |
| Distribution via the Extension (`gemini-extension.json`) format | Separate mechanism layer | Not supported by this target. There is room to introduce a separate `gemini-extension` Target later |

## Reference Links

- Extensions overview: <https://google-gemini.github.io/gemini-cli/docs/extensions/>
- Custom commands: <https://geminicli.com/docs/cli/custom-commands/>
- Subagents: <https://geminicli.com/docs/core/subagents/>
- Agent Skills: <https://geminicli.com/docs/cli/skills/>
- GEMINI.md (memory): <https://geminicli.com/docs/cli/gemini-md/>
- Configuration reference: <https://geminicli.com/docs/reference/configuration/>
