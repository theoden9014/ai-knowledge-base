# Gemini CLI target

This document records the Gemini-specific mapping implemented by `knit`.

## Supported mappings

| neutral kind | artifact |
|---|---|
| `skill` | `skills/<name>/SKILL.md` plus sibling assets |
| `agent` | `agents/<name>.md` |

User artifacts are installed below `~/.gemini/`; project artifacts below
`<repo>/.gemini/`.

Skills and agents use YAML frontmatter plus Markdown. Neutral `name` and
`description` are generated first, then `tools.gemini.frontmatter` is merged
over them. Skill sibling assets are copied recursively.

## Invocation limitation

Gemini CLI supports explicit and autonomous skill activation, but currently
has no per-skill metadata equivalent to Claude's
`disable-model-invocation` or Codex's
`policy.allow_implicit_invocation`.

Therefore a skill with `invocation: manual` and Gemini enabled fails the build
with `ErrUnsupportedSkillInvocation`. Pack authors must either allow both
invocation modes or disable Gemini for that entry. Silently weakening a manual
policy is not permitted.

`GEMINI.md` and custom commands are intentionally outside the neutral pack
model. Repository instructions belong to repository governance; reusable
commands should be authored as skills where the target can honor the requested
invocation policy.

## References

- [Agent Skills](https://geminicli.com/docs/cli/using-agent-skills/)
- [Subagents](https://geminicli.com/docs/core/subagents/)
- [GEMINI.md](https://geminicli.com/docs/cli/gemini-md/)
