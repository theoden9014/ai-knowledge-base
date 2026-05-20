package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

// withRemoteFixture installs a stubFetcher on the fixture's Runtime so
// URL E2E tests can run without touching the network. The pack named
// "remote-p" is wired with a minimal manifest + one skill, mirroring the
// shape of newCmdFixture's local pack.
func withRemoteFixture(t *testing.T, f *cmdFixture, host, packDir, packName string) *Runtime {
	t.Helper()
	stub := &stubFetcher{
		supportsHost: host,
		files:        remotePackFS(packDir, packName),
		packDir:      packDir,
	}
	rt, _, _ := f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}
	return rt
}

// TestInstallCommand_Run_remoteURL verifies the install command's URL
// path end-to-end with a stub Fetcher: parses the remote arg, dispatches
// via remote.Dispatch, loads the pack from the stub's FS, and runs the
// real claude Installer against the fixture HOME.
func TestInstallCommand_Run_remoteURL(t *testing.T) {
	f := newCmdFixture(t)
	stub := &stubFetcher{
		supportsHost: "github.com",
		files:        remotePackFS("remote-p", "remote-p"),
		packDir:      "remote-p",
	}
	rt, stdout, _ := f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}

	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=user", "--target=claude",
		"github.com/owner/remote-p",
	})
	if err != nil {
		t.Fatalf("install err: %v", err)
	}
	want := filepath.Join(f.homeDir, ".claude", "skills", "remote-p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected artifact at %s: %v", want, err)
	}
	if !strings.Contains(stdout.String(), "installed") {
		t.Errorf("stdout missing 'installed': %q", stdout.String())
	}
}

func TestInstallCommand_Run_remoteURL_unsupportedHostIsConfigError(t *testing.T) {
	f := newCmdFixture(t)
	stub := &stubFetcher{supportsHost: "gitlab.com"} // wrong host
	rt, _, _ := f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}

	cmd := NewInstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=user", "--target=claude",
		"github.com/owner/remote-p",
	})
	if !errors.Is(err, remote.ErrUnsupportedHost) {
		t.Errorf("err = %v, want remote.ErrUnsupportedHost", err)
	}
}

// TestUpdateCommand_Run_remoteURL verifies that update can also accept a
// remote URL: install once locally, then re-distribute from a remote URL.
// Because the pack name in installationBelongsToPack comes from
// Pack.Name (= manifest's pack: field), the local install and the
// remote update must share the same pack name to be matched.
func TestUpdateCommand_Run_remoteURL(t *testing.T) {
	f := newCmdFixture(t)
	// First, install the local pack so update has prior installations
	// for the same Pack.Name.
	rt, _, _ := f.runtime(t)
	if err := runCommand(t, NewInstallCommand(), rt, []string{
		"--scope=user", "--target=claude", f.pack,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	// Stub a remote pack named "p" (same as local) but with new body.
	stub := &stubFetcher{
		supportsHost: "github.com",
		files:        remotePackFS("p", "p"),
		packDir:      "p",
	}
	rt2, stdout, _ := f.runtime(t)
	rt2.Fetchers = []remote.Fetcher{stub}
	cmd := NewUpdateCommand()
	if err := runCommand(t, cmd, rt2, []string{
		"--scope=user", "--target=claude",
		"github.com/owner/p",
	}); err != nil {
		t.Fatalf("update remote URL err: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated") {
		t.Errorf("stdout missing 'updated': %q", stdout.String())
	}
	// The installed artifact must reflect the remote pack body.
	got, err := os.ReadFile(filepath.Join(f.homeDir, ".claude", "skills", "p-a", "SKILL.md"))
	if err != nil {
		t.Fatalf("read installed: %v", err)
	}
	if !strings.Contains(string(got), "body of remote skill a") {
		t.Errorf("installed artifact not from remote pack:\n%s", got)
	}
}

// TestBuildCommand_Run_remoteURL verifies that build accepts a remote URL.
func TestBuildCommand_Run_remoteURL(t *testing.T) {
	f := newCmdFixture(t)
	rt := withRemoteFixture(t, f, "github.com", "remote-p", "remote-p")
	outDir := filepath.Join(f.tmp, "out")
	cmd := NewBuildCommand()
	if err := runCommand(t, cmd, rt, []string{
		"--target=claude", "-o", outDir,
		"github.com/owner/remote-p",
	}); err != nil {
		t.Fatalf("build err: %v", err)
	}
	want := filepath.Join(outDir, "skills", "remote-p-a", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected artifact at %s: %v", want, err)
	}
}


// TestUninstallCommand_Run_rejectsRemoteURL verifies that uninstall
// refuses URL args even with a Fetcher registered. The rejection is
// unconditional; see cmd_uninstall.go's Run godoc for the rationale.
func TestUninstallCommand_Run_rejectsRemoteURL(t *testing.T) {
	f := newCmdFixture(t)
	rt := withRemoteFixture(t, f, "github.com", "remote-p", "remote-p")
	cmd := NewUninstallCommand()
	err := runCommand(t, cmd, rt, []string{
		"--scope=user", "--target=claude",
		"github.com/owner/remote-p",
	})
	if !errors.Is(err, ErrUsage) {
		t.Errorf("err = %v, want ErrUsage", err)
	}
}

// TestExecute_goldenPath_remoteURL drives a full URL-based install → list
// → uninstall (local pack name) flow through Execute, mirroring the
// existing Wave3 local-only golden path.
func TestExecute_goldenPath_remoteURL(t *testing.T) {
	f := newCmdFixture(t)
	stub := &stubFetcher{
		supportsHost: "github.com",
		files:        remotePackFS("remote-p", "remote-p"),
		packDir:      "remote-p",
	}
	skill := filepath.Join(f.homeDir, ".claude", "skills", "remote-p-a", "SKILL.md")

	// install via URL
	rt, _, _ := f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}
	rt.Args = []string{"install", "--target=claude", "--scope=user", "github.com/owner/remote-p"}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("install exit = %v", code)
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("install did not create %s: %v", skill, err)
	}

	// list (local-only command)
	rt, stdout, _ := f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}
	rt.Args = []string{"list", "--target=claude", "--scope=user"}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("list exit = %v", code)
	}
	if !strings.Contains(stdout.String(), "remote-p.skill.a") {
		t.Errorf("list output missing source-entry: %q", stdout.String())
	}

	// uninstall by local pack name (URL would be ErrUsage; the install
	// path recorded Pack.Name = "remote-p" so the local name suffices)
	rt, _, _ = f.runtime(t)
	rt.Fetchers = []remote.Fetcher{stub}
	rt.Args = []string{"uninstall", "--target=claude", "--scope=user", "remote-p"}
	if code := Execute(context.Background(), rt, "knit", "v0"); code != ExitSuccess {
		t.Fatalf("uninstall exit = %v", code)
	}
	if _, err := os.Stat(skill); !os.IsNotExist(err) {
		t.Errorf("uninstall did not remove %s: %v", skill, err)
	}
}
