---
id: git-pr-workflow.skill.pr-review-fix
kind: skill
name: pr-review-fix
description: Address PR review feedback. Collect both unresolved inline comments and review-body comments on a GitHub PR, build a fix plan and turn it into tasks, then work through them one by one (assess -> fix -> commit -> push -> reply). Use when invoked as /pr-review-fix, when asked to "address the review comments" or "respond to PR feedback", or for follow-up work after a PR review.
tags: [git-pr-workflow, github, pull-request, review]
---

# PR Review Fix

Exhaustively collect PR review feedback, build a fix plan, turn it into tasks, then address each item one by one.

## Preconditions

- The target PR URL or PR number is available
- The local repository is checked out on the target PR's branch
- The `gh` CLI is authenticated

## Workflow

### 1. Collect all comments

Use the GraphQL API to fetch both **inline comments (reviewThreads)** and **review bodies (reviews)**:

```bash
cat <<'QUERY' | gh api graphql --input -
{"query": "query { repository(owner: \"{owner}\", name: \"{repo}\") { pullRequest(number: {pr_number}) { reviews(first: 50) { nodes { databaseId body state author { login } url } } reviewThreads(first: 100) { nodes { isResolved comments(first: 10) { nodes { id databaseId body path line author { login } url } } } } } } }"}
QUERY
```

### 2. Classify comments and build a fix plan

Classify the fetched comments into the following two categories and build a fix plan.

#### 2.1 Inline comments (reviewThreads)

- Only target threads where `isResolved: false`
- Record the file path and line number

#### 2.2 Review-body comments (reviews)

**By default, only target the latest review body** (if the user instructs you to address a specific review or all of them, follow that instead).

Steps to identify the latest review body:
1. Scan the fetched `reviews` from the end of the array and take the first review whose `body` is non-empty as the "latest review body"
2. Apply the following filters to that review:
   - Skip reviews whose `state` is `APPROVED` (look one further back)
   - Also skip when the body is purely praise or an approval message (e.g. "LGTM", "nice improvement")
3. Parse whether the body contains concrete feedback (fix requests, improvement suggestions, etc.)

#### 2.3 Turn the fix plan into tasks

After analyzing all comments, build a fix plan in the following format and turn it into tasks with TodoWrite:

```
## Fix Plan

### Inline comments
1. {file}:{line} - {summary of the feedback}
2. ...

### Feedback from review body
1. {review URL} - {summary of the feedback}
2. ...
```

**Important**: Present the fix plan to the user and get confirmation before starting work.

### 3. Address each item

Work through the tasked feedback one item at a time:

#### 3.1 Review the comment

- Confirm the file path and line number (for inline comments)
- Understand the feedback (read the surrounding code as well)

#### 3.2 Evaluate the validity of the feedback

Evaluate the validity of the feedback itself across the following three lenses **in order**. **As soon as a lens fails, stop evaluating and move on to the judgment in 3.3** (early rejection).

1. **Correctness of premises**
   - Does the reviewer's reading of the code match the actual behavior?
   - Are the cited lines / functions / variables / types understood correctly?
   - If a premise is wrong, skip the remaining lenses and treat it as an "invalid" candidate

2. **Type of argument**
   - Classify the feedback as "bug report / design improvement / convention compliance / matter of preference"
   - Gauge the strength of the argument (a bug report is strong, a matter of preference is weak)
   - Do not weigh a "matter of preference" the same as a "bug"

3. **Consistency with existing intent**
   - Does it break the design intent of the existing implementation or consistency with other parts?
   - Reference surrounding code and related past PRs/commit messages as needed

#### 3.3 Overall validity judgment

Based on the results of the three lenses above, make a final judgment using these three values. **Do not define a mechanical formula such as scoring or AND/OR; make an overall judgment informed by the lens results** (the lenses are not equal and have dependencies).

- **Valid**: fix as suggested
- **Partially valid**: fix only part of it, or propose an alternative
- **Invalid**: do not fix; politely explain the wrong premise or misreading

Keep the rationale of the judgment concisely articulated so it can be reused in the reply body (3.7).

#### 3.4 Edit files

If the judgment is "valid" or "partially valid", edit the relevant files. If "invalid", do not edit; skip 3.5-3.6 and go to 3.7 (reply).

#### 3.5 Commit

Write the commit message in the repository's standard language (infer it from `git log`); keep the Conventional Commits type keyword (`fix`, ...) in English.

```bash
git add <edited files>
git commit -m "fix: <summary of the feedback>

<detailed description of the change>

Co-Authored-By: Claude <noreply@anthropic.com>"
```

#### 3.6 Push

```bash
git push origin <branch-name>
```

#### 3.7 Reply to the inline comment

**Only for inline comments**, reply in thread form (set `in_reply_to` to the parent comment's `databaseId`):

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments -f body="Done.

<explanation of what was done>

Commit: https://github.com/{owner}/{repo}/commit/{commit_sha}" -F in_reply_to={parent_comment_database_id}
```

**Do not reply to review-body feedback.**

**Notes when writing the reply body:**

The reader of the reply is the reviewer, who has not seen the conversation between Claude and the user. Therefore write the reply body so it stands on its own with only the reviewer's comment as context.

Specifically:

- Write the reply in the repository's standard language (match the language of the reviewer's comment and the PR). The example bodies in this skill are illustrative; adapt their wording to that language
- Do not surface symbols/numbers used for organization in the conversation (e.g. "proposal (a)", "#1", "option 3"). The reviewer does not know what they refer to
- Do not cite terms absent from the reviewer's original comment as if the reviewer had provided them
- When you pick one of several options, reference it by restating its content, not by number (e.g. "I addressed this following the approach of moving the error message up to the caller" rather than "addressed per proposal (a)")

### 4. Next item

Repeat 3.1-3.7 until all items in the task list are complete.

## Prohibitions

- **Do not resolve**: never perform the resolve operation on feedback. The correct flow is for the reviewer to confirm and resolve.

## Output Format

After addressing each item, report in the following format:

### For an inline comment
```
## Item #{n}: {file}:{line}
**Type**: inline comment
**Feedback**: {summary of the comment}
**Validity**: {valid / partially valid / invalid} - {rationale (which lens: premise / argument / consistency)}
**Action**: {fixed / reason no action needed}
**Commit**: {commit URL}
**Reply**: done
```

### For review-body feedback
```
## Item #{n}: review body ({review URL})
**Type**: review body
**Feedback**: {summary of the feedback}
**Validity**: {valid / partially valid / invalid} - {rationale (which lens: premise / argument / consistency)}
**Action**: {fixed / reason no action needed}
**Commit**: {commit URL}
```

## When no action is needed

How to handle an "invalid" or "partially valid (with parts not addressed)" judgment from 3.3.

### For an inline comment
Always reply even when not fixing. Include the **concrete rationale** from the validity evaluation in 3.2 (wrong premise, weak argument, consistency with existing intent) in the reply body:

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments -f body="Thank you for the feedback.

<explanation of why no change was made (cite the rationale from the validity evaluation)>" -F in_reply_to={parent_comment_database_id}
```

### For review-body feedback
Only report the reason no action is needed; do not reply.
