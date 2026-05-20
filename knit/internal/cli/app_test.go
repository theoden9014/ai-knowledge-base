package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestNewApp(t *testing.T) {
	cmds := []Command{
		&fakeCommand{name: "install"},
		&fakeCommand{name: "list"},
	}
	app := NewApp("knit", "v1.2.3", cmds)
	if app == nil {
		t.Fatalf("NewApp returned nil")
	}
	if app.Name != "knit" {
		t.Errorf("Name = %q, want %q", app.Name, "knit")
	}
	if app.Version != "v1.2.3" {
		t.Errorf("Version = %q, want %q", app.Version, "v1.2.3")
	}
	if len(app.Commands) != len(cmds) {
		t.Errorf("Commands length = %d, want %d", len(app.Commands), len(cmds))
	}
}

func TestNewApp_panicOnDuplicateName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate command name")
		}
	}()
	NewApp("knit", "v0", []Command{
		&fakeCommand{name: "install"},
		&fakeCommand{name: "install"},
	})
}

func TestNewApp_panicOnEmptyCommands(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on empty commands")
		}
	}()
	NewApp("knit", "v0", nil)
}

// appExecTestCase drives App.Execute with a controlled Runtime.Args and
// asserts the resulting ExitCode plus the routing side-effects (which
// command's Run was invoked, what stdout/stderr looks like).
type appExecTestCase struct {
	name               string
	args               []string
	cmdRunErr          error
	wantExit           ExitCode
	wantStdoutContains []string
	wantStderrContains []string
	wantCalled         string // expected command Name() that should have been Run
}

func TestApp_Execute(t *testing.T) {
	tests := []appExecTestCase{
		{
			name:               "no args prints overview, ExitUsage",
			args:               nil,
			wantExit:           ExitUsage,
			wantStderrContains: []string{"available commands", "install"},
		},
		{
			name:               "-h prints overview to stdout, ExitSuccess (POSIX/GNU convention)",
			args:               []string{"-h"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"available commands"},
		},
		{
			name:               "--help prints overview to stdout, ExitSuccess (POSIX/GNU convention)",
			args:               []string{"--help"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"available commands"},
		},
		{
			name:               "help subcommand routed to help command",
			args:               []string{"help"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"available commands"},
			wantCalled:         "help",
		},
		{
			name:               "-v prints version, ExitSuccess",
			args:               []string{"-v"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"v1.2.3"},
		},
		{
			name:               "--version prints version, ExitSuccess",
			args:               []string{"--version"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"v1.2.3"},
		},
		{
			name:               "unknown subcommand returns ExitUsage",
			args:               []string{"frobnicate"},
			wantExit:           ExitUsage,
			wantStderrContains: []string{"unknown command", "frobnicate"},
		},
		{
			name:       "known subcommand calls its Run",
			args:       []string{"install"},
			wantExit:   ExitSuccess,
			wantCalled: "install",
		},
		{
			name:               "subcommand --help prints command help to stdout, ExitSuccess",
			args:               []string{"install", "--help"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"install help"},
		},
		{
			name:               "subcommand -h prints command help to stdout, ExitSuccess",
			args:               []string{"install", "-h"},
			wantExit:           ExitSuccess,
			wantStdoutContains: []string{"install help"},
		},
		{
			name:               "subcommand Run error → mapped exit code",
			args:               []string{"install"},
			cmdRunErr:          ErrMissingArgument,
			wantExit:           ExitUsage,
			wantStderrContains: []string{"missing argument"},
			wantCalled:         "install",
		},
		{
			name:               "subcommand Run unknown error → ExitGeneral",
			args:               []string{"install"},
			cmdRunErr:          errors.New("boom"),
			wantExit:           ExitGeneral,
			wantStderrContains: []string{"boom"},
			wantCalled:         "install",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			install := &fakeCommand{name: "install", synopsis: "install pack", help: "install help\n", runErr: tt.cmdRunErr}
			list := &fakeCommand{name: "list", synopsis: "list installations", help: "list help\n"}
			cmds := []Command{install, list}
			cmds = append(cmds, NewHelpCommand(cmds))
			app := NewApp("knit", "v1.2.3", cmds)

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			rt := &Runtime{
				Stdout: stdout,
				Stderr: stderr,
				Args:   tt.args,
				Getenv: func(string) string { return "" },
				Getwd:  func() (string, error) { return "/", nil },
			}
			got := app.Execute(context.Background(), rt)
			if got != tt.wantExit {
				t.Errorf("ExitCode = %v, want %v", got, tt.wantExit)
			}
			out := stdout.String()
			for _, s := range tt.wantStdoutContains {
				if !strings.Contains(out, s) {
					t.Errorf("stdout missing %q\nstdout:\n%s", s, out)
				}
			}
			errOut := stderr.String()
			for _, s := range tt.wantStderrContains {
				if !strings.Contains(errOut, s) {
					t.Errorf("stderr missing %q\nstderr:\n%s", s, errOut)
				}
			}
			if tt.wantCalled != "" {
				switch tt.wantCalled {
				case "install":
					if !install.runCalled {
						t.Errorf("install.Run should have been called")
					}
				case "list":
					if !list.runCalled {
						t.Errorf("list.Run should have been called")
					}
				case "help":
					// help routed: we cannot directly inspect because help is
					// freshly constructed; assertion on output above suffices.
				}
			}
		})
	}
}

// Verify that the FlagSet returned by Command.Flags is parsed against
// rt.Args[1:] and that fs.Args() contains the post-flag positional args.
type flagSpyCmd struct {
	fakeCommand
	gotScope string
}

func (s *flagSpyCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet(s.name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&s.gotScope, "scope", "user", "scope")
	return fs
}

func TestApp_Execute_passesParsedFlagsToRun(t *testing.T) {
	spy := &flagSpyCmd{fakeCommand: fakeCommand{name: "install", help: "install help\n"}}
	cmds := []Command{spy}
	cmds = append(cmds, NewHelpCommand(cmds))
	app := NewApp("knit", "v0", cmds)
	rt := &Runtime{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Args:   []string{"install", "--scope=project", "pack-name"},
		Getenv: func(string) string { return "" },
		Getwd:  func() (string, error) { return "/", nil },
	}
	if got := app.Execute(context.Background(), rt); got != ExitSuccess {
		t.Fatalf("ExitCode = %v, want %v", got, ExitSuccess)
	}
	if spy.gotScope != "project" {
		t.Errorf("--scope flag not parsed: got %q, want %q", spy.gotScope, "project")
	}
	if len(spy.gotArgs) != 1 || spy.gotArgs[0] != "pack-name" {
		t.Errorf("positional args = %v, want [pack-name]", spy.gotArgs)
	}
}
