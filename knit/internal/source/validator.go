package source

// Validator validates the neutral knowledge format against the JSON Schema
// documents under schema/. It is split into two methods so that the manifest
// and the per-entry frontmatter can be checked independently as they are
// encountered during loading.
//
// Validator accepts the raw bytes of the YAML/JSON source so that schema
// errors can be reported with the original document positions intact.
//
// Validator is typically composed into a Loader via NewLoader rather than
// invoked directly by orchestrating code, but it is exposed as a standalone
// interface so it can be reused outside the loading pass (for example, by
// an editor integration that only wants to validate frontmatter).
type Validator interface {
	// ValidateManifest validates the raw bytes of a manifest.yaml against
	// schema/manifest.schema.json. Schema violations are returned wrapped
	// around ErrSchemaViolation.
	ValidateManifest(raw []byte) error

	// ValidateEntryFrontmatter validates the raw bytes of an entry's YAML
	// frontmatter block against schema/entry.schema.json. Schema
	// violations are returned wrapped around ErrSchemaViolation, except
	// that a kind value outside the recognized set is reported wrapped
	// around ErrInvalidKind so callers can distinguish "unknown kind"
	// from other schema failures with errors.Is.
	ValidateEntryFrontmatter(raw []byte) error
}
