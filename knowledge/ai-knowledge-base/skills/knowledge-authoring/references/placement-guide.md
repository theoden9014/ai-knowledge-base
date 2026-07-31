# Knowledge Placement Guide

Use this guide before designing or substantially restructuring pack entries in
ai-knowledge-base. Its purpose is to prevent Skills and Agents from absorbing responsibilities
that belong to persistent guidance, canonical documentation, runtime intent, or deterministic
enforcement.

## Contents

- Two-plane model
- Placement flow
- Persistent guidance
- Canonical documentation
- Runtime prompt
- Enforcement surfaces
- Product-specific distinctions
- Operational maintenance

## Two-plane model

Separate guidance from enforcement.

```text
Guidance
├── Persistent: AGENTS.md / CLAUDE.md
├── Scoped: nested instructions / Claude path-scoped rules
├── On demand: Skills and their references
└── Runtime intent: the current user prompt

Enforcement
├── permissions / sandbox / approvals
├── hooks / command-execution rules
└── schema / type checks / lint / tests / CI / branch protection
```

Do not present guidance as a guarantee. Move requirements that must never be violated to an
enforcement mechanism.

## Placement flow

Ask in this order:

1. **Can it be checked or blocked mechanically?**
   Use schema, types, lint, tests, CI, permissions, hooks, sandbox, or command-execution rules.
2. **Is it specific to the current request?**
   Put it in the runtime prompt.
3. **Is it a reusable workflow or knowledge needed only for certain tasks?**
   Put it in a Skill or a directly linked Skill reference.
4. **Is it needed for most work in a repository or subtree?**
   Put the smallest non-obvious constraint in the nearest persistent instruction file.
5. **Is it general project information useful to humans too?**
   Keep it in README, CONTRIBUTING, ADR, OpenAPI, or ordinary documentation. Let persistent
   guidance or a Skill link to that source.

When uncertain, prefer the canonical document and an on-demand Skill over duplicating detail
in always-loaded guidance.

## Persistent guidance

Use root or nested `AGENTS.md` and `CLAUDE.md` for concise, durable context that is:

- needed for most work in its scope
- specific to the repository or directory
- non-obvious from code and normal documentation
- advisory rather than mechanically enforceable

Good persistent content includes documentation routing, generated-file warnings, non-obvious
repository constraints, and a small number of universal verification expectations.

Do not put these in persistent guidance:

- complete architecture, setup, API, or release documentation
- long multi-step procedures
- current ticket details or temporary scope
- prohibitions that need deterministic enforcement
- duplicated commands or specifications maintained elsewhere

Apply both tests:

- **Inclusion test:** Without this line, would normal work in this scope frequently go wrong?
- **Deletion test:** If removing the line would not change most outcomes, remove or relocate it.

Keep the root instruction file as a compact router and minimal constraint set. Use the nearest
nested file for subtree-specific differences.

## Canonical documentation

Prefer these sources of truth:

| Information | Preferred source |
|---|---|
| Project overview and setup | README |
| Development and validation procedures | CONTRIBUTING or development docs |
| Architecture and decisions | Architecture docs or ADRs |
| Public API contract | OpenAPI or equivalent contract |
| Database and migration policy | Dedicated migration docs |
| Reusable operational workflow | Skill, optionally linking to detailed docs |

A Skill may instruct the agent to read a canonical source. It should not embed a second copy
that can drift.

## Runtime prompt

Keep one-off task intent in the runtime prompt:

- objective
- affected scope
- explicit non-goals
- temporary constraints
- completion criteria
- required deliverables
- Skill to invoke, when the user wants explicit selection

Promote instructions to a Skill only after they become a repeatable capability. Do not put
current issue numbers, branches, files, or acceptance details into a reusable Skill unless they
are parameters supplied at runtime.

## Enforcement surfaces

Choose the mechanism by what it can actually guarantee:

| Requirement | Mechanism |
|---|---|
| Validate artifact consistency | schema, lint, tests, or CI |
| Block a tool or file operation | permission, sandbox, or pre-tool hook |
| Run a deterministic action at an event | hook |
| Control sandbox-external command execution in Codex | Codex command-execution rules |
| Communicate conditional path guidance in Claude | Claude path-scoped instruction rules |
| Apply a human-judgment review procedure | Skill |

Do not use the generic word `rule` in a design without naming the product and mechanism. Codex
command-execution rules, Claude path-scoped instruction rules, and Claude permission rules have
different responsibilities.

## Product-specific distinctions

- Claude Custom Commands are a legacy authoring form merged into Skills. Existing files may
  work, but create new reusable workflows as Skills.
- Codex Custom Prompts have been removed from current Codex versions. Convert reusable prompts
  to Skills.
- Built-in slash commands are product UI operations, not neutral pack entry kinds.
- Skill discovery locations and product-specific metadata differ. Keep one neutral capability
  and let target builders produce the required layout.
- Do not assume nested instruction discovery behaves identically across products; verify the
  current target behavior.

## Operational maintenance

Start from observed needs instead of creating a comprehensive instruction system in advance:

```text
Observed failure
├── missing canonical information -> update docs
├── missing always-needed context -> update scoped persistent guidance
├── missing reusable procedure -> add or revise a Skill
├── missing deterministic control -> add enforcement
└── one-off ambiguity -> improve the runtime prompt
```

Periodically remove:

- guidance now inferable from code
- instructions now enforced mechanically
- duplicates of canonical documentation
- stale product-specific assumptions
- unused Skills
- descriptions that over-trigger or fail to trigger
- Agents whose isolation or specialization no longer adds value

Evaluate with representative tasks:

- compare outcomes with and without persistent guidance
- test positive and negative Skill triggers
- verify nested instructions load in the intended scope
- verify permissions and hooks actually block or run operations
- verify CI catches violations without relying on model compliance
- verify Agents improve results enough to justify context and coordination cost
