// Package codex provides the concrete distribution implementation for the
// OpenAI Codex CLI.
//
// This package implements the four interfaces source.Builder,
// inventory.Installer, inventory.Uninstaller, and inventory.Lister, and keeps
// all knowledge specific to Target = "codex" in one place: directory
// conventions, frontmatter and TOML conventions, rule concatenation, and label
// persistence.
//
// For the underlying target research, see
// [/Users/junki.kaneko/workdir/ai-knowledge-base/knit/docs/codex-target.md].
//
// # Target
//
// The target handled by this package is exposed as [Target], a source.Target
// constant whose value is the kebab-case string "codex". Each Builder,
// Installer, Uninstaller, and Lister implementation is dedicated to this single
// target, and its Target method returns that constant.
//
// # Directory Conventions (Inventory)
//
// Following the Codex CLI conventions, installations are placed as follows. The
// inventory root changes by scope:
//
//   - ScopeUser:    $CODEX_HOME (typically $HOME/.codex/)
//   - ScopeProject: <project root>/.codex/
//
// The structure under each root is the same for every scope:
//
//   - skills/<name>/SKILL.md   <- Kind: skill (YAML frontmatter + Markdown body)
//   - agents/<name>.toml       <- Kind: agent (TOML format, per the Codex subagent spec)
//   - prompts/<name>.md        <- Kind: prompt (YAML frontmatter + flat Markdown body)
//   - AGENTS.md                <- Kind: rule (multiple rule entries concatenated with headings)
//
// Note: prompts/ cannot contain subdirectories. Codex recognizes only
// top-level `.md` files there as slash-command candidates, per the official
// spec, and this Builder follows that rule by generating only flat prompt
// files.
//
// # Artifact.Path Convention
//
// source.Artifact.Path values produced by Builder are relative to the inventory
// root above, for example "skills/orchestrator/SKILL.md",
// "agents/solid-reviewer.toml", "AGENTS.md", or "prompts/review.md". This
// convention lets Builder produce paths without depending on scope, while the
// Installer, Uninstaller, and Lister centralize `(Scope, root) -> absolute
// path` resolution in one place.
//
// # Frontmatter / TOML Conventions
//
// Builder switches output format by kind.
//
// SKILL.md artifacts for kind `skill` use YAML frontmatter. The generated
// frontmatter is built as follows:
//
//  1. Neutral fields such as `name` and `description` are copied into the
//     Codex-specific form.
//  2. Keys and values from Entry.Tools["codex"].Frontmatter are merged in as
//     overwrites, with Frontmatter values taking precedence on key collisions.
//
// agents/<name>.toml artifacts for kind `agent` are **TOML**, unlike the YAML
// frontmatter used by other kinds. Per the Codex subagent specification, they
// emit:
//
//   - name (from the neutral `name`)
//   - description (from the neutral `description`)
//   - developer_instructions (the neutral body copied as a multiline string)
//   - any additional fields supplied through Entry.Tools["codex"].Frontmatter,
//     such as model, sandbox_mode, or mcp_servers, with Frontmatter taking
//     precedence over neutral values when keys collide
//
// The neutral frontmatter field `uses_skills`, which is expressed as neutral
// IDs, is intentionally ignored in this phase. Codex `[[skills.config]]`
// requires concrete paths, and the mapping from neutral IDs to paths is still
// undecided.
//
// prompts/<name>.md artifacts for kind `prompt` use YAML frontmatter. The
// generated frontmatter copies only `description` from the neutral form, and
// Entry.Tools["codex"].Frontmatter can add fields such as `argument-hint`.
//
// AGENTS.md artifacts for kind `rule` never have frontmatter. They consist only
// of Markdown body, with the pack name as H1 and each rule entry name as H2,
// concatenated in manifest order. Supplying Entry.Tools["codex"].Frontmatter in
// this case returns ErrFrontmatterMergeConflict.
//
// # Label Persistence
//
// Labels are not embedded into artifact files. Instead they are stored in a
// dedicated sidecar directory under the inventory root:
//
//   - <inventory root>/.knit/labels/<target>/<scope>/<installation id>.json
//
// Each sidecar stores the minimal metadata required to reconstruct an
// installation: Label, Provenance, and the corresponding relative inventory
// path. Codex CLI ignores the `.knit/` subtree, so this layout coexists cleanly
// with Codex's native conventions.
//
// The reasons for using a sidecar model are the same as in the claude target:
//
//   - xattrs work on both macOS and Linux, but they are fragile on filesystems
//     such as NFS or FAT and can interfere with dotfiles synchronization.
//   - Managing labels outside the inventory root makes it hard to track moves
//     or deletions of the Codex configuration itself.
//   - Sidecars under the same root keep the whole inventory self-contained
//     inside a single `~/.codex/` tree, making moves and deletions easy to
//     follow.
//
// # Scope Handling
//
// The project root that anchors `"<project>/.codex"` for ScopeProject is passed
// in via constructor argument. This package does not discover the project root
// on its own; that decision belongs to the CLI layer.
//
// # Not Supported in This Phase
//
//   - No `agents/openai.yaml` file is generated for skills.
//   - Nothing is written to `AGENTS.override.md`, preserving user-authored
//     content.
//   - No enable/disable changes are applied through `config.toml`
//     `[[skills.config]]`.
//   - `uses_skills` on agent entries is currently dropped.
package codex
