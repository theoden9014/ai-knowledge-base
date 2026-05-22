package gemini

import (
	"bytes"
	"fmt"

	"github.com/pelletier/go-toml/v2"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type promptRenderer struct{}

func (promptRenderer) Kind() source.Kind { return source.KindPrompt }

// Render encodes the prompt as commands/<name>.toml. The neutral body is
// embedded under "prompt", description under "description", and any
// tools.gemini.frontmatter keys are merged as TOML top-level entries with
// last-write-wins semantics. Returns ErrUnsupportedFrontmatterValue when
// a value cannot be expressed in TOML.
func (promptRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
	v := map[string]any{
		"prompt": string(e.Body),
	}
	if e.Description != "" {
		v["description"] = e.Description
	}
	if cfg, ok := e.Tools[Target]; ok {
		for k, val := range cfg.Frontmatter {
			if val == nil {
				delete(v, k)
				continue
			}
			if !isTOMLEncodable(val) {
				return source.Artifact{}, fmt.Errorf("%w: prompt %q: unsupported value type %T at key %q", ErrUnsupportedFrontmatterValue, e.ID, val, k)
			}
			v[k] = val
		}
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return source.Artifact{}, fmt.Errorf("%w: prompt %q: %v", ErrUnsupportedFrontmatterValue, e.ID, err)
	}
	return source.Artifact{
		Target:         Target,
		Path:           "commands/" + e.Name + ".toml",
		Content:        buf.Bytes(),
		SourceEntryIDs: []string{e.ID},
	}, nil
}
