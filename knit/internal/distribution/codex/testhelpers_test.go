package codex

// must is a tiny generic helper for tests: it returns v when err is nil
// and panics otherwise.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
