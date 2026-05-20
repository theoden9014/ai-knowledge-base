# knit

A package manager dedicated to managing AI knowledge.

It installs and manages tool-neutral "knowledge packs" stored under `knowledge/` in the `ai-knowledge-base` repository into the configuration directories of AI coding tools such as Claude Code, Codex CLI, and Gemini CLI.

## Positioning

- It handles knowledge for AI coding tools only (Skill / Agent / Rule / Prompt)
- It uses package-management vocabulary such as `install` / `uninstall` / `list` / `update` on a per-pack basis
- It does not provide a remote registry. `knowledge/` is the single local source of truth

## Responsibilities

| Responsibility | Summary |
|---|---|
| Build | Convert the neutral format under `knowledge/<pack>/` (YAML frontmatter + Markdown) into each target AI tool's specific format and directory structure |
| Distribution | Reflect artifacts by copying files into each AI tool's configuration directory |
| State management | **It does not maintain its own state file or DB**. The filesystem is the source of truth, and each distributed file or directory is identified by metadata labels indicating it came from `knit` |
| Uninstall | Remove only distributed artifacts labeled as originating from this tool |

Because state is not stored in a separate file, if the user manually deletes a distributed artifact, it naturally disappears from `knit list` as well.

## Distribution Scope

`knit` can target distribution in two scopes.

| Scope | Example destination |
|---|---|
| **user** | Under the home directory, such as `~/.claude/skills/`, `~/.codex/`, `~/.gemini/` |
| **project** | Under the current project (for example, equivalent to `<project>/.claude/skills/`) |

## Subcommands (planned)

It is expected to use subcommands such as `install`, `uninstall`, `list`, and `update`. The concrete flag scheme and full subcommand list will be finalized during implementation.

## Status

Not implemented yet. Only the naming and overall positioning have been decided so far. The implementation plan will be documented separately.
