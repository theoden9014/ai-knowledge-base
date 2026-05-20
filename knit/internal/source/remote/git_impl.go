package remote

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// defaultGitClient invokes the system "git" binary via os/exec.
type defaultGitClient struct{}

func (defaultGitClient) CloneShallow(ctx context.Context, url, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("%w: git exit status %d: %s", ErrCloneFailed, ee.ExitCode(), out)
		}
		return fmt.Errorf("%w: %v: %s", ErrCloneFailed, err, out)
	}
	return nil
}
