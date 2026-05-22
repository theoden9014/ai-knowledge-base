package gemini

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"time"

	"sigs.k8s.io/yaml"
)

// composeYAMLFrontmatter joins fm (as YAML) with body. Output format:
//
//	---\n<yaml>---\n<body>
//
// Keys are emitted in alphabetical order so repeated runs over the same
// input produce identical bytes. The underlying yaml.Marshal does not
// guarantee map iteration order, so we marshal one key at a time after
// sorting.
func composeYAMLFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("---\n")
	for _, k := range keys {
		single := map[string]any{k: fm[k]}
		out, err := yaml.Marshal(single)
		if err != nil {
			return nil, fmt.Errorf("gemini: marshal frontmatter key %q: %w", k, err)
		}
		buf.Write(out)
	}
	buf.WriteString("---\n")
	buf.Write(body)
	return buf.Bytes(), nil
}

// isTOMLEncodable recursively reports whether val is a Go type the prompt
// renderer can encode as TOML.
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
