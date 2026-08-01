# knit Module Design

This document defines how the conceptual model described in [concept.md](./concept.md) should be arranged as Go packages. Concrete signatures and type definitions remain the responsibility of the code.

## Package Structure

```text
knit/
├── main.go                       # Entry point (thin; only calls internal/cli)
├── go.mod                        # module github.com/theoden9014/ai-knowledge-base/knit
├── go.sum
├── README.md
├── docs/
└── internal/
    ├── cli/                      # Subcommand tree
    ├── source/                   # Source-side framework
    ├── inventory/                # Distribution-side framework
    └── distribution/             # Target-specific concrete implementations
        ├── claude/
        ├── codex/
        └── gemini/
```

## Responsibilities of Each Directory

| Directory | Role |
|---|---|
| `cli` | Defines and assembles the subcommand tree. It combines `source`, `inventory`, and `distribution` to implement subcommand behavior |
| `source` | **Source-side framework**. Defines neutral data types and source-side interfaces. Target-independent shared processing such as Loader and Validator is implemented here |
| `inventory` | **Distribution-side framework**. Defines destination-side data types and interfaces. Target-independent transaction, label, and artifact-resolution contracts are implemented here |
| `distribution/<target>` | **Concrete implementation for each Target**. Target-specific implementations of Builder, Installer, Uninstaller, and Lister live together here. Knowledge specific to each Target, such as distribution paths, format conversion, and Labeling, is consolidated into a single package |

## Ownership of Data Types

| Concept | Package |
|---|---|
| Pack, Entry, Kind, Target, Artifact | `source` |
| Installation, Label, Scope, ArtifactResolver | `inventory` |

The `Target` type belongs to the `source` domain because it represents the Target returned by the Builder, and `inventory` refers to the `source.Target` type.

## Ownership of Roles (interfaces)

| Role | Interface definition | Implementation |
|---|---|---|
| Loader | `source` | `source` (Target-independent) |
| Validator | `source` | `source` (Target-independent) |
| Builder | `source` | `distribution/<target>` |
| Installer | `inventory` | `distribution/<target>` |
| Uninstaller | `inventory` | `distribution/<target>` |
| Lister | `inventory` | `distribution/<target>` |

## Dependency Direction

```text
cli ──> (source, inventory, distribution)
distribution/<target> ──> (source, inventory)
inventory ──> source                      (references the Target type)
```

- `source` depends on nothing else and is the lowest layer
- `inventory` only refers to the `Target` type from `source`
- `distribution/<target>` implements interfaces from both `source` and `inventory`
- `cli` is the top-level layer, where concrete implementation selection, including which Target to use, is performed

## Design Effects

- **All Target-specific code is centralized in `distribution/<target>`**. Because Builder conversion logic and Installer placement logic live together per Target, changes for that Target stay within one package
- **Logical artifact paths are independent of physical roots**. A target may route artifact families to different roots through `inventory.ArtifactResolver` (Codex routes skills to `.agents` and agents to `.codex`)
- **Adding a new Target only requires creating `distribution/<new-target>/` and implementing `source.Builder` plus `inventory.Installer` / `Uninstaller` / `Lister`**. `source` and `inventory` do not need to change
- **`source` and `inventory` remain stable as frameworks**. They are not affected by Target-specific differences
