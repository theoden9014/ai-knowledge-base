package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_defaultCommands(t *testing.T) {
	cmds := defaultCommands()
	want := []string{"install", "uninstall", "list", "update", "build"}
	if len(cmds) != len(want) {
		t.Fatalf("len = %d, want %d", len(cmds), len(want))
	}
	for i, w := range want {
		if cmds[i].Name() != w {
			t.Errorf("cmds[%d].Name() = %q, want %q", i, cmds[i].Name(), w)
		}
	}
}

func TestExecute_version(t *testing.T) {
	f := newCmdFixture(t)
	rt, stdout, _ := f.runtime(t)
	rt.Args = []string{"--version"}
	code := Execute(context.Background(), rt, "knit", "v9.9.9")
	if code != ExitSuccess {
		t.Errorf("ExitCode = %v, want %v", code, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "v9.9.9") {
		t.Errorf("stdout missing version: %q", stdout.String())
	}
}

func TestExecute_unknownCommand(t *testing.T) {
	f := newCmdFixture(t)
	rt, _, stderr := f.runtime(t)
	rt.Args = []string{"frobnicate"}
	code := Execute(context.Background(), rt, "knit", "v0")
	if code != ExitUsage {
		t.Errorf("ExitCode = %v, want %v", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "frobnicate") {
		t.Errorf("stderr missing command name: %q", stderr.String())
	}
}

// TestExecute_goldenPath drives a full install → list → uninstall flow
// through Execute, verifying the routing seam end-to-end without touching
// os.Args / the real HOME.
func TestExecute_goldenPath(t *testing.T) {
	f := newCmdFixture(t)
	skill := filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md")

	// install
	rt, _, _ := f.runtime(t)
	rt.Args = []string{"install", "--target=claude", "--scope=user", f.pack}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("install exit = %v", code)
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("install did not create %s: %v", skill, err)
	}

	// list
	rt, stdout, _ := f.runtime(t)
	rt.Args = []string{"list", "--target=claude", "--scope=user"}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("list exit = %v", code)
	}
	if !strings.Contains(stdout.String(), "p.skill.a") {
		t.Errorf("list output missing source-entry: %q", stdout.String())
	}

	// uninstall
	rt, _, _ = f.runtime(t)
	rt.Args = []string{"uninstall", "--target=claude", "--scope=user", f.pack}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("uninstall exit = %v", code)
	}
	if _, err := os.Stat(skill); !os.IsNotExist(err) {
		t.Errorf("uninstall did not remove %s: %v", skill, err)
	}

	// list again — should be empty
	rt, stdout, _ = f.runtime(t)
	rt.Args = []string{"list", "--target=claude", "--scope=user"}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("list (post-uninstall) exit = %v", code)
	}
	if strings.Contains(stdout.String(), "p.skill.a") {
		t.Errorf("list still contains source-entry after uninstall: %q", stdout.String())
	}
}
