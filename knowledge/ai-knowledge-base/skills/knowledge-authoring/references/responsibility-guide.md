# Knowledge Pack Responsibility Guide

Use this guide during new pack design or substantial restructuring to decide where knowledge
belongs and how to divide an ai-knowledge-base pack.

## Contents

- Placement decision
- Skill responsibility
- Agent responsibility
- Invocation policy
- Pack boundary
- Dependency direction
- Portability boundary

## Placement decision

Ask the questions in order:

1. **Is this a one-off instruction for the current task?**
   Keep it in the runtime prompt or task context.
2. **Must this apply to every task in a repository or subtree?**
   Put the concise, durable convention in the nearest applicable `AGENTS.md`.
3. **Must compliance be deterministic?**
   Implement it through code, schema, tests, a linter, hook, permission, or administrative
   policy. Natural-language guidance is not enforcement.
4. **Is this reusable knowledge or a repeatable workflow loaded when relevant?**
   Create a skill.
5. **Does the work require an isolated specialist context, delegation, independent judgment,
   or restricted tools?**
   Create an agent, usually backed by one or more skills.

Do not force every useful instruction into a pack. Packs contain reusable capabilities, not
all configuration surfaces used by an AI coding tool.

## Skill responsibility

A skill owns a reusable capability. Typical patterns include:

- a repeatable task workflow
- domain or repository knowledge needed only for certain work
- a checklist with an observable output
- an orchestration procedure coordinating smaller capabilities
- templates, examples, scripts, and references needed by that capability

A skill should have:

- one recognizable job
- a description that defines positive and negative trigger boundaries
- explicit inputs, outputs, and stopping conditions
- portable core instructions
- progressive disclosure for detailed references

Create separate skills when capabilities have different triggers, can be reused independently,
or evolve on different schedules. Keep them together when splitting would create artificial
handoffs without independent value.

Do not create a skill for:

- a persona without a reusable workflow
- rules that must apply to every repository task
- the current ticket, change scope, or one-off completion criteria
- deterministic policy enforcement
- a single sentence that belongs in `AGENTS.md`
- a copy of README, CONTRIBUTING, OpenAPI, ADR, or other canonical documentation
- an encyclopedia formed by collecting related material without an executable purpose
- a provider-specific slash-command wrapper around an existing capability

## Agent responsibility

An agent owns an execution context or specialist judgment, not merely a document.

Create an agent when at least one of these is essential:

- isolation from the main conversation context
- independent review or adversarial evaluation
- parallel or delegated work
- a stable specialist role with a specific output contract
- a restricted model, tool set, permission boundary, or context budget
- preloading a deliberate subset of skills

Define:

- the role and decision authority
- required inputs
- the skills or context it may use
- prohibited work or tools
- the exact output contract
- when the caller should use its result

Do not create an agent for:

- an ordinary repeatable procedure that a skill can express
- a passive knowledge store or documentation bundle
- always-on repository guidance
- a one-off persona or writing tone
- a task that is merely long or has several steps
- work without a clear caller, input, or output contract

Use a skill unless context isolation, specialist judgment, or delegation changes the behavior.

## Invocation is not a kind

Manual versus automatic invocation is an orthogonal property of a skill.

| Intent | Neutral source | Target result |
|---|---|---|
| Allow implicit and explicit use | Omit `invocation` or use `invocation: both` | Default Skill behavior |
| Explicit/manual use only | `invocation: manual` | Builder emits the target's explicit-only control |

Keep the neutral meaning as “manual invocation” rather than introducing `command` or `prompt`
as a semantic kind or hand-authoring generated target metadata. Check every enabled target
before choosing `manual`: Claude and Codex can represent it, while the current Gemini builder
rejects it rather than silently weakening the policy.

## Pack boundary

Put entries in the same pack when they share:

- a domain and intended audience
- an installation and removal lifecycle
- a versioning and release cadence
- meaningful reuse or dependency relationships
- a coherent description understandable without listing unrelated features

Split a pack when entries merely share an author, target tool, file format, or repository
location. Avoid catch-all packs such as `utilities`, `misc`, or `prompts`.

A pack may contain:

- one independently useful skill
- several peer skills
- a self-contained public orchestrator alongside independently invocable skills
- skills plus specialist agents

## Dependency direction

The current neutral schema declares only Agent-to-Skill dependencies through `uses_skills`. It
does not declare Skill-to-Skill, Skill-to-Agent, or Agent-to-Agent dependencies. Do not describe
those relationships as required portable dependencies.

Before using `uses_skills`, verify that every enabled target preserves the relationship. If any
target drops it, either disable that target for the Agent or make the Agent self-contained
without relying on preloaded skills. A successful build is insufficient when required semantics
are discarded.

Keep peer Skills independently invocable. Put orchestration steps in a self-contained public
Skill and treat optional calls to other entries as runtime choices, not schema-enforced
dependencies. Avoid cycles, duplicated policy, and required dependencies expressed only in
prose.

## Portability boundary

Keep capability meaning and workflow in the neutral source. Isolate differences such as:

- invocation controls
- UI metadata
- model and tool declarations
- provider-specific agent fields
- destination paths

Use target overrides only when a target cannot derive the behavior from neutral semantics.
Do not mention provider-specific tool names in the core body unless the capability itself is
provider-specific.
