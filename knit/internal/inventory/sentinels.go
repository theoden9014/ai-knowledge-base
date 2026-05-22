package inventory

import (
	"errors"
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// SentinelMap declares the target-specific sentinel errors that wrap the
// neutral inventory sentinels exposed to CLI consumers. Each distribution
// package owns one constant SentinelMap value and feeds it into the thin
// wrappers around TransactionalInstaller / Uninstaller / Lister, so the
// "neutral error -> target-flavored error" translation lives in exactly
// one place per target instead of being copy-pasted into nine wrapper
// methods.
type SentinelMap struct {
	// InvalidArtifactPath replaces source.ErrInvalidArtifactPath (and the
	// equivalent PathPolicy violation surfaced through PathResolver).
	InvalidArtifactPath error

	// ProjectRootNotConfigured replaces inventory.ErrProjectRootNotConfigured.
	ProjectRootNotConfigured error

	// UnmanagedArtifactExists replaces inventory.ErrUnmanagedArtifactExists.
	UnmanagedArtifactExists error
}

// TranslateInstallError maps a neutral install error to the
// target-specific sentinel set declared in m. The artifactPath is woven
// into the resulting error message so CLI output can show which artifact
// the failure was about.
func (m SentinelMap) TranslateInstallError(err error, artifactPath string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, source.ErrInvalidArtifactPath):
		return fmt.Errorf("%w: %s", m.InvalidArtifactPath, artifactPath)
	case errors.Is(err, ErrProjectRootNotConfigured):
		return m.ProjectRootNotConfigured
	case errors.Is(err, ErrUnmanagedArtifactExists):
		return fmt.Errorf("%w: path=%s", m.UnmanagedArtifactExists, artifactPath)
	default:
		return err
	}
}

// TranslateUninstallError mirrors TranslateInstallError for the uninstall
// path. UnmanagedArtifactExists is excluded because Uninstall never
// produces it.
func (m SentinelMap) TranslateUninstallError(err error, artifactPath string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, source.ErrInvalidArtifactPath):
		return fmt.Errorf("%w: %s", m.InvalidArtifactPath, artifactPath)
	case errors.Is(err, ErrProjectRootNotConfigured):
		return m.ProjectRootNotConfigured
	default:
		return err
	}
}

// TranslateListError mirrors the above for the List path, which can only
// surface ProjectRootNotConfigured / InvalidScope on the input side.
func (m SentinelMap) TranslateListError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrProjectRootNotConfigured):
		return m.ProjectRootNotConfigured
	default:
		return err
	}
}
