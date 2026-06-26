---
id: ai-knowledge-base.skill.knowledge-authoring
kind: skill
name: ai-knowledge-base-knowledge-authoring
description: Author or update tool-neutral knowledge packs in ai-knowledge-base.
tags:
  - knowledge
  - authoring
  - repository
---

# Knowledge Authoring

Use this skill when creating or updating content under `knowledge/` in the
ai-knowledge-base repository.

## Authoring Flow

1. Inspect the target pack first.
   - Read `manifest.yaml`.
   - Read nearby entries of the same kind.
   - Preserve existing naming, frontmatter, and tone unless there is a clear
     reason to change them.
2. Choose the right entry kind.
   - Use `skill` for reusable procedures, workflows, checklists, viewpoints, or
     output formats.
   - Use `agent` for a specialist role that can operate in an isolated context.
   - Use `rule` for always-on instructions.
   - Use `prompt` for reusable prompt or command-style entry points.
3. Add or update the manifest entry.
   - Entry id format is `<pack>.<kind>.<name>`.
   - Skill paths are directories: `skills/<name>`.
   - Agent, rule, and prompt paths are files:
     `agents/<name>.md`, `rules/<name>.md`, or `prompts/<name>.md`.
4. Add or update the entry source.
   - Skills place their body in `skills/<name>/SKILL.md`.
   - Other entry kinds are single markdown files.
   - YAML frontmatter must include `id`, `kind`, `name`, and `description`.
   - The frontmatter `id` must exactly match the manifest entry id.
5. Keep the content tool-neutral.
   - Describe concepts and workflows independently of Claude, Codex, Gemini, or
     another target unless the entry is intentionally target-specific.
   - Use `tools.<target>` only for target-specific frontmatter or enablement.
6. Validate before finishing.
   - Run `go test ./...` from `knit/`.
   - If local sandboxing blocks the default Go caches, run with `GOCACHE` and
     `GOMODCACHE` pointed at writable directories inside the repository or
     `/tmp`.

## Quality Bar

- A skill should say when to use it, what context to inspect, what steps to
  follow, and what output or verification is expected.
- An agent should define its role, scope, inputs, review criteria, and output
  format.
- A rule should be short enough to stay always-on without distracting from the
  active task.
- A prompt should be directly executable by a user or assistant and should make
  required inputs clear.
- Prefer concrete repository paths and id examples over generic prose.
- Avoid duplicating large sections from another entry. Extract shared guidance
  into a separate skill only when it will be reused.
