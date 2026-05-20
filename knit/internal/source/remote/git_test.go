package remote

import (
	"testing"
)

// Note: defaultGitClient.CloneShallow is not unit-tested in this file
// because it shells out to the real "git" binary and would either depend
// on network access (to clone a real repository) or require a sophisticated
// PATH-shim setup. The exec.Command / ExitError wrap / ctx.Err preference
// paths are exercised by the integration tests in git_integration_test.go
// (build tag `integration`), which are opt-in and only run when a usable
// git binary is on PATH.

func TestNewDefaultGitClient(t *testing.T) {
	if c := NewDefaultGitClient(); c == nil {
		t.Fatal("NewDefaultGitClient() returned nil")
	}
}
