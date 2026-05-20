package source

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

//go:embed schemas/manifest.schema.json schemas/entry.schema.json
var embeddedSchemas embed.FS

// NewValidator returns a Validator that validates against the JSON Schema
// documents embedded under internal/source/schemas/. The returned Validator
// is safe for concurrent use.
func NewValidator() (Validator, error) {
	manifestSchema, err := loadSchema("schemas/manifest.schema.json")
	if err != nil {
		return nil, fmt.Errorf("source: load manifest schema: %w", err)
	}
	entrySchema, err := loadSchema("schemas/entry.schema.json")
	if err != nil {
		return nil, fmt.Errorf("source: load entry schema: %w", err)
	}
	return &jsonSchemaValidator{
		manifestSchema: manifestSchema,
		entrySchema:    entrySchema,
	}, nil
}

func loadSchema(path string) (*jsonschema.Schema, error) {
	raw, err := embeddedSchemas.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(path, doc); err != nil {
		return nil, err
	}
	return c.Compile(path)
}

type jsonSchemaValidator struct {
	manifestSchema *jsonschema.Schema
	entrySchema    *jsonschema.Schema
}

func (v *jsonSchemaValidator) ValidateManifest(raw []byte) error {
	doc, err := yamlToAny(raw)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSchemaViolation, err)
	}
	if err := v.manifestSchema.Validate(doc); err != nil {
		return fmt.Errorf("%w: %s", ErrSchemaViolation, err)
	}
	return nil
}

func (v *jsonSchemaValidator) ValidateEntryFrontmatter(raw []byte) error {
	doc, err := yamlToAny(raw)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSchemaViolation, err)
	}
	if err := v.entrySchema.Validate(doc); err != nil {
		if isInvalidKindError(doc, err) {
			return fmt.Errorf("%w: %s", ErrInvalidKind, err)
		}
		return fmt.Errorf("%w: %s", ErrSchemaViolation, err)
	}
	return nil
}

// yamlToAny parses a YAML document into the untyped Go form expected by
// jsonschema (map[string]any / []any / scalar).
func yamlToAny(raw []byte) (any, error) {
	j, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(j, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// isInvalidKindError reports whether the validation error stems from a kind
// field whose value is outside the recognized set. The contract documented
// on Validator says such failures are wrapped around ErrInvalidKind so that
// callers can branch with errors.Is.
func isInvalidKindError(doc any, schemaErr error) bool {
	obj, ok := doc.(map[string]any)
	if !ok {
		return false
	}
	kindVal, ok := obj["kind"]
	if !ok {
		return false
	}
	kindStr, ok := kindVal.(string)
	if !ok {
		// kind is present but not a string; schema reports it but it is
		// a generic schema violation, not an "unknown kind" case.
		return false
	}
	if Kind(kindStr).IsValid() {
		return false
	}
	// Confirm that the validator complained about the kind property.
	var ve *jsonschema.ValidationError
	if !errors.As(schemaErr, &ve) {
		return false
	}
	return containsKindCause(ve)
}

// containsKindCause reports whether the validation error (or any of its
// recursive causes) points at the root-level "kind" property. The
// InstanceLocation must be exactly ["kind"] to qualify; deeper paths that
// merely contain "kind" as a segment (for example a future
// properties.<x>.properties.kind) are intentionally not matched, so that
// extending the schema does not cause this detector to misfire.
func containsKindCause(ve *jsonschema.ValidationError) bool {
	if len(ve.InstanceLocation) == 1 && ve.InstanceLocation[0] == "kind" {
		return true
	}
	for _, child := range ve.Causes {
		if containsKindCause(child) {
			return true
		}
	}
	return false
}
