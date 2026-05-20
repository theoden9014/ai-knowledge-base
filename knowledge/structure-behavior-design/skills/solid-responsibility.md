---
id: structure-behavior-design.skill.solid-responsibility
kind: skill
name: structure-behavior-design-solid-responsibility
description: |
  Internal Structure-Behavior Design skill for SOLID-guided responsibility assignment.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# SOLID-Guided Responsibility Assignment

## Goal

Assign responsibilities before implementation using SOLID as design guidance.

Do not use SOLID to create unnecessary abstractions.

## SRP

Single Responsibility Principle means separating reasons to change.

Do:
- group responsibilities that change for the same reason
- separate responsibilities that change for different reasons
- identify the actor or change driver behind each responsibility

Do not:
- interpret SRP as one tiny class per action
- create excessive one-method classes
- hide procedural code in Manager, Processor, Helper, or Util objects
- split code only by technical steps when the change reason is the same

## OCP

Open/Closed Principle means isolating real variation.

Do:
- introduce extension points when variation already exists or is clearly expected
- keep unstable decisions localized

Do not:
- create Strategy, Factory, Plugin, or Adapter abstractions for a single implementation without real variation
- replace simple conditionals with many classes unless it improves changeability

## LSP

Liskov Substitution Principle means implementations preserve the expected contract.

Do:
- define preconditions, postconditions, and error behavior
- ensure implementations are substitutable

Do not:
- require callers to know concrete implementation details
- silently weaken guarantees in one implementation

## ISP

Interface Segregation Principle means consumers should depend only on what they use.

Do:
- design small consumer-oriented interfaces
- split read/write or command/query interfaces when needed

Do not:
- create broad Repository or Service interfaces
- force consumers to depend on unused methods

## DIP

Dependency Inversion Principle means high-level policy should not depend on low-level details.

Do:
- hide frameworks, DBs, SDKs, queues, HTTP, and filesystem details behind boundaries
- keep core behavior independent from infrastructure

Do not:
- leak DB DTOs, HTTP DTOs, or SDK models into core behavior
- let domain or application logic depend directly on external SDKs

## Required output

## Responsibility Assignment

| Responsibility | Owner | Reason to change | SOLID concern | Not owner | Reason |
|---|---|---|---|---|---|

## SOLID Risk Assessment

| Principle | Risk | Mitigation |
|---|---|---|

## Procedural Risk

- Rules at risk of being placed in handlers/use cases:
- Behavior that should move closer to state:
- Abstractions that may be premature:
