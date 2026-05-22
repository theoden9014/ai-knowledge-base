---
id: structure-behavior-design.skill.test-specification
kind: skill
name: structure-behavior-design-test-specification
description: |
  Internal Structure-Behavior Design skill for tests-as-specification and behavior design.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# Test Specification / Behavior Design

## Goal

Use tests to specify behavior before implementation.

Tests are executable specifications.

## Specify

- observable behavior
- rules
- state transitions
- invariants
- error cases
- edge cases
- boundary behavior

## Prefer

- Given / When / Then
- behavior-oriented test names
- tests that describe requirements
- tests that drive interface design
- tests that are stable under refactoring

## Avoid

- private method tests
- getter/setter-only tests
- excessive mocks
- tests coupled to implementation details
- tests that lock in procedural structure
- tests that only verify internal call order

## Design feedback

If tests are hard to write or read, reconsider:

- interface design
- responsibility assignment
- type design
- module boundaries
- excessive coupling
- missing concepts

Do not compensate for poor design with excessive mocking.

## Required output

## Test Specifications

| Behavior | Given | When | Then | Test Level | Notes |
|---|---|---|---|---|---|

## Invariant Tests

| Invariant | Example | Expected Result |
|---|---|---|

## Error / Edge Case Tests

| Case | Given | When | Then |
|---|---|---|---|

## Testability Feedback

- Interface concerns:
- Responsibility concerns:
- Coupling concerns:
