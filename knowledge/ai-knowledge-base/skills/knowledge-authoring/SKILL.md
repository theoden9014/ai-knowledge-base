---
id: ai-knowledge-base.skill.knowledge-authoring
kind: skill
name: ai-knowledge-base-knowledge-authoring
description: |
  Design, create, update, substantially restructure, and validate tool-neutral knowledge
  packs in ai-knowledge-base. Use for any change under knowledge/, including new pack
  boundaries, Skill or Agent responsibility decisions, migrations from commands or prompts,
  routine entry maintenance, metadata updates, and cross-target validation.
tags:
  - knowledge
  - authoring
  - pack-design
  - repository
---

# Knowledge Authoring

Treat a pack as a cohesive installation and versioning unit, not as a convenient folder for
unrelated instructions. Scale the workflow to the change: perform explicit design for a new
pack or a substantial restructure, and use the maintenance path for an established pattern.

## Read the local contract

Before editing:

1. Read every applicable `AGENTS.md`.
2. Read the root README, the current knowledge-format documentation, and the live schemas.
3. Inspect the target pack and at least one similar entry.
4. Inspect target builders when target-specific behavior matters.
5. Check `git status` and preserve unrelated work.

Treat schemas and builder behavior as the executable contract. If documentation, schemas, and
builders disagree, report and resolve the mismatch instead of guessing.

## Select the path

Use the **design path** when any of these are true:

- creating a pack
- splitting, merging, or substantially restructuring a pack
- deciding whether content belongs in a Skill, Agent, persistent instruction, runtime prompt,
  canonical documentation, or enforcement
- migrating custom commands, custom prompts, or generic rules
- introducing a new public capability or unclear responsibility boundary

Use the **maintenance path** when the target pack and entry pattern are already established and
the change is limited to content, metadata, resources, or a compatible entry addition.

For the design path, read
[placement-guide.md](references/placement-guide.md),
[responsibility-guide.md](references/responsibility-guide.md), and
[quality-checklist.md](references/quality-checklist.md). For routine maintenance, read only the
quality checklist sections relevant to the change.

## Design a pack

### 1. Establish the charter

Record:

- intended users and recurring outcome
- two or three representative requests
- non-goals
- supported targets
- why the entries share one install and version lifecycle

Split capabilities that do not share a domain, audience, or change lifecycle. A one-entry pack
is valid when independently useful.

### 2. Place responsibilities

Use Skills for reusable procedures, domain knowledge, checklists, or output contracts. Use
Agents only when isolated context, a specialist role, restricted tools, delegation, or
independent judgment is essential.

Do not introduce a knowledge kind merely to encode an invocation style or scope. Model reusable
command and prompt workflows as Skills. Keep durable repository guidance in the narrowest
applicable persistent instruction file, canonical project information in documentation, current
intent in the runtime prompt, and deterministic guarantees in code or policy.

Prefer one public orchestration Skill plus focused supporting Skills or Agents when a workflow
has multiple independently valuable stages. Reject cycles, duplicated responsibilities, and
hidden dependencies.

### 3. Name entries

Use a short kebab-case pack name and a concise local entry key. Prefix the public artifact name
with the pack name, but do not repeat that prefix in the entry key, ID suffix, or path:

```text
pack:        git-pr-workflow
entry:       review-fix
public name: git-pr-workflow-review-fix
entry id:    git-pr-workflow.skill.review-fix
source path: skills/review-fix
```

### 4. Produce the artifact plan

List each proposed entry before editing:

| ID | Kind | Path | Responsibility | Trigger or caller | Dependencies | Targets | Resources |
|---|---|---|---|---|---|---|---|

Verify that each responsibility has one owner and each dependency has a clear direction. Remove
entries that only rename another entry or repeat repository-wide guidance.

## Author or maintain the pack

1. Update `knowledge/<pack>/manifest.yaml`.
2. Add or update each source in the path required by the live schema.
3. Keep IDs, paths, names, and manifest entries one-to-one.
4. Keep the body tool-neutral. Add target-specific metadata only through a field or resource
   that the current schema and builder can scope to that target.
5. Add Skill resources only when they improve repeatability:
   - `scripts/` for deterministic repeated operations
   - `references/` for detailed knowledge loaded on demand
   - `assets/` for files copied or transformed into outputs
6. Write Skill descriptions as trigger contracts containing the action, positive triggers, and
   meaningful exclusions. Keep the main file concise and link directly to optional references.
7. Give each Agent a narrow role, explicit inputs, boundaries, and output contract. Declare
   dependencies using only fields supported by the repository.

Do not add target-specific sibling resources such as `agents/openai.yaml` while the builder
copies every Skill sibling to every enabled target. Add them only after target-scoped resource
support exists.

Do not add auxiliary README, changelog, installation, or quick-reference files inside a Skill.
Do not weaken schemas or builder tests to accept malformed content.

## Validate

Validate in this order:

1. Check manifest, frontmatter, IDs, paths, names, and dependencies against the live schemas.
2. Confirm there are no missing, duplicate, or orphaned entries.
3. Run repository tests:

   ```bash
   go -C knit test ./...
   ```

4. Build the affected pack for every enabled target:

   ```bash
   go -C knit run . build --target=<target> -o <temporary-output-dir>/<target> ../knowledge/<pack>
   ```

5. Inspect the emitted files, paths, and metadata in each target's output directory, not only
   command success.
6. Test representative trigger and non-trigger requests for public Skills.
7. Run `git diff --check` and inspect the complete diff.

If sandboxing blocks Go caches, point `GOCACHE` and `GOMODCACHE` at writable task-specific
directories.

## Report completion

Return:

1. the charter and artifact plan for design-path changes
2. files created or changed
3. material placement, responsibility, and naming decisions
4. validation commands and results for every enabled target
5. remaining limitations or follow-up work
