---
id: structure-behavior-design.skill.interface-design
kind: skill
name: structure-behavior-design-interface-design
description: |
  Internal Structure-Behavior Design skill for interface, signature, and contract design.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# Interface Design

## Goal

Design clear boundaries and contracts before implementation.

## Design principles

Interfaces should be:
- small
- consumer-oriented
- explicit
- testable
- based on problem-area vocabulary
- independent from infrastructure details

## Design before implementation

For each interface, function, or method, define:

- purpose
- consumer
- inputs
- outputs
- error contract
- side effects
- example call site
- implementation detail that must remain hidden

## Avoid

- vague names such as Process, Execute, Handle when a domain-specific name exists
- too many primitive parameters
- boolean flags that switch behavior
- infrastructure DTOs leaking into core interfaces
- large interfaces built around providers instead of consumers
- test-only abstractions with no real boundary

## Required output

## Proposed Interfaces / Signatures

| Name | Consumer | Responsibility | Signature | Error Contract |
|---|---|---|---|---|

## Example Call Sites

```language
// show intended usage here
```

## Boundary Decisions

| Boundary | Hidden detail | Reason |
|---|---|---|

## Interface Risks

- Oversized interfaces:
- Primitive obsession:
- Infrastructure leakage:
- Boolean flag risks:
