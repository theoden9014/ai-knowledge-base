// Package codex implements build and inventory operations for OpenAI Codex.
//
// Skills use logical paths below skills/<name>/ and are physically installed
// under $HOME/.agents or <project>/.agents. Custom agents use logical paths
// below agents/ and are installed under $CODEX_HOME (normally $HOME/.codex) or
// <project>/.codex. The package's ArtifactResolver performs this family-based
// routing while retaining one logical label namespace.
//
// Skills use YAML frontmatter and preserve sibling assets. A manual-only skill
// creates or merges agents/openai.yaml with
// policy.allow_implicit_invocation=false. Agents are emitted as TOML.
//
// Repository AGENTS.md files and legacy custom prompts are not generated from
// knowledge packs.
package codex
