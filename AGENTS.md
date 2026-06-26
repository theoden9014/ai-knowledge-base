# Repository Guidelines

This repository stores tool-neutral AI coding knowledge under `knowledge/` and
the `knit` builder/installer under `knit/`.

## Development Checks

- Run `go test ./...` from `knit/` before changing Go code or knowledge format
  behavior.
- When editing schemas, update both copies:
  - `schema/*.schema.json`
  - `knit/internal/source/schemas/*.schema.json`
- The schema sync test fails intentionally when those copies drift.
- Knowledge content changes should be loadable by `knit/internal/source` and
  must keep manifest entry ids, paths, and entry frontmatter ids consistent.

## Knowledge Authoring

- Put canonical source under `knowledge/<pack>/`.
- Each pack needs `manifest.yaml` with `pack`, `version`, `description`,
  optional `default_tools`, and an `entries` list.
- Skill entries use a directory path such as `skills/<name>` and store their
  body in `skills/<name>/SKILL.md`.
- Agent, rule, and prompt entries use single markdown files under `agents/`,
  `rules/`, or `prompts/`.
- Entry ids use `<pack>.<kind>.<name>` and must match the id in the entry
  frontmatter exactly.
- Entry `name` values are target-facing identifiers. Prefer
  `<pack>-<entry-name>` unless an existing pack establishes a different
  convention.
- Keep knowledge tool-neutral by default. Use `tools.<target>` frontmatter only
  when a target needs explicit enablement or target-specific fields.
- If a skill needs assets, keep them as siblings of `SKILL.md` inside the skill
  directory.

## Repository Style

- Keep docs and knowledge in plain Markdown with YAML frontmatter where the
  schemas require it.
- Prefer small, focused changes: avoid mixing knowledge edits, schema changes,
  and `knit` behavior changes unless they depend on each other.
- Do not commit generated local installs from Claude, Codex, or Gemini config
  directories.
