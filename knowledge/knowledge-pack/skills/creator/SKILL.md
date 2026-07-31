---
id: knowledge-pack.skill.creator
kind: skill
name: knowledge-pack-creator
description: |
  Design, create, substantially restructure, and validate a tool-neutral knowledge pack.
  Use when adding a new pack, deciding whether content belongs in a skill or agent,
  decomposing pack responsibilities, migrating command or prompt workflows into skills,
  or reviewing pack cohesion and portability. Do not use for small metadata or content
  edits that already follow an established pack pattern.
tags: [pack-authoring, knowledge-design, skill-design]
---

# Create Knowledge Pack

Design the pack before creating files. Treat a pack as a cohesive installation and
versioning unit, not as a convenient folder for unrelated instructions.

## Read the local contract

Before deciding the structure:

1. Find the repository root and read every applicable `AGENTS.md`.
2. Read the root README, the current knowledge-format documentation, and the live manifest
   and entry schemas.
3. Read any repository-provided knowledge-authoring guidance for mechanical format rules.
4. Inspect at least one current pack with a similar shape.
5. Inspect the relevant target builders when target-specific behavior matters.
6. Check `git status` and preserve unrelated work.

Treat the live schemas and builder behavior as the executable contract. If documentation,
schemas, and builders disagree, stop authoring the affected field, report the mismatch, and
resolve it instead of guessing.

Read [placement-guide.md](references/placement-guide.md) to decide whether proposed content
belongs in a pack at all. Then read
[responsibility-guide.md](references/responsibility-guide.md) before choosing entries.
Read [quality-checklist.md](references/quality-checklist.md) before implementation and again
before completion.

## Workflow

### 1. Establish the pack charter

Write a compact charter containing:

- intended users and recurring outcome
- two or three representative requests
- non-goals
- supported targets
- why the entries belong in one install/version lifecycle

Choose a short kebab-case pack name. Apply the pack prefix to the public artifact name while
keeping the entry key local to the pack:

```text
pack:        knowledge-pack
entry:       creator
public name: knowledge-pack-creator
entry id:    knowledge-pack.skill.creator
source path: skills/creator
```

For an agent, use `<pack>.agent.<entry>`, `agents/<entry>.md`, and public name
`<pack>-<entry>`. Do not repeat the pack prefix in the entry key, ID suffix, or source path;
the builder uses the public `name` for the generated target path.

Split content when entries do not share a domain, audience, or change lifecycle. A one-entry
pack is acceptable when it is independently useful and installable.

### 2. Model the entries

Classify each reusable capability as a `skill` or `agent` using the responsibility guide.
Do not introduce `command`, `prompt`, or `rule` merely to represent an invocation style or
scope. Model manual commands and reusable prompts as skills; keep durable repository guidance
in scoped `AGENTS.md` files and deterministic enforcement in code, tests, hooks, or policy.

Apply the placement guide before moving existing documentation into a pack. Keep project
overview, development setup, architecture, public contracts, and current-task scope in their
appropriate source-of-truth surfaces. Link to those sources when a skill needs them; do not
copy them into the skill.

Create an agent only when isolated context, a specialist role, restricted tools, independent
review, or delegation is part of the required behavior. Declare its skill dependencies through
the repository's supported dependency field.

Reject cycles and hidden dependencies. Prefer one public orchestration skill plus focused
supporting skills or agents when a workflow has multiple stages.

### 3. Produce an artifact plan

Before editing, list every proposed entry:

| ID | Kind | Path | Responsibility | Trigger or caller | Dependencies | Targets | Resources |
|---|---|---|---|---|---|---|---|

Verify that each responsibility has exactly one owner and every dependency points in a clear
direction. Remove entries that only rename another entry or repeat repository-wide guidance.

### 4. Author the pack

Create only files required by the current schema and the planned entries:

1. Add `knowledge/<pack>/manifest.yaml`.
2. Add each skill as `skills/<name>/SKILL.md`.
3. Add skill resources only when they improve repeatability:
   - `scripts/` for deterministic repeated operations
   - `references/` for detailed knowledge loaded on demand
   - `assets/` for files copied or transformed into outputs
   - `agents/` for product-specific skill metadata when supported
4. Add each agent in the path required by the current schema.
5. Keep the neutral body portable. Add target-specific metadata only where behavior genuinely
   differs.

Write skill descriptions as trigger contracts: state what the skill does, when it should run,
and meaningful exclusions. Write bodies in imperative form. Keep the main `SKILL.md` concise
and link directly to each reference that may be needed. Do not add README, changelog, or
installation files inside a skill.

Write agents with a narrow role, explicit inputs, boundaries, and output contract. Do not use
an agent as a container for ordinary procedural instructions.

### 5. Validate structure and behavior

Validate in this order:

1. Check manifest, entry frontmatter, IDs, paths, names, and declared dependencies against the
   live schemas.
2. Confirm that manifest entries and files are one-to-one: no missing, duplicate, or orphaned
   entries.
3. Run the repository's pack build for every enabled target. From this repository, the usual
   form is:

   ```bash
   go -C knit run . build --target=<target> ../knowledge/<pack>
   ```

4. Inspect generated artifact paths and metadata, not only command success.
5. Test representative trigger and non-trigger prompts for each public skill.
6. Verify agents can perform their stated role using only declared skills and allowed context.
7. Run `git diff --check` and review the complete diff for unrelated changes.

Do not weaken schemas or builder tests merely to make a malformed pack pass.

## Completion report

Return:

1. pack charter and entry summary
2. files created or changed
3. key classification and boundary decisions
4. validation commands and results for each target
5. remaining limitations or follow-up work
