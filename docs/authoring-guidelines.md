# Authoring responsibilities

Use this decision guide when deciding whether knowledge belongs in repository
instructions, a skill, or an agent.

## Responsibility matrix

| mechanism | responsibility | use when | avoid |
|---|---|---|---|
| Root `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` | Small, repository-wide routing and invariant layer | Every task in the repository needs the instruction, or the file routes agents to scoped guidance | Full workflows, rare use cases, tool tutorials, large policy copies |
| Scoped instruction file | Always-on constraints for one subtree | Every task below that directory needs the constraint | Instructions relevant only when a particular workflow is chosen |
| Skill | Reusable task procedure, specialized knowledge, tools, checks, and output contract | Applicability can be described and the workflow should load only when relevant | Persistent repository facts that every task must know; an independent persona |
| Agent | Isolated specialist role with its own context and deliverable | Delegation or independent review materially improves the work | A simple checklist or workflow that can run in the current context |

The neutral pack model has only `skill` and `agent`. It deliberately does not
define `rule` or `prompt`.

## Root instruction files are indexes by default

A root instruction file should usually contain:

- a one-sentence repository purpose or scope;
- non-negotiable invariants that apply to nearly every task;
- links and conditions that route work to narrower documents or skills;
- a short list of universal validation commands.

This is a default, not a prohibition on concrete rules. Put a detailed rule
directly in the root only when it is both broadly applicable and costly to
miss. Otherwise, move it to the narrowest applicable directory or document and
keep only a conditional pointer at the root.

A useful test is:

> Would omitting this instruction harm a normal task that does not use the
> specialized workflow?

If no, it should not consume root context.

This repository treats root `AGENTS.md` as the canonical routing layer.
`CLAUDE.md` and `GEMINI.md` are intentionally tiny native adapters that import
it, so repository-wide guidance has one editable source rather than three
copies.

## Skill patterns

A skill should define:

- a precise trigger description;
- inputs and assumptions;
- an ordered workflow with decision points;
- required tools or referenced assets;
- validation and completion criteria;
- failure and fallback behavior.

Use `invocation: manual` for command-like workflows that must never be selected
implicitly. Omit `invocation` (or use `both`) when autonomous selection is
safe.

Do not duplicate full repository policy inside a skill. Link to stable source
documents, and include only the operational context needed when the skill is
active.

## Agent patterns

An agent should have:

- one coherent specialty or review perspective;
- clear inputs and authority boundaries;
- an explicit output contract;
- completion criteria;
- no ownership of unrelated orchestration.

Prefer a skill when isolation is unnecessary. Prefer an agent when independent
context, parallel investigation, or a genuinely separate review perspective is
part of the desired behavior.

## Prompt and command migration

Custom prompts and commands are task entry points, not a separate knowledge
type. Model them as skills:

```yaml
kind: skill
invocation: manual
```

This maps to native explicit-only controls where the target supports them:

- Claude Code: `disable-model-invocation: true`
- Codex: `agents/openai.yaml` with
  `policy.allow_implicit_invocation: false`
- Gemini CLI: unsupported; disable Gemini for the entry or permit both modes

Always-on rule content should move to the appropriate repository instruction
file, not to a pack entry.

## Review checklist

- Is the guidance located at the narrowest scope where it is always relevant?
- Does root context route rather than duplicate?
- Is a reusable workflow a skill rather than an always-on instruction?
- Does an agent require isolation or a separate perspective?
- Is command-like behavior represented by a manual skill?
- Can every requirement be validated by an observable check?
