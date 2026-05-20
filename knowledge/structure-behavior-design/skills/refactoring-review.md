---
id: structure-behavior-design.skill.refactoring-review
kind: skill
name: structure-behavior-design-refactoring-review
description: |
  Internal Structure-Behavior Design skill for post-implementation review and refactoring.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# Refactoring Review

## Goal

Review implementation for maintainability, responsibility placement, procedural-code risk, and test quality.

## Review for

- procedural transaction scripts
- long functions
- deep if/switch chains
- use case or application service obesity
- handler/controller decision logic
- misplaced behavior
- data-only models with behavior elsewhere
- primitive obsession
- hidden side effects
- infrastructure leakage
- large interfaces
- unnecessary abstraction
- magic numbers
- brittle tests
- tests that mirror implementation

## Refactoring rules

Do:
- move behavior closer to the concept/state it governs
- preserve observable behavior
- keep tests passing
- simplify before abstracting
- name concepts explicitly
- remove duplication when it represents the same rule

Do not:
- create Manager / Processor / Helper classes without clear ownership
- introduce patterns only to look object-oriented
- split code into many tiny classes without a change reason
- change behavior during refactoring

## Required output

## Review Findings

| Location | Issue | Principle | Severity | Suggested Fix |
|---|---|---|---|---|

## Procedural Risk

| Location | Smell | Better Owner | Refactoring Direction |
|---|---|---|---|

## Refactoring Plan

1.
2.
3.

## Tests to protect refactoring

-
