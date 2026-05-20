package remote

import "errors"

// Sentinel errors returned by the remote package. Implementations should
// wrap these with fmt.Errorf so callers can match with errors.Is.
var (
	// ErrInvalidLocator indicates that a string supplied to Parse does
	// not match the expected "<host>/<owner>/<repo>[/<subpath>...]"
	// shape.
	ErrInvalidLocator = errors.New("remote: invalid locator")

	// ErrUnsupportedHost indicates that no registered Fetcher claimed
	// support for the Locator's host. Callers should surface this as a
	// configuration error rather than a transient failure.
	ErrUnsupportedHost = errors.New("remote: unsupported host")

	// ErrCloneFailed indicates that the underlying GitClient reported a
	// non-zero exit code or otherwise failed to materialize the repo.
	ErrCloneFailed = errors.New("remote: clone failed")

	// ErrCleanupFailed indicates that releasing the temporary directory
	// backing a FetchedPack failed. The directory may still exist on
	// disk after a Close call that returns this error.
	ErrCleanupFailed = errors.New("remote: cleanup failed")
)
