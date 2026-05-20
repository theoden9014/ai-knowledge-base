package codex

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// Compile-time guarantee that the inventory package is imported. The
// SidecarLabelStore reference inside the helpers keeps it linked even when a
// test build skips the labeled helpers.
var _ = (*inventory.SidecarLabelStore)(nil)

// tomlAssign detects either `key = "value"` or `key = 'value'` in TOML text.
// pelletier/go-toml v2 may choose either quote style depending on the
// implementation, so the tests accept both.
func tomlAssign(content, key, value string) bool {
	double := fmt.Sprintf(`%s = "%s"`, key, value)
	single := fmt.Sprintf(`%s = '%s'`, key, value)
	return strings.Contains(content, double) || strings.Contains(content, single)
}

// newTempRoots is a helper that creates and returns user/project inventory
// roots and a LabelStore wired against a temp filesystem. Label sidecars live
// under a separate knit root, mirroring how the cli factory composes them.
func newTempRoots(t *testing.T) (userRoot, projectRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user")
	projectRoot = filepath.Join(base, "project")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	projectLabelsRoot := filepath.Join(base, "project", ".knit", "labels")
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, projectLabelsRoot)
	return userRoot, projectRoot, labels
}

// newTempRootsUserOnly returns a fixture where projectRoot is intentionally
// empty so callers can exercise ErrProjectRootNotConfigured paths.
func newTempRootsUserOnly(t *testing.T) (userRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, "")
	return userRoot, labels
}

// makeSampleArtifact returns a sample Artifact shared by contract tests.
// Its installation path is deterministic, so installing it twice causes a
// collision by design.
func makeSampleArtifact() source.Artifact {
	return source.Artifact{
		Target:         Target,
		Path:           "skills/sample/SKILL.md",
		Content:        []byte("---\nname: sample\ndescription: sample skill\n---\nbody\n"),
		SourceEntryIDs: []string{"demo.skill.sample"},
	}
}

// seedInstall creates and returns one existing Installation in the given
// scope. It is used by the Uninstaller/Lister contract tests.
func seedInstall(t *testing.T, inst *Installer, scope inventory.Scope, artifactPath string) inventory.Installation {
	t.Helper()
	a := source.Artifact{
		Target:         Target,
		Path:           artifactPath,
		Content:        []byte("---\nname: seeded\ndescription: seeded\n---\nbody\n"),
		SourceEntryIDs: []string{"demo.skill.seeded"},
	}
	got, err := inst.Install(context.Background(), scope, a)
	if err != nil {
		t.Fatalf("seedInstall: Install() err = %v", err)
	}
	return got
}
