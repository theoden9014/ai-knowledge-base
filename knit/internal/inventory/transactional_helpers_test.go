package inventory

import (
	"path/filepath"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// txTestTarget is the synthetic target identifier used by the Transactional*
// unit tests. It is not used by any real distribution package.
const txTestTarget source.Target = "test-target"

// testPathPolicy is the PathPolicy used by the Transactional* unit tests.
// It accepts any path whose top segment is "skills" or "agents", plus the
// single flat file "AGENTS.md".
type testPathPolicy struct{}

func (testPathPolicy) Target() source.Target { return txTestTarget }

func (testPathPolicy) Validate(p source.ArtifactPath) error {
	switch p.TopSegment() {
	case "skills", "agents", "AGENTS.md":
		return nil
	default:
		return source.ErrInvalidArtifactPath
	}
}

// transactionalFixture wires up a complete in-memory inventory for tests:
// a MemoryArtifactStore, a SidecarLabelStore over t.TempDir(), and a
// PathResolver bound to user/project roots that also live under t.TempDir().
type transactionalFixture struct {
	store    *MemoryArtifactStore
	labels   *SidecarLabelStore
	resolver *PathResolver
	roots    InventoryRoots
}

func newTransactionalFixture(t *testing.T, withProject bool) transactionalFixture {
	t.Helper()
	tempDir := t.TempDir()
	userRootDir := filepath.Join(tempDir, "user")
	userRoot := must(NewInventoryRoot(userRootDir))
	var projectRoot InventoryRoot
	var projectLabelsRoot string
	if withProject {
		projectRootDir := filepath.Join(tempDir, "project")
		projectRoot = must(NewInventoryRoot(projectRootDir))
		projectLabelsRoot = filepath.Join(tempDir, "project-labels")
	}
	roots := must(NewInventoryRoots(userRoot, projectRoot))

	userLabelsRoot := filepath.Join(tempDir, "user-labels")
	labels := NewSidecarLabelStore(txTestTarget, userLabelsRoot, projectLabelsRoot)

	resolver := must(NewPathResolver(testPathPolicy{}, roots))
	store := NewMemoryArtifactStore()
	return transactionalFixture{
		store:    store,
		labels:   labels,
		resolver: resolver,
		roots:    roots,
	}
}

func sampleArtifact(t *testing.T, relPath string, target source.Target) source.Artifact {
	t.Helper()
	if _, err := source.NewArtifactPath(relPath); err != nil {
		t.Fatalf("invalid sample path %q: %v", relPath, err)
	}
	return source.Artifact{
		Target:         target,
		Path:           relPath,
		Content:        []byte("sample content\n"),
		SourceEntryIDs: []string{"test-pack.skill.entry"},
	}
}
