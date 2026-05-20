# Codex Target Specification Research

This document investigates the OpenAI Codex CLI configuration format needed to implement `internal/distribution/codex`, and summarizes how it maps to the neutral format (`knowledge-format`).

The research was conducted across context7, official documentation (`developers.openai.com/codex/...`), and related articles. Confirmed and unconfirmed information is documented separately.

---

## 1. Configuration Directories Used by Codex CLI

| Scope | Path | Notes |
|---|---|---|
| user (global) | `$CODEX_HOME` (default: `~/.codex/`) | Shared user settings such as `config.toml` |
| project (local) | `<repo>/.codex/` | Project-specific override settings |

The user-side root can be changed with the `$CODEX_HOME` environment variable. However, the `knit` Installer is designed to use the absolute path provided by the CLI layer as-is, and this package does not resolve environment variables itself.

---

## 2. Codex Mapping by Knowledge Kind

Codex CLI has almost all of the same concepts as Claude Code. The following mappings were confirmed.

| Neutral kind | Codex concept | Destination (relative to root) | Format |
|---|---|---|---|
| `skill` | Agent Skills | `skills/<name>/SKILL.md` | YAML frontmatter (`name`, `description`) + Markdown body. Directory-based |
| `agent` | Subagents | `agents/<name>.toml` | TOML (not frontmatter). `name`, `description`, `developer_instructions` required |
| `rule` | AGENTS.md (custom instructions) | `AGENTS.md` | No YAML frontmatter. Pure Markdown |
| `prompt` | Custom Prompts (`/prompts:<name>`) | `prompts/<name>.md` | YAML frontmatter (`description`, `argument-hint`) + Markdown body. Flat layout (no subdirectories) |

Sources and supplementary notes for each kind follow.

### 2.1 skill (Agent Skills)
- Official: <https://developers.openai.com/codex/skills>
- Directory-based (`skills/<skill-name>/SKILL.md`). Helper files such as `scripts/`, `references/`, `assets/`, and `agents/openai.yaml` can coexist, but in this phase the `knit` Builder generates only `SKILL.md`.
- The frontmatter requires `name` (skill identifier) and `description` (activation condition). The neutral frontmatter `name` and `description` are copied as-is.
- user scope: `~/.codex/skills/<name>/SKILL.md`, project scope: `<repo>/.codex/skills/<name>/SKILL.md`.

### 2.2 agent (Subagents)
- Official: <https://developers.openai.com/codex/subagents>
- The file format is **TOML**, unlike the other kinds.
- Required fields: `name` / `description` / `developer_instructions`.
- Optional fields: `model` / `model_reasoning_effort` / `sandbox_mode` / `nickname_candidates` / `[mcp_servers.<server>]` / `[[skills.config]]`, and others inherited from the parent session.
- `knit` Builder behavior: write the neutral Entry `Body` as the value of `developer_instructions`, and merge optional fields from `tools.codex.frontmatter` into the TOML table. If the same key exists in both the neutral conversion result and `tools.codex.frontmatter`, `tools.codex.frontmatter` takes precedence.
- Neutral frontmatter `uses_skills` (an array of neutral IDs) is treated as unsupported in this phase because Codex `[[skills.config]]` requires `path`, and resolution from a neutral ID to a path is not yet finalized. This should be documented in godoc as unsupported.

### 2.3 rule (AGENTS.md)
- Official: <https://developers.openai.com/codex/guides/agents-md>
- No YAML frontmatter is used. Codex walks user -> project directories and concatenates `AGENTS.override.md` and `AGENTS.md` in order.
- `knit` Builder behavior: concatenate all `kind: rule` Entries in pack manifest order and generate a single `AGENTS.md` with headings, using the same folding strategy as `CLAUDE.md` in the Claude Code target.
- Output is written to `AGENTS.md` at the root. Writing `AGENTS.override.md` is out of scope in this phase, following the convention of preserving user-authored overrides.

### 2.4 prompt (Custom Prompts)
- Official: <https://developers.openai.com/codex/custom-prompts>
- Path: `~/.codex/prompts/<name>.md` (flat; subdirectories are not allowed). The slash command name is derived from the filename without extension and invoked as `/prompts:<name>`.
- Frontmatter: optional `description` and `argument-hint`.
- `knit` Builder behavior: use neutral `name` as the filename, copy neutral `description` to frontmatter `description`, allow override via `tools.codex.frontmatter`, and write the Body as the Markdown body unchanged.

---

## 3. Inventory Root and `Artifact.Path` Convention

