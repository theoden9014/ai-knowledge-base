// Package claude implements build and inventory operations for Claude Code.
//
// Skills are rendered to skills/<name>/SKILL.md with sibling assets, and
// agents to agents/<name>.md. User and project roots are $HOME/.claude and
// <project>/.claude respectively.
//
// Both formats use YAML frontmatter. Target frontmatter overrides neutral
// generated fields, except that a neutral manual invocation policy always
// produces disable-model-invocation: true for skills.
//
// Labels are stored separately under the .knit metadata tree selected by the
// CLI layer. Repository instruction files and custom commands are not
// generated from knowledge packs.
package claude
