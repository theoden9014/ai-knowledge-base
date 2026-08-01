package source

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func newLoaderForTest(t *testing.T) Loader {
	t.Helper()
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	return NewLoader(v)
}

func TestNewLoader_returnsNonNil(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	if l := NewLoader(v); l == nil {
		t.Fatal("NewLoader returned nil")
	}
}

func TestLoader_LoadPack_success(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: test pack
default_tools: [claude]
entries:
  - id: p.skill.a
    path: skills/a
  - id: p.agent.x
    path: agents/x.md
`)},
		"p/skills/a/SKILL.md": {Data: []byte(`---
id: p.skill.a
kind: skill
name: p-a
description: skill a
tools:
  claude:
    enabled: true
    frontmatter:
      foo: bar
---
body of skill a
`)},
		"p/agents/x.md": {Data: []byte(`---
id: p.agent.x
kind: agent
name: p-x
description: agent x
uses_skills:
  - p.skill.a
---
body of agent x
`)},
	}

	l := newLoaderForTest(t)
	pack, info, err := l.LoadPack(context.Background(), fsys, "p")
	if err != nil {
		t.Fatalf("LoadPack() error = %v", err)
	}

	wantInfo := LoadInfo{PackDir: "p"}
	if diff := cmp.Diff(wantInfo, info); diff != "" {
		t.Errorf("LoadInfo mismatch (-want +got):\n%s", diff)
	}

	enabledTrue := true
	wantSkillMeta, err := NewSkillMeta("skills/a", nil)
	if err != nil {
		t.Fatalf("NewSkillMeta: %v", err)
	}
	wantPack := &Pack{
		Name:         "p",
		Version:      "0.1.0",
		Description:  "test pack",
		DefaultTools: []Target{Target("claude")},
		Entries: []Entry{
			{
				ID:          "p.skill.a",
				Kind:        KindSkill,
				Name:        "p-a",
				Description: "skill a",
				Tools: map[Target]ToolConfig{
					Target("claude"): {
						Enabled:     &enabledTrue,
						Frontmatter: map[string]any{"foo": "bar"},
					},
				},
				Path:  "skills/a",
				Body:  []byte("body of skill a\n"),
				Skill: wantSkillMeta,
			},
			{
				ID:          "p.agent.x",
				Kind:        KindAgent,
				Name:        "p-x",
				Description: "agent x",
				Agent:       &AgentMeta{UsesSkills: []string{"p.skill.a"}},
				Path:        "agents/x.md",
				Body:        []byte("body of agent x\n"),
			},
		},
	}
	if diff := cmp.Diff(wantPack, pack, cmp.AllowUnexported(SkillMeta{}, SkillAsset{})); diff != "" {
		t.Errorf("Pack mismatch (-want +got):\n%s", diff)
	}
}

func TestLoader_LoadPack_acceptsPackRootFilesystem(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: root pack
entries:
  - id: p.skill.a
    path: skills/a
`)},
		"skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
	}

	pack, _, err := newLoaderForTest(t).LoadPack(context.Background(), fsys, ".")
	if err != nil {
		t.Fatalf("LoadPack() error = %v", err)
	}
	if pack.Name != "p" {
		t.Errorf("Pack.Name = %q, want p", pack.Name)
	}
}

func TestLoader_LoadPack_errors(t *testing.T) {
	tests := []struct {
		name     string
		fsys     fs.FS
		ctxFn    func() context.Context
		packDir  string
		wantKind error
	}{
		{
			name: "missing manifest",
			fsys: fstest.MapFS{
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrManifestNotFound,
		},
		{
			name: "manifest schema violation",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: P
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrSchemaViolation,
		},
		{
			name: "skill path missing",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
			},
			packDir:  "p",
			wantKind: ErrSkillPathNotFound,
		},
		{
			name: "agent file missing surfaces ErrEntryNotFound",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.agent.x
    path: agents/x.md
`)},
			},
			packDir:  "p",
			wantKind: ErrEntryNotFound,
		},
		{
			name: "entry frontmatter schema violation",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: PackA\ndescription: bad name\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrSchemaViolation,
		},
		{
			name: "frontmatter id does not match manifest",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.b\nkind: skill\nname: p-b\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrIDMismatch,
		},
		{
			name: "manifest pack does not match directory",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: q
version: 0.1.0
description: d
entries:
  - id: q.skill.a
    path: skills/a
`)},
			},
			packDir:  "p",
			wantKind: ErrPackMismatch,
		},
		{
			name: "entry id pack does not match manifest",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: q.skill.a
    path: skills/a
`)},
			},
			packDir:  "p",
			wantKind: ErrPackMismatch,
		},
		{
			name: "frontmatter kind does not match id",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: agent\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrKindMismatch,
		},
		{
			name: "entry path name does not match id",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/b
`)},
			},
			packDir:  "p",
			wantKind: ErrPathMismatch,
		},
		{
			name: "duplicate entry id in manifest",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
  - id: p.skill.a
    path: skills/a2
`)},
				"p/skills/a/SKILL.md":  {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
				"p/skills/a2/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrDuplicateEntryID,
		},
		{
			name: "frontmatter kind unknown surfaces ErrInvalidKind",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skil\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			packDir:  "p",
			wantKind: ErrInvalidKind,
		},
		{
			name: "cancelled context aborts before reading",
			fsys: fstest.MapFS{
				"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a
`)},
				"p/skills/a/SKILL.md": {Data: []byte("---\nid: p.skill.a\nkind: skill\nname: p-a\ndescription: d\n---\nbody\n")},
			},
			ctxFn: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			packDir:  "p",
			wantKind: context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newLoaderForTest(t)
			ctx := context.Background()
			if tt.ctxFn != nil {
				ctx = tt.ctxFn()
			}
			_, _, err := l.LoadPack(ctx, tt.fsys, tt.packDir)
			if err == nil {
				t.Fatalf("LoadPack() error = nil, want %v", tt.wantKind)
			}
			if !errors.Is(err, tt.wantKind) {
				t.Errorf("LoadPack() error = %v, want errors.Is %v", err, tt.wantKind)
			}
		})
	}
}

func TestLoader_LoadPack_skipsAgentMetaForNonAgent(t *testing.T) {
	fsys := fstest.MapFS{
		"p/manifest.yaml": {Data: []byte(`pack: p
version: 0.1.0
description: d
default_tools: [claude]
entries:
  - id: p.skill.s
    path: skills/s
`)},
		"p/skills/s/SKILL.md": {Data: []byte(`---
id: p.skill.s
kind: skill
name: p-s
description: skill s
---
skill body
`)},
	}
	l := newLoaderForTest(t)
	pack, _, err := l.LoadPack(context.Background(), fsys, "p")
	if err != nil {
		t.Fatalf("LoadPack() error = %v", err)
	}
	if len(pack.Entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(pack.Entries))
	}
	if pack.Entries[0].Agent != nil {
		t.Errorf("Entries[0].Agent = %+v, want nil for non-agent kind", pack.Entries[0].Agent)
	}
}
