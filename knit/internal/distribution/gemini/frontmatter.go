package gemini

import (
	"reflect"
	"time"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// frontmatterRenderer is the shared MarkdownFrontmatter configuration
// used by Gemini's skill and agent renderers. Gemini does not insert a
// blank line between the closing "---" and the body.
var frontmatterRenderer = source.MarkdownFrontmatter{}

// isTOMLEncodable recursively reports whether val is a Go type the
// prompt renderer can encode as TOML.
func isTOMLEncodable(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, time.Time:
		return true
	case []any:
		for _, el := range v {
			if !isTOMLEncodable(el) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, el := range v {
			if !isTOMLEncodable(el) {
				return false
			}
		}
		return true
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
