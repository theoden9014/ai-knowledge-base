# Repository guidance

This file is a routing layer for repository-wide work. Keep detailed,
use-case-specific procedures in the documents or skills linked below.

- For knowledge-pack authoring or review, read
  [`docs/authoring-guidelines.md`](docs/authoring-guidelines.md),
  [`docs/knowledge-format.md`](docs/knowledge-format.md), and the
  [`knowledge-authoring` skill](knowledge/ai-knowledge-base/skills/knowledge-authoring/SKILL.md).
- For `knit` development, read [`knit/README.md`](knit/README.md) and
  [`knit/docs/concept.md`](knit/docs/concept.md), then the relevant target
  document under `knit/docs/`.
- Validate Go and knowledge-format changes from `knit/` with `go test ./...`.
- Validate schema changes in both `schema/` and
  `knit/internal/source/schemas/`; these copies must remain equivalent.
- Do not commit generated local installs from Claude, Codex, or Gemini
  configuration directories.
