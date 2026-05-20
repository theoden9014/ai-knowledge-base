# AI Knowledge Base

An open repository for managing reusable AI coding knowledge across tools such as Claude Code, Codex CLI, and Gemini CLI.

The repository keeps prompts, skills, agents, and always-on rules in a tool-neutral source format under `knowledge/`, then converts and installs them into each tool's local configuration layout through [`knit`](./knit/README.md).

## What This Repository Is For

- Keep knowledge in one source of truth
- Reuse the same workflows across multiple AI coding tools
- Absorb tool-specific format and directory differences through builders
- Make local installation reproducible and maintainable

## How It Works

```text
knowledge/   ->   tool-neutral source
      | 
      v
knit        ->   build and install for each target tool
      |
      v
tool config directories such as ~/.claude/ and ~/.gemini/
```

This repository uses four tool-neutral kinds:

| kind | purpose |
| ---- | ------- |
| `skill` | Reusable procedures, viewpoints, and output formats for a task |
| `agent` | A specialist role, such as a reviewer operating in an isolated context |
| `rule` | Instructions that should always apply |
| `prompt` | Reusable prompts or slash-command style entry points |

These kinds are internal abstractions. Each target tool may map them differently.

## Repository Layout

```text
ai-knowledge-base/
├── README.md
├── docs/                 # format and design documentation
├── knowledge/            # canonical editable source
│   └── <pack-name>/
│       ├── manifest.yaml
│       ├── skills/
│       ├── agents/
│       ├── rules/
│       └── prompts/
├── knit/                 # build/install tool
└── schema/               # validation schemas
```

The content model is documented in [docs/knowledge-format.md](./docs/knowledge-format.md).

## Tool Support

Current implementation work is centered on Claude Code, Codex CLI, and Gemini CLI.

Support is intentionally builder-driven: the source format stays stable while each target decides how that source is rendered and installed.

## Why Keep a Tool-Neutral Source?

The same workflow often needs to run on more than one AI coding tool. If each tool had its own copy of the same knowledge, maintenance would quickly diverge. A neutral source keeps the content authored once and lets builders handle per-tool translation.

## Contributing

Issues and pull requests are welcome.

Good contributions include:

- New knowledge packs
- Improvements to existing packs
- New target-tool builders
- Validation, packaging, and installation improvements in `knit`
- Documentation fixes and clarifications

When contributing knowledge content, keep it tool-neutral unless a target-specific override is strictly necessary.

## License

This repository uses different licenses for code and knowledge content.

| area | license | scope |
| ---- | ------- | ----- |
| Code and tooling | [MIT License](./LICENSE) | `knit/`, `docs/`, `schema/`, `README.md`, and other non-`knowledge/` files |
| Knowledge packs | [CC BY-SA 4.0](./knowledge/LICENSE) ([deed](https://creativecommons.org/licenses/by-sa/4.0/)) | all content under `knowledge/` |

If you modify or redistribute knowledge packs, follow the attribution and share-alike requirements of CC BY-SA 4.0.
