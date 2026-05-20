package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCommand_metadata(t *testing.T) {
	c := NewInstallCommand()
	if c.Name() != "install" {
		t.Errorf("Name = %q, want install", c.Name())
	}
	if c.Synopsis() == "" {
		t.Errorf("Synopsis empty")
	}
	if !strings.Contains(c.Help(), "usage:") {
		t.Errorf("Help missing usage")
	}
	if c.Flags() == nil {
		t.Errorf("Flags returned nil")
	}
}

func TestInstallCommand_Run_singleTarget(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=user",
		"--target=claude",
		
		f.pack,
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	want := filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected artifact at %s: %v", want, err)
	}
	if !strings.Contains(stdout.String(), "installed") {
		t.Errorf("stdout missing 'installed': %q", stdout.String())
	}
}

func TestInstallCommand_Run_missingPack(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{"--target=claude"})
	if !errors.Is(err, ErrMissingArgument) {
		t.Errorf("err = %v, want ErrMissingArgument", err)
	}
}

func TestInstallCommand_Run_invalidScope(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=global",
		"--target=claude",
		
		f.pack,
	})
	if !errors.Is(err, ErrInvalidFlagValue) {
		t.Errorf("err = %v, want ErrInvalidFlagValue", err)
	}
}

func TestInstallCommand_Run_targetAll_multiTargetSuccess(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=user",
		"--target=all",
		
		f.pack,
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	// Default tools is only [claude], so codex/gemini build nothing; we
	// still expect no error. The claude artifact must exist on disk.
	want := filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected artifact at %s: %v", want, err)
	}
	// The "no artifacts for this target" path must surface via stdout so
	// users can tell a no-op apart from a successful install.
	out := stdout.String()
	if !strings.Contains(out, "skipped codex/user") {
		t.Errorf("stdout missing skip message for codex: %q", out)
	}
	if !strings.Contains(out, "skipped gemini/user") {
		t.Errorf("stdout missing skip message for gemini: %q", out)
	}
	if !strings.Contains(out, "installed 1 artifacts to claude/user") {
		t.Errorf("stdout missing install message for claude: %q", out)
	}
}
