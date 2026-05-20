package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
)

// helpCommand implements the `knit help [<subcommand>]` command.
//
// Semantics:
//   - Without arguments: list the Name and Synopsis of all commands.
//   - With <subcommand>: show that command's Help. If it is unknown,
//     return ErrUnknownCommand.
//
// Design intent:
//   - help is handled as an independent Command rather than being built
//     into App. That gives only help the authority to inspect the command
//     set, and App itself does not need to know about help
//     (SRP: the router does not assemble help content).
//   - Because help needs access to other commands' Synopsis and Help,
//     it receives a reference to the command set at construction time.
//     To avoid cyclic dependencies, Execute builds `defaultCommands()`
//     first and then calls NewHelpCommand to inject the set.
type helpCommand struct {
	// commands is the slice of all commands that help can inspect.
	// It may include help itself so that help can describe help.
	commands []Command
}

// NewHelpCommand constructs the help command. The commands argument
// must contain the full command set, including help itself.
func NewHelpCommand(commands []Command) Command {
	return &helpCommand{commands: commands}
}

// Name returns "help".
func (c *helpCommand) Name() string { return "help" }

// Synopsis returns a one-line description.
func (c *helpCommand) Synopsis() string {
	return "show help for knit or a subcommand"
}

// Help returns the detailed help text.
func (c *helpCommand) Help() string {
	return `usage: knit help [<subcommand>]

Without arguments, lists all available subcommands.
With a subcommand name, prints that subcommand's usage details.
`
}

// Flags returns an empty FlagSet because help has no command-specific flags.
func (c *helpCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// Run prints help according to the semantics described above.
func (c *helpCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	if fs.NArg() == 0 {
		printOverview(rt.Stdout, c.commands)
		return nil
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("%w: help accepts at most one subcommand name", ErrUsage)
	}
	name := fs.Arg(0)
	for _, cmd := range c.commands {
		if cmd.Name() == name {
			_, _ = fmt.Fprint(rt.Stdout, cmd.Help())
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrUnknownCommand, name)
}

// printOverview writes the global usage summary listing each command's
// name and synopsis. It is shared with App.Execute so the same layout is
// produced regardless of whether the user runs `knit help` or just `knit`.
func printOverview(w io.Writer, commands []Command) {
	_, _ = fmt.Fprintln(w, "usage: knit <command> [flags] [args]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "available commands:")
	for _, cmd := range commands {
		_, _ = fmt.Fprintf(w, "  %-10s %s\n", cmd.Name(), cmd.Synopsis())
	}
}
