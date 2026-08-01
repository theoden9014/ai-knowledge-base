# Repository guidance

This file is a routing layer for repository-wide work. Keep detailed,
use-case-specific procedures in the documents or skills linked below.

- For knowledge-pack authoring or review, read
  [`docs/authoring-guidelines.md`](docs/authoring-guidelines.md) and
  [`docs/knowledge-format.md`](docs/knowledge-format.md).
- For `knit` development, read [`knit/README.md`](knit/README.md) and
  [`knit/docs/concept.md`](knit/docs/concept.md), then the relevant target
  document under `knit/docs/`.
- Validate Go changes from `knit/` with `go test ./...`.
- Validate schema changes in both `schema/` and
  `knit/internal/source/schemas/`; these copies must remain equivalent.
