// Package gemini implements build and inventory operations for Gemini CLI.
//
// Skills are rendered to skills/<name>/SKILL.md with sibling assets, and
// agents to agents/<name>.md. User and project roots are $HOME/.gemini and
// <project>/.gemini respectively.
//
// Gemini has no per-skill metadata for disabling implicit invocation, so a
// neutral manual-only skill returns ErrUnsupportedSkillInvocation instead of
// silently weakening the policy.
//
// GEMINI.md and custom commands are not generated from knowledge packs.
package gemini
