---
id: git-pr-workflow.skill.pr-create
kind: skill
name: pr-create
description: Create GitHub Pull Request with project's PR template. Use when user asks to create PR, says "create PR", "/pr-create", or after completing implementation when asked to open a pull request.
tags: [git-pr-workflow, github, pull-request]
---

# Create Pull Request

Create PRs following project's template with clear explanations.

## Workflow

1. Verify branch (NEVER work on the default branch)
2. Check change size and suggest splitting if needed
3. Design PR granularity (decide what to include in this PR)
4. Commit changes for this PR (delegate to git-commit skill)
5. Find and read project's PR template
6. Check past PR titles for consistency
7. Analyze changes with `git diff` and `git log`
8. Push to remote and create PR using `gh pr create`

## Step 1: Verify Branch

**CRITICAL: Always check the current branch first. Never create a PR from the default branch.**

```bash
git branch --show-current
```

### Branch naming convention

Format: `feature/<microservice-or-component>/<descriptive-name>`

- If the change belongs to a specific microservice or component, include it (e.g., `feature/contract/add-coupon-transfer`)
- If no specific microservice or component applies, omit it (e.g., `feature/update-ci-config`)

### If on the default branch

```bash
# 1. Stash uncommitted changes with a descriptive name
git stash save "pr-create: uncommitted changes before branch switch"

# 2. Verify the stash was saved
git stash list
# Confirm the stash entry appears at the top (stash@{0})

# 3. Update the default branch to latest remote
DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | cut -d ':' -f 2 | xargs)
git fetch origin "$DEFAULT_BRANCH"
git merge origin/"$DEFAULT_BRANCH"

# 4. Create and switch to feature branch
git checkout -b feature/<appropriate-branch-name>

# 5. Restore stashed changes by specifying the exact stash entry
git stash apply stash@{0}

# 6. Verify changes are restored, then drop the stash
git status
git stash drop stash@{0}
```

### If already on a feature branch

Check if a PR already exists for the current branch:

```bash
gh pr view --json number,title,url,state 2>/dev/null
```

- **If a PR already exists**: Inform the user and ask how to proceed:
  - Update the existing PR (push additional commits to the same branch)
  - Create a new branch from the current branch for a separate PR
- **If no PR exists**: Proceed to the next step.

## Step 2: Check Change Size

Run `git diff --stat` and `git status` to assess uncommitted changes.

**Suggest splitting when:**
- More than ~400 lines changed
- Multiple unrelated features/fixes
- Changes span multiple domains (e.g., API + UI + DB)

## Step 3: Design PR Granularity

If changes are small and focused, proceed as a single PR.

If splitting is needed, propose logical boundaries to the user and get explicit confirmation before proceeding with any splitting or further actions:
```
This change is large (~800 lines). Consider splitting into separate PRs:
1. PR1: Database schema changes
2. PR2: API endpoint implementation
3. PR3: Frontend integration

Which files should be included in this PR?
```

After agreement, only stage and commit the files relevant to this PR.

## Step 4: Commit Changes

**Delegate to the git-commit skill.** Do NOT run git commit yourself:

```
Skill tool: skill="git-commit"
```

If all changes are already committed, skip this step.

## Step 5: Find PR Template

**IMPORTANT**: When using Glob tool, always specify the repository root as the path.
If working in a subdirectory, searching from current directory will miss templates.

```bash
# First, find the repository root
git rev-parse --show-toplevel

# Then use Glob tool with repository root as path parameter
```

Search patterns:
- `.github/pull_request_template.md` (lowercase, most common)
- `.github/PULL_REQUEST_TEMPLATE.md` (uppercase)
- `.github/PULL_REQUEST_TEMPLATE/*.md` (directory)

If no template found, use the default format below.

## Step 6: Check Past PR Titles

Run `gh pr list --state merged --limit 10` to see recent PR titles.

Observe and match:
- Language (Japanese/English)
- Prefix style (`feat:`, `[Add]`, etc.)
- Scope format (`feat(api):` vs `feat/api:`)
- Capitalization

Write both the PR title and the PR body in the repository's standard language (the one used in recent PRs and the template). Do not default to English when the repository's PRs are written in another language.

## Step 7: Write PR Description

**If a project PR template was found in Step 5, always follow that template's structure.**
The structure and example below are only used when no project template exists.

### Top priority rules (apply these before anything else)

1. **Match length to change size.** A few-line change gets a few-line description. Do NOT pad a small change with long background, multiple tables, exhaustive "out of scope" lists, or every checklist item filled. Over-explaining a trivial change makes it *harder* to review, not easier.
2. **Never write what the diff already shows.** Skip implementation mechanics — variable/function names, "added via `concat`", which line was edited. State *what* changed and *why*, in the reader's terms, not *how* it was coded. The reviewer reads the diff for the how.
3. **Omit non-applicable template sections.** Drop them, or collapse to one line. Do not fill a section just because it exists in the template.

### Structure

Follow the project's template. For each section, write only what a reviewer **cannot** get from reading the diff:

- **What was done**: 1-2 sentences in domain terms (not code mechanics)
- **Why / background**: the problem or motivation, when it isn't obvious from the change itself
- Other sections (spec, out-of-scope, review-focus): include only when there is something real to say

### Sizing guide

- **A few lines ~ 10 lines changed**: "What" + "Why" in 1-2 sentences each. Skip everything else.
- **100+ lines / multiple domains**: prose background, tables, and detailed sections are appropriate.

### Example (small change — a config edit adding internal IPs to an allow list)

This example is written in Japanese to illustrate the rule above: write the PR body in the repository's standard language. Match your own repository's language instead of copying Japanese.

```markdown
## 対応したこと
検証環境のエンドポイントのIP許可リストに、社内IPなどのCIDRの範囲を追加

## この対応を行なった背景・経緯・理由
社内から検証環境の動作確認ができるようにするため

## スコープ外
- 本番への影響なし（対象設定は検証環境のみ有効）
```

Note what is **absent**: no mention of *how* it was coded (which variable was reused, `concat`, the edited line). The diff already shows that. Sections with nothing real to say are omitted entirely.

## Step 8: Push and Create PR

Always create PRs as **draft** by default.

```bash
# Push to remote
git push -u origin <branch-name>

# Create draft PR
gh pr create --draft --title "[title matching project style]" --body "$(cat <<'EOF'
[PR description following template]
EOF
)"
```

## Default Template (when no project template exists)

```markdown
## Overview
[1-2 sentence summary understandable by someone with no domain knowledge]

## Background
[Current problem or business motivation — why this change is needed]

## Changes
[Prose explanation of changes grouped by concern, with design rationale]

## Testing
[How this was tested]
```
