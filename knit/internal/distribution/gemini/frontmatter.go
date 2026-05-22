package gemini

import (
	"bytes"
	"fmt"
	"reflect"
	"time"

	"sigs.k8s.io/yaml"
)

// composeYAMLFrontmatter joins fm (as YAML) with body. Output format:
//
//	---\n<yaml>---\n<body>
func composeYAMLFrontmatter(fm map[string]any, body []byte) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("yaml.Marshal failed: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlBytes)
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
