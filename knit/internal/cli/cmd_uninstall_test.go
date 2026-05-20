package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallCommand_metadata(t *testing.T) {
	c := NewUninstallCommand()
	if c.Name() != "uninstall" {
		t.Errorf("Name = %q, want uninstall", c.Name())
	}
	if c.Synopsis() == "" {
		t.Errorf("Synopsis empty")
	}
}

func TestUninstallCommand_Run_afterInstall(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	install := NewInstallCommand()
	if err := runCommand(t, install, rt, []string{
		"--scope=user", "--target=claude",
		 f.pack,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	want := filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("install did not create artifact: %v", err)
	}

	rt, stdout, _ := f.runtime(t)
	cmd := NewUninstallCommand()
	if err := runCommand(t, cmd, rt, []string{"--scope=user", "--target=claude", f.pack}); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !strings.Contains(stdout.String(), "removed") {
		t.Errorf("stdout missing 'removed': %q", stdout.String())
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("artifact still exists after uninstall: %v", err)
	}
}

func TestUninstallCommand_Run_noPriorInstall(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, stderr := f.runtime(t)
	cmd := NewUninstallCommand()
	if err := runCommand(t, cmd, rt, []string{"--scope=user", "--target=claude", f.pack}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("expected warning in stderr: %q", stderr.String())
	}
}

func TestUninstallCommand_Run_missingPack(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, _ := f.runtime(t)
	cmd := NewUninstallCommand()
	err := runCommand(t, cmd, rt, []string{"--scope=user", "--target=claude"})
	if !errors.Is(err, ErrMissingArgument) {
		t.Errorf("err = %v, want ErrMissingArgument", err)
	}
}
