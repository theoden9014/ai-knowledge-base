---
id: structure-behavior-design.skill.conceptual-modeling
kind: skill
name: structure-behavior-design-conceptual-modeling
description: |
  Internal Structure-Behavior Design skill for conceptual modeling and structure design.
  Use only as part of the Structure-Behavior Design workflow.
tags: [structure-behavior-design, internal]
tools:
  claude:
    enabled: true
    frontmatter:
      user-invocable: false
---

# Conceptual Modeling / Structure Design

## Goal

Model the problem area before designing packages, interfaces, or implementation.

Modeling is not limited to business domains.

Problem areas may include:
- business domain
- authentication
- authorization
- billing
- configuration
- caching
- retry
- API clients
- batch processing
- CLI tools
- infrastructure abstractions
- UI state
- testing utilities
- developer tooling

## Identify

- Concepts
- Relationships
- State
- Behavior
- Constraints
- Invariants
- Collaborators
- Inputs and outputs
- Stable parts
- Change-prone parts
- Boundaries that hide implementation details

## Avoid

- modeling only procedural steps
- creating only DTOs
- ignoring technical concepts
- hiding important state in primitive values
- treating use cases, handlers, or scripts as the only design units

## Required output

## Conceptual Model

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|

## Relationships

| Concept A | Relationship | Concept B | Notes |
|---|---|---|---|

## Structural Risks

- Missing concepts:
- Hidden state:
- Change-prone areas:
- Boundary candidates:
