package source

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedSchemas_inSyncWithRepoRoot verifies that the schema files
// embedded under internal/source/schemas/ are byte-identical to the
// canonical copies at the repository root (../../../schema/). The embed
// duplicates exist because Go's //go:embed cannot reach outside the
// package directory; this test prevents silent drift between editor
// linting (which uses the repo-root copies) and knit's build-time
// validation (which uses the embedded copies).
//
// If this test fails after editing a schema, copy the canonical file
// into internal/source/schemas/ so both locations stay in sync.
func TestEmbeddedSchemas_inSyncWithRepoRoot(t *testing.T) {
	// Locate the repository root from this test file. The path is
	// relative to internal/source/, which sits three levels below the
	// repo root: knit/internal/source -> knit -> ai-knowledge-base.
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cases := []struct {
		name     string
		embedded string
		canon    string
	}{
		{
			name:     "manifest schema",
			embedded: "schemas/manifest.schema.json",
			canon:    filepath.Join(repoRoot, "schema", "manifest.schema.json"),
		},
		{
			name:     "entry schema",
			embedded: "schemas/entry.schema.json",
			canon:    filepath.Join(repoRoot, "schema", "entry.schema.json"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			embedded, err := embeddedSchemas.ReadFile(tc.embedded)
			if err != nil {
				t.Fatalf("read embedded %s: %v", tc.embedded, err)
			}
			canon, err := os.ReadFile(tc.canon)
			if err != nil {
				t.Fatalf("read canonical %s: %v (is the working directory the package root?)", tc.canon, err)
			}
			if !bytes.Equal(embedded, canon) {
				t.Errorf(
					"schema drift: %s differs from %s.\n"+
						"Run: cp %s %s",
					tc.embedded, tc.canon, tc.canon,
					filepath.Join("internal", "source", tc.embedded),
				)
			}
		})
	}
}
