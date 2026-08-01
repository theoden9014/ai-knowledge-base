# Knowledge Pack Quality Checklist

Use the relevant sections of this checklist while designing, implementing, maintaining, and
reviewing an ai-knowledge-base pack.

## Contract

- [ ] Applicable `AGENTS.md`, repository docs, live schemas, and relevant builders were read.
- [ ] Any disagreement between docs, schemas, and implementation was resolved or reported.
- [ ] The pack name, version, paths, IDs, and frontmatter satisfy the current schema.
- [ ] Public names use `<pack>-<entry>` while IDs and source paths use the local entry key.
- [ ] The pack prefix is not duplicated in the entry key, ID suffix, or source path.
- [ ] Only currently supported kinds and fields are used.

## Cohesion and responsibility

- [ ] The pack has a one-sentence outcome and a clear intended audience.
- [ ] All entries share an installation, versioning, and change lifecycle.
- [ ] Every entry owns one distinct responsibility.
- [ ] Proposed content passed the placement flow before being made a pack entry.
- [ ] Skills represent reusable capabilities or knowledge.
- [ ] Agents exist only where isolated context, specialist judgment, or delegation is required.
- [ ] Manual invocation is modeled as skill policy, not as a command or prompt kind.
- [ ] Always-on guidance and deterministic enforcement are outside the pack where appropriate.
- [ ] Canonical project documentation is linked rather than copied into Skills or Agents.
- [ ] Current-task objectives, scope, and completion criteria remain runtime inputs.
- [ ] Every required dependency is representable by the schema and preserved by every enabled target.
- [ ] Unsupported dependency behavior is made self-contained or disabled for that target.

## Skill quality

- [ ] The description states what the skill does and when it should and should not trigger.
- [ ] The body is imperative, task-oriented, and concise.
- [ ] Inputs, outputs, decisions, stopping conditions, and error handling are clear.
- [ ] Detailed material is moved into directly linked references.
- [ ] Scripts exist only for repeated deterministic operations and have been executed in tests.
- [ ] Assets are output resources rather than additional documentation.
- [ ] No README, changelog, installation guide, or placeholder file exists inside the skill.

## Agent quality

- [ ] The role and authority are narrow and concrete.
- [ ] Required inputs and allowed context are explicit.
- [ ] Tool, model, and permission restrictions are intentional.
- [ ] The output contract is directly usable by the caller.
- [ ] `uses_skills` is used only when every enabled target preserves the dependency.
- [ ] The agent does not duplicate the main workflow or act as an unnecessary persona.
- [ ] The benefit of isolation or delegation justifies the Agent's coordination and context cost.

## Portability

- [ ] Core instructions remain tool-neutral.
- [ ] Target-specific metadata is minimal and justified.
- [ ] Provider invocation syntax is not treated as a semantic entry kind.
- [ ] Every enabled target can represent the required behavior without silent degradation.
- [ ] Unsupported target behavior is disabled or documented instead of approximated incorrectly.

## Repository integrity

- [ ] Every manifest entry resolves to exactly one source entry.
- [ ] No source entry is omitted from the manifest.
- [ ] No entry or sibling file escapes its pack or skill directory.
- [ ] Names and IDs are unique.
- [ ] Existing user changes are preserved.
- [ ] The diff contains only intended pack changes.

## Verification

- [ ] The pack builds successfully for every enabled target.
- [ ] Generated paths, frontmatter, sibling files, and dependency mappings were inspected.
- [ ] Representative positive trigger prompts select each public skill.
- [ ] Representative negative prompts do not select overly broad skills.
- [ ] Agents were checked against their input and output contracts.
- [ ] Maintenance risks such as stale guidance, unused Skills, and over-triggering were reviewed.
- [ ] `git diff --check` succeeds.

Report the exact commands run and any checks that could not be completed.
