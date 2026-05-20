package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// cmdFixture wires a tempdir-backed environment for sub-command Run
// tests: a knowledge/<pack>/ directory with a single skill entry, an
// installable HOME, and a Runtime that points at both via os.Getenv-like
// shims.
type cmdFixture struct {
	tmp          string
	knowledgeDir string
	homeDir      string
	pack         string
}

// newCmdFixture builds a fresh fixture for use within a single test.
// All paths are absolute and resolved via filepath so they survive
// platform differences.
func newCmdFixture(t *testing.T) *cmdFixture {
	t.Helper()
	tmp := t.TempDir()
	pack := "p"
	knowledgeDir := filepath.Join(tmp, "knowledge")
	if err := os.MkdirAll(filepath.Join(knowledgeDir, pack, "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `pack: p
version: 0.1.0
description: test pack
default_tools: [claude]
entries:
  - id: p.skill.a
    path: skills/a.md
`
	skill := `---
id: p.skill.a
kind: skill
name: p-a
description: skill a
---
body of skill a
`
	if err := os.WriteFile(filepath.Join(knowledgeDir, pack, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(knowledgeDir, pack, "skills", "a.md"), []byte(skill), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	homeDir := filepath.Join(tmp, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	return &cmdFixture{
		tmp:          tmp,
		knowledgeDir: knowledgeDir,
		homeDir:      homeDir,
		pack:         pack,
	}
}

// runtime returns a Runtime configured for sub-command tests. stdout /
// stderr are bytes.Buffers so callers can assert on their contents.
func (f *cmdFixture) runtime(t *testing.T) (*Runtime, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	rt := &Runtime{
		Stdout: stdout,
		Stderr: stderr,
		Args:   nil,
		Getenv: func(k string) string {
			switch k {
			case "HOME":
				return f.homeDir
			default:
				return ""
			}
		},
		Getwd: func() (string, error) { return f.tmp, nil },
		Fsys:  os.DirFS("/"),
	}
	return rt, stdout, stderr
}

// runCommand parses args against cmd's FlagSet and invokes Run.
func runCommand(t *testing.T, cmd Command, rt *Runtime, args []string) error {
	t.Helper()
	fs := cmd.Flags()
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return cmd.Run(context.Background(), rt, fs)
}
