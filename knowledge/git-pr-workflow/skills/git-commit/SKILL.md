---
id: git-pr-workflow.skill.git-commit
kind: skill
name: git-commit
description: Git commit with Conventional Commits format. Use when user asks to commit changes, says "commit", "/commit", or after completing code changes when explicitly asked to commit.
tags: [git-pr-workflow, git, commit]
---

# Git Commit

Create commits following Conventional Commits format with project-appropriate scope.

## Workflow

1. Run `git status` and `git diff` to understand changes
2. Run `git log --oneline -10` to learn the project's commit style, scope granularity, and language
3. Stage relevant files with `git add`
4. Create commit with properly formatted message

## Commit Message Format

```
<type>(<scope>): <summary>

<body with bullet points>

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Type

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code change without feature/fix
- `test`: Adding/updating tests
- `chore`: Build, CI, dependencies

### Scope

Determine from project's commit history:
- Monorepo: microservice name (e.g., `api`, `web`, `auth-service`)
- Single app: module/component name (e.g., `auth`, `parser`, `ui`)
- Match the granularity used in recent commits

### Body

Write summary in natural language, then list specific changes:

```
feat(contract): Add subscription cancellation endpoint

Add endpoint to cancel active subscriptions with proration.

- Add CancelSubscription RPC to contract.proto
- Implement cancellation logic in usecase layer
- Add proration calculation for partial periods

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Language

Write the summary and body in the repository's standard language, inferred from recent `git log`. Keep the Conventional Commits type keyword (`feat`, `fix`, ...) in English regardless of the repository language.

## Safety Rules

- NEVER commit without reading `git status` and `git diff` first
- NEVER commit files containing secrets (.env, credentials, keys)
- NEVER use `--amend` unless explicitly requested
- NEVER use `--force` push
- NEVER skip hooks (`--no-verify`) unless explicitly requested
- Use HEREDOC for commit messages to preserve formatting

## Commit Command

```bash
git commit -m "$(cat <<'EOF'
feat(scope): Summary

Body with details.

- Change 1
- Change 2

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```
