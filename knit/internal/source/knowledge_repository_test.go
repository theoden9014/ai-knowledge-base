package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryKnowledgePacksLoad(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	knowledgeRoot := filepath.Join(repoRoot, "knowledge")
	packDirs, err := os.ReadDir(knowledgeRoot)
	if err != nil {
		t.Fatalf("read knowledge root: %v", err)
	}

	l := newLoaderForTest(t)
	for _, packDir := range packDirs {
		if !packDir.IsDir() {
			continue
		}
		t.Run(packDir.Name(), func(t *testing.T) {
			pack, info, err := l.LoadPack(context.Background(), os.DirFS(knowledgeRoot), packDir.Name())
			if err != nil {
				t.Fatalf("LoadPack(%q) error = %v", packDir.Name(), err)
			}
			if info.PackDir != packDir.Name() {
				t.Errorf("LoadInfo.PackDir = %q, want %q", info.PackDir, packDir.Name())
			}
			if pack.Name != packDir.Name() {
				t.Errorf("Pack.Name = %q, want directory name %q", pack.Name, packDir.Name())
			}
			if len(pack.Entries) == 0 {
				t.Errorf("Pack.Entries is empty")
			}
		})
	}
}