Use the same convention as the Claude target: the Builder fills `source.Artifact.Path` with a path relative to the Inventory root, and Installer / Uninstaller / Lister resolve it to an absolute path according to Scope.

| Scope | Inventory root |
|---|---|
| `ScopeUser` | `$HOME/.codex/` (received as an absolute path from the CLI layer) |
| `ScopeProject` | `<project root>/.codex/` (received as an absolute path from the CLI layer) |

The first segment of `Artifact.Path` is limited to these four forms:
- `skills/<name>/SKILL.md`
- `agents/<name>.toml`
- `AGENTS.md`
- `prompts/<name>.md`

Any other path returns `ErrInvalidArtifactPath`.

---

## 4. Label Persistence Strategy

Use the same sidecar strategy as the Claude implementation. knit-managed metadata lives under a dedicated `.knit/` root that is kept separate from the AI tool's own home directory (`~/.codex/`), so users can manage `~/.codex/` as plain Codex configuration without knit artifacts mixed in.

```text
$HOME/.knit/labels/codex/<scope>/<encoded installation id>.json
```

For `scope=project`, the equivalent location is `<projectRoot>/.knit/labels/codex/project/...`.

Reasons for this choice:
- The xattr approach is fragile on NFS and FAT, and could also affect user dotfile synchronization.
- Keeping knit metadata under `~/.knit/` rather than inside `~/.codex/` makes it explicit that knit owns these files; users syncing `~/.codex/` dotfiles do not accidentally pick them up.
- A single sidecar directory tree (`<knitRoot>/labels/<target>/<scope>/`) is shared by every distribution, which makes scope/target enumeration symmetrical.

---

## 5. Frontmatter Merge Rules (Neutral -> Codex)

The `knit` neutral format allows target-specific frontmatter overrides via `tools.<target>.frontmatter` ([knowledge-format.md](../../docs/knowledge-format.md) section "Propagation Rules for `tools.<target>`"). The Codex implementation follows these rules as well.

| kind | Keys emitted by neutral conversion | Mergeable keys | Format |
|---|---|---|---|
| skill | `name`, `description` | Any keys in `tools.codex.frontmatter` (same-name keys prefer frontmatter) | YAML frontmatter |
| agent | `name`, `description`, `developer_instructions` | Any keys in `tools.codex.frontmatter` (same-name keys prefer frontmatter) | **TOML table** (not frontmatter, but merged by the same rule) |
| rule | (none) | (none) | No frontmatter. If `tools.codex.frontmatter` is specified, return `ErrFrontmatterMergeConflict` |
| prompt | `description` | Any keys in `tools.codex.frontmatter` (same-name keys prefer frontmatter) | YAML frontmatter |

Only `agent` is converted to TOML, which differs from other Targets such as Claude and Gemini. The Builder switches conversion strategy internally by kind.

---

## 6. Open Questions and Policy for This Phase

| Item | Status | Policy for this phase |
|---|---|---|
| `agents/openai.yaml` (Codex-specific metadata inside a skill) | Official docs describe it as optional, but the `knit` neutral format has no corresponding field | Do not generate it in this phase. Leave room to support it later via `tools.codex.frontmatter` |
| Mechanism for disabling skills via `[[skills.config]]` | Requires editing `config.toml`, which is outside the Installer's responsibility | `knit` does not modify `config.toml` in this phase |
| Writing to `AGENTS.override.md` | Overrides are typically kept for user-authored content | Do not generate it in this phase. Always output `AGENTS.md` |
| Escaping when copying the neutral Body into `developer_instructions` | TOML multiline strings (`"""..."""`) are expected, but behavior on delimiter collisions needs detailed design | **Locked in as a signature contract**: Build must absorb escaping internally and must not add a new sentinel error. The set of possible Build errors remains only `ErrFrontmatterMergeConflict`. The concrete escaping method (basic string conversion, delimiter splitting, etc.) will be decided during implementation |
| Codex representation of `uses_skills` (neutral IDs) | `[[skills.config]]` requires `path`, and resolution from neutral ID is not finalized | Ignore it in this phase (information drop). Document it in godoc as unsupported |

---

## 7. Reference Links

- [Custom instructions with AGENTS.md](https://developers.openai.com/codex/guides/agents-md)
- [Agent Skills](https://developers.openai.com/codex/skills)
- [Subagents](https://developers.openai.com/codex/subagents)
- [Custom Prompts](https://developers.openai.com/codex/custom-prompts)
- [Config Reference](https://developers.openai.com/codex/config-reference)
- [Features](https://developers.openai.com/codex/cli/features)
