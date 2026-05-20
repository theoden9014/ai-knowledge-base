---
id: structure-behavior-design.skill.orchestrator
kind: skill
name: structure-behavior-design-orchestrator
description: |
  Use this skill for non-trivial software changes. It coordinates requirements specification,
  conceptual modeling, SOLID-guided responsibility assignment, interface design, test specification,
  TDD construction, and review to avoid procedural implementation.
tags: [structure-behavior-design, design, workflow]
tools:
  claude:
    enabled: true
---

# Structure-Behavior Design Orchestrator

## Goal

Guide non-trivial software development through structure design and behavior design before implementation.

Do not jump directly from requirement to code.

This workflow prevents:
- procedural transaction scripts
- overgrown handlers, use cases, or services
- misplaced responsibilities
- anemic models where behavior belongs with concepts
- excessive abstractions
- implementation-detail tests
- brittle interfaces

## Workflow

1. Requirements Analysis
2. Requirements Specification
3. Conceptual Modeling / Structure Design
4. SOLID-Guided Responsibility Assignment
5. Architectural Design
6. Interface Design
7. Test Specification / Behavior Design
8. Detailed Design
9. Test-Driven Construction
10. Review and Refactoring

## Invocation policy

Use this workflow for:
- feature additions
- behavior changes
- state transitions
- API/usecase/domain/application logic changes
- refactoring that changes responsibilities or boundaries
- authentication, authorization, billing, contract, cache, configuration, batch, or workflow logic
- changes spanning multiple files or modules

Do not use the full workflow for:
- typo fixes
- comment-only changes
- formatting-only changes
- trivial one-line changes
- changes that exactly follow an existing pattern without design judgment

## Risk-based process

### Low risk

For small changes that follow an existing design:

Required:
- Requirements Specification
- Test Specification
- TDD Construction
- Review

Recommended reviewers:
- structure-behavior-design-test-reviewer
- structure-behavior-design-code-quality-reviewer

### Medium risk

For changes involving new behavior, interfaces, or multiple files:

Required:
- Requirements Specification
- Conceptual Modeling
- SOLID Responsibility Assignment
- Interface Design
- Test Specification
- TDD Construction
- Review

Recommended reviewers:
- structure-behavior-design-solid-reviewer
- structure-behavior-design-interface-reviewer
- structure-behavior-design-test-reviewer
- structure-behavior-design-procedural-code-detector
- structure-behavior-design-code-quality-reviewer

### High risk

For public API, database schema, authentication, authorization, billing, contract, migration, or cross-module architecture changes:

Required:
- all workflow stages
- explicit design output before implementation
- human review before construction

Recommended reviewers:
- structure-behavior-design-requirements-reviewer
- structure-behavior-design-model-reviewer
- structure-behavior-design-solid-reviewer
- structure-behavior-design-architecture-reviewer
- structure-behavior-design-interface-reviewer
- structure-behavior-design-test-reviewer
- structure-behavior-design-procedural-code-detector
- structure-behavior-design-code-quality-reviewer

## Required output before implementation

Before writing production code, produce:

1. Requirement summary
2. Requirement specification
3. Conceptual model
4. Responsibility assignment table
5. SOLID risk assessment
6. Module/package boundary plan
7. Interface/signature proposal
8. Test specifications
9. Detailed design
10. TDD construction plan

## Structure design principle

Structure design defines:
- concepts
- relationships
- state
- constraints
- invariants
- responsibilities
- boundaries
- dependency direction
- contracts

## Behavior design principle

Behavior design is performed through tests.

Tests should specify:
- observable behavior
- rules
- state transitions
- invariants
- error cases
- edge cases
- boundary behavior

Tests should not mirror private implementation details.

## TDD construction principle

For each behavior:

1. Red: write one failing behavior test.
2. Green: write the smallest production code needed to pass.
3. Refactor: improve names, duplication, responsibility placement, and structure while keeping behavior unchanged.

## Review principle

After implementation, review for:
- procedural transaction-script style
- use case or handler obesity
- misplaced responsibility
- SOLID violations
- unnecessary abstraction
- oversized interfaces
- primitive obsession
- infrastructure leakage
- brittle tests
- missing behavior tests
