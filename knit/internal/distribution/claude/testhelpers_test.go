package claude

// must is a tiny generic helper for tests: it returns v when err is nil
// and panics otherwise. Use it to keep test fixtures concise when calling
// constructors whose only realistic failure mode is a programmer error in
// the test itself.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
