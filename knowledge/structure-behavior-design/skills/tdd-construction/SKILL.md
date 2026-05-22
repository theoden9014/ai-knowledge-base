---
id: structure-behavior-design.skill.tdd-construction
kind: skill
name: structure-behavior-design-tdd-construction
description: |
  Internal Structure-Behavior Design skill for TDD construction using Red-Green-Refactor.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# TDD Construction

## Goal

Implement behavior through Red-Green-Refactor.

Do not write production code before specifying the behavior with a failing test.

## Loop

For each behavior:

### Red

- Write one failing behavior test.
- The test should fail for the expected reason.
- Do not write production code first.

### Green

- Write the smallest production code needed to pass.
- Do not generalize early.
- Do not introduce extra abstractions.
- Keep the change focused.

### Refactor

- Improve names.
- Remove duplication.
- Move behavior to the object/module that owns it.
- Improve responsibility placement.
- Keep all tests passing.
- Do not change observable behavior.

## Avoid

- implementing multiple behaviors before tests
- writing broad abstractions before the second concrete need
- skipping refactoring after Green
- leaving behavior in handlers/use cases when a concept should own it
- making tests pass by weakening assertions

## Required output

## TDD Plan

| Behavior | Red Test | Green Implementation | Refactor Target |
|---|---|---|---|

## Construction Log

### Behavior 1

- Red:
- Green:
- Refactor:
- Tests run:
