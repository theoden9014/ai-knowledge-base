package source

import (
	"context"
	"errors"
	"sort"
	"testing"
	"testing/fstest"
)

func skillManifest(id, p string) string {
	return "pack: p\nversion: 0.1.0\ndescription: d\ndefault_tools: [claude]\nentries:\n  - id: " + id + "\n    path: " + p + "\n"
}

func skillBody(id, name string) string {
	return "---\nid: " + id + "\nkind: skill\nname: " + name + "\ndescription: d\n---\nbody\n"
}

func manualSkillBody(id, name string) string {
	return "---\nid: " + id + "\nkind: skill\nname: " + name + "\ndescription: d\ninvocation: manual\n---\nbody\n"
}

func loadSkill(t *testing.T, fsys fstest.MapFS) *Pack {
	t.Helper()
	l := newLoaderForTest(t)
	pack, _, err := l.LoadPack(context.Background(), fsys, "p")
	if err != nil {
		t.Fatalf("LoadPack: %v", err)
	}
	return pack
}

func TestLoader_skillWithSiblings_collectsAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml":              {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a/SKILL.md":          {Data: []byte(skillBody("p.skill.a", "p-a"))},
		"p/skills/a/scripts/run.sh":    {Data: []byte("echo hi\n")},
		"p/skills/a/refs/sub/notes.md": {Data: []byte("# notes\n")},
	}
	pack := loadSkill(t, fsys)
	if len(pack.Entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(pack.Entries))
	}
	e := pack.Entries[0]
	if e.Skill == nil {
		t.Fatal("Skill is nil")
	}
	if e.Skill.Root() != "skills/a" {
		t.Errorf("Root() = %q, want %q", e.Skill.Root(), "skills/a")
	}
	got := e.Skill.Assets()
	paths := make([]string, 0, len(got))
	contents := make(map[string]string, len(got))
	for _, a := range got {
		paths = append(paths, a.Path())
		contents[a.Path()] = string(a.Content())
	}
	sort.Strings(paths)
	wantPaths := []string{"refs/sub/notes.md", "scripts/run.sh"}
	if len(paths) != len(wantPaths) {
		t.Fatalf("assets len = %d (%v), want %d", len(paths), paths, len(wantPaths))
	}
	for i, want := range wantPaths {
		if paths[i] != want {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], want)
		}
	}
	if contents["scripts/run.sh"] != "echo hi\n" {
		t.Errorf("scripts/run.sh content = %q", contents["scripts/run.sh"])
	}
	if contents["refs/sub/notes.md"] != "# notes\n" {
		t.Errorf("refs/sub/notes.md content = %q", contents["refs/sub/notes.md"])
	}
}

func TestLoader_skillWithoutSiblings(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml":     {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a/SKILL.md": {Data: []byte(skillBody("p.skill.a", "p-a"))},
	}
	pack := loadSkill(t, fsys)
	if pack.Entries[0].Skill == nil {
		t.Fatal("Skill is nil")
	}
	if got := pack.Entries[0].Skill.Assets(); len(got) != 0 {
		t.Errorf("Assets len = %d, want 0", len(got))
	}
	if got := pack.Entries[0].Skill.Invocation(); got != SkillInvocationBoth {
		t.Errorf("Invocation() = %q, want %q", got, SkillInvocationBoth)
	}
}

func TestLoader_manualSkill(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml":     {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a/SKILL.md": {Data: []byte(manualSkillBody("p.skill.a", "p-a"))},
	}
	pack := loadSkill(t, fsys)
	if got := pack.Entries[0].Skill.Invocation(); got != SkillInvocationManual {
		t.Errorf("Invocation() = %q, want %q", got, SkillInvocationManual)
	}
}

func TestLoader_skillBodyMissing(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml":           {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a/scripts/run.sh": {Data: []byte("x")},
	}
	l := newLoaderForTest(t)
	_, _, err := l.LoadPack(context.Background(), fsys, "p")
	if !errors.Is(err, ErrSkillBodyNotFound) {
		t.Errorf("err = %v, want ErrSkillBodyNotFound", err)
	}
	if !errors.Is(err, ErrSkillResolution) {
		t.Errorf("err = %v, want errors.Is ErrSkillResolution", err)
	}
}

func TestLoader_skillPathIsFile(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml": {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a":      {Data: []byte("not a directory")},
	}
	l := newLoaderForTest(t)
	_, _, err := l.LoadPack(context.Background(), fsys, "p")
	if !errors.Is(err, ErrSkillPathNotDirectory) {
		t.Errorf("err = %v, want ErrSkillPathNotDirectory", err)
	}
	if !errors.Is(err, ErrSkillResolution) {
		t.Errorf("err = %v, want errors.Is ErrSkillResolution", err)
	}
}

func TestLoader_skillBodyIgnoresExactSKILLOnly(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml":     {Data: []byte(skillManifest("p.skill.a", "skills/a"))},
		"p/skills/a/SKILL.md": {Data: []byte(skillBody("p.skill.a", "p-a"))},
		"p/skills/a/skill.md": {Data: []byte("# lowercase variant\n")},
	}
	pack := loadSkill(t, fsys)
	assets := pack.Entries[0].Skill.Assets()
	if len(assets) != 1 {
		t.Fatalf("assets len = %d, want 1 (lowercase skill.md is a sibling)", len(assets))
	}
	if assets[0].Path() != "skill.md" {
		t.Errorf("assets[0].Path = %q, want skill.md", assets[0].Path())
	}
}
