package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestListCommand_metadata(t *testing.T) {
	c := NewListCommand()
	if c.Name() != "list" {
		t.Errorf("Name = %q, want list", c.Name())
	}
	if c.Synopsis() == "" {
		t.Errorf("Synopsis empty")
	}
}

func TestListCommand_Run_emptyInventory(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	cmd := NewListCommand()
	if err := runCommand(t, cmd, rt, []string{"--scope=user", "--target=claude"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	out := stdout.String()
	for _, col := range []string{"TARGET", "SCOPE", "PACK", "ENTRY_ID", "PATH"} {
		if !strings.Contains(out, col) {
			t.Errorf("expected header column %q in stdout, got:\n%s", col, out)
		}
	}
}

func TestListCommand_Run_afterInstall(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	// install first
	install := NewInstallCommand()
	if err := runCommand(t, install, rt, []string{
		"--scope=user", "--target=claude",
		f.pack,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	// then list
	rt, stdout, _ := f.runtime(t)
	cmd := NewListCommand()
	if err := runCommand(t, cmd, rt, []string{"--scope=user", "--target=claude"}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "claude") {
		t.Errorf("expected claude in listing:\n%s", out)
	}
	if !strings.Contains(out, "p.skill.a") {
		t.Errorf("expected source-entry id in listing:\n%s", out)
	}
}

func TestListCommand_Run_rejectsPositionalArgs(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	cmd := NewListCommand()
	err := runCommand(t, cmd, rt, []string{"--target=claude", "spurious"})
	if !errors.Is(err, ErrUsage) {
		t.Errorf("err = %v, want ErrUsage", err)
	}
}
