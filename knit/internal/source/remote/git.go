package remote

import "context"

// GitClient abstracts the subprocess boundary used to talk to the git
// binary. Production code uses the default exec.Command-based
// implementation; tests inject a fake to avoid depending on a real git
// installation and on network access.
type GitClient interface {
	// CloneShallow performs the equivalent of
	// "git clone --depth=1 <url> <dir>". The directory dir must already
	// exist and be empty. Implementations honor ctx for cancellation
	// and surface non-zero exit codes as ErrCloneFailed-wrapped errors.
	//
	// Why shallow-only: knit consumes the cloned repository read-only
	// at HEAD (it just needs the pack files to feed into source.Loader).
	// History, tags, branches, submodules, and LFS objects are not
	// required, and avoiding them keeps clones fast and bandwidth-cheap.
	// When future requirements (ref pinning, submodules) appear, this
	// interface should grow a new method rather than overloading
	// CloneShallow.
	CloneShallow(ctx context.Context, url, dir string) error
}

// NewDefaultGitClient returns a GitClient that invokes the system "git"
// binary via os/exec. The returned client is safe for concurrent use.
func NewDefaultGitClient() GitClient {
	return defaultGitClient{}
}
