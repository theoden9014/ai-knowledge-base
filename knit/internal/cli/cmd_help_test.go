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

// fakeCommand implements Command for testing the help / app routing logic
// without needing the real install/list/etc. commands.
type fakeCommand struct {
	name      string
	synopsis  string
	help      string
	runErr    error
	runCalled bool
	gotArgs   []string
}

func (f *fakeCommand) Name() string     { return f.name }
func (f *fakeCommand) Synopsis() string { return f.synopsis }
func (f *fakeCommand) Help() string     { return f.help }
func (f *fakeCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet(f.name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}
func (f *fakeCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	f.runCalled = true
	f.gotArgs = append([]string(nil), fs.Args()...)
	return f.runErr
}

func TestNewHelpCommand(t *testing.T) {
	cmds := []Command{&fakeCommand{name: "x"}}
	got := NewHelpCommand(cmds)
	if got == nil {
		t.Fatalf("NewHelpCommand returned nil")
	}
	if got.Name() != "help" {
		t.Errorf("Name = %q, want %q", got.Name(), "help")
	}
}

func Test_helpCommand_metadata(t *testing.T) {
	c := NewHelpCommand(nil)
	if c.Name() != "help" {
		t.Errorf("Name mismatch: %q", c.Name())
	}
	if c.Synopsis() == "" {
		t.Errorf("Synopsis empty")
	}
	if !strings.Contains(c.Help(), "usage:") {
		t.Errorf("Help missing usage hint: %q", c.Help())
	}
	if c.Flags() == nil {
		t.Errorf("Flags returned nil")
	}
}

func Test_helpCommand_Run(t *testing.T) {
	cmds := []Command{
		&fakeCommand{name: "install", synopsis: "install pack", help: "install help body\n"},
		&fakeCommand{name: "list", synopsis: "list installations", help: "list help body\n"},
	}
	help := NewHelpCommand(cmds)

	tests := []struct {
		name         string
		args         []string
		wantContains []string
		wantErr      error
	}{
		{
			name:         "no args lists overview",
			args:         nil,
			wantContains: []string{"available commands", "install", "install pack", "list installations"},
		},
		{
			name:         "single subcommand prints its Help",
			args:         []string{"install"},
			wantContains: []string{"install help body"},
		},
		{
			name:    "unknown subcommand returns ErrUnknownCommand",
			args:    []string{"nosuch"},
			wantErr: ErrUnknownCommand,
		},
		{
			name:    "too many args returns ErrUsage",
			args:    []string{"install", "extra"},
			wantErr: ErrUsage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			rt := &Runtime{Stdout: stdout, Stderr: io.Discard, Getenv: func(string) string { return "" }}
			fs := help.Flags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			err := help.Run(context.Background(), rt, fs)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			out := stdout.String()
			for _, s := range tt.wantContains {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\noutput:\n%s", s, out)
				}
			}
		})
	}
}
