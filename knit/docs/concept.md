# knit Conceptual Model

This document summarizes the main concepts of `knit` and their relationships. Implementation details such as package layout and interface signatures are covered in separate documents.

## Data (things)

| Concept | Meaning |
|---|---|
| **Pack** | A unit under `knowledge/<pack-name>/` that groups multiple Entries |
| **Entry** | An individual piece of knowledge inside a pack. It has one of `kind: skill / agent / rule / prompt` |
| **Target** | A destination AI tool for distribution (`claude` / `codex` / `gemini` ...) |
| **Scope** | Distribution scope (`user` / `project`) |
| **Artifact** | An intermediate representation converted into a Target-specific format |
| **Installation** | A locally placed Artifact plus an identifying Label |
| **Label** | Metadata used to identify an Installation as originating from `knit` |
| **Inventory** | A set of Installations for a given `(Scope, Target)`. The filesystem is the source of truth |

## Roles (actions)

The roles are divided into two groups: the source side and the distribution side.

### Source side

| Concept | Responsibility |
|---|---|
| **Loader** | Load Packs and Entries from `knowledge/` |
| **Validator** | Validate manifests and Entry frontmatter using JSON Schema |
| **Builder** | Convert Entry -> Target-specific Artifact |

### Distribution side - operating on Inventory

| Concept | Responsibility | Counterpart |
|---|---|---|
| **Installer** | Add an Artifact to Inventory (placement + labeling) | ← → Uninstaller |
| **Uninstaller** | Remove labeled Installations from Inventory | ← → Installer |
| **Lister** | Enumerate labeled Installations in Inventory | - |

## Main Relationships

```text
[Source side]                       [Inventory side]
Pack ──> Entry                      Inventory = { Installation* }
Entry ──> Builder ──> Artifact      Installation = Artifact + Label
                          │
                          └─> Installer ──> Inventory
                                              │
                                              ├─> Uninstaller
                                              └─> Lister
```

## Main Design Separations

- **Builders are replaceable per Target** and should align on a common interface
- **Loader / Validator are Target-independent**, since they belong to the neutral-format concerns
- **Installer / Uninstaller / Lister resolve paths according to Target and Scope**, but their file operations and Label logic can be shared
- `knit` does not maintain its own state file or DB; **the filesystem representation of Inventory is the source of truth**
