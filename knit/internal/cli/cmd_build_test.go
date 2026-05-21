package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCommand_metadata(t *testing.T) {
	c := NewBuildCommand()
	if c.Name() != "build" {
		t.Errorf("Name = %q, want build", c.Name())
	}
}

func TestBuildCommand_Run_listOnly(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	cmd := NewBuildCommand()
	if err := runCommand(t, cmd, rt, []string{
		"--target=claude",

		f.pack,
	}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "SKILL.md") {
		t.Errorf("expected artifact path in stdout listing:\n%s", out)
	}
	if !strings.Contains(out, "claude") {
		t.Errorf("expected target name in stdout listing:\n%s", out)
	}
}

func TestBuildCommand_Run_withOutputDir(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	outDir := filepath.Join(f.tmp, "out")
	cmd := NewBuildCommand()
	if err := runCommand(t, cmd, rt, []string{
		"--target=claude",

		"-o", outDir,
		f.pack,
	}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote") {
		t.Errorf("stdout missing 'wrote': %q", stdout.String())
	}
	want := filepath.Join(outDir, "skills", "p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected artifact at %s: %v", want, err)
	}
}

func TestBuildCommand_Run_rejectsAllTarget(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	cmd := NewBuildCommand()
	err := runCommand(t, cmd, rt, []string{
		"--target=all",

		f.pack,
	})
	if !errors.Is(err, ErrInvalidFlagValue) {
		t.Errorf("err = %v, want ErrInvalidFlagValue", err)
	}
}
