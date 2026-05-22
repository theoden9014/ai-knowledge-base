package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCommand_metadata(t *testing.T) {
	c := NewUpdateCommand()
	if c.Name() != "update" {
		t.Errorf("Name = %q, want update", c.Name())
	}
	if c.Synopsis() == "" {
		t.Errorf("Synopsis empty")
	}
}

func TestUpdateCommand_Run_afterInstall(t *testing.T) {
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
	// Mutate the source body to verify update picks up the new content.
	skillPath := filepath.Join(f.knowledgeDir, f.pack, "skills", "a", "SKILL.md")
	updated := `---
id: p.skill.a
kind: skill
name: p-a
description: skill a (updated)
---
body v2
`
	if err := os.WriteFile(skillPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("rewrite skill: %v", err)
	}
	// update
	rt, stdout, _ := f.runtime(t)
	cmd := NewUpdateCommand()
	if err := runCommand(t, cmd, rt, []string{
		"--scope=user", "--target=claude",
		f.pack,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated") {
		t.Errorf("stdout missing 'updated': %q", stdout.String())
	}
	installed := filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md")
	got, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("read installed artifact: %v", err)
	}
	if !strings.Contains(string(got), "body v2") {
		t.Errorf("installed artifact not updated; got:\n%s", got)
	}
}

func TestUpdateCommand_Run_skipsTargetsWithoutPriorInstall(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, stderr := f.runtime(t)
	cmd := NewUpdateCommand()
	if err := runCommand(t, cmd, rt, []string{
		"--scope=user", "--target=claude",
		f.pack,
	}); err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if !strings.Contains(stderr.String(), "nothing to update") {
		t.Errorf("expected warning in stderr: %q", stderr.String())
	}
}
