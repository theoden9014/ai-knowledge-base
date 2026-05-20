package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
)

// App is the router for the subcommand tree.
//
// Single responsibility: take the subcommand name from the front of
// Runtime.Args, choose the matching Command, parse the remaining args
// with its Flags(), and call Run.
//
// This type does not own the following responsibilities:
//   - command-specific execution logic (delegated to Command implementations)
//   - per-command flag definitions (delegated to Command.Flags)
//   - I/O and args handling (delegated to Runtime; the single source of
//     truth for args is [Runtime.Args])
//   - DistributionFactory construction (delegated to the caller of
//     Execute / NewApp)
type App struct {
	// Name is the binary name, usually "knit". It is used in usage output.
	Name string

	// Version is the build-time version string.
	// It is used for --version output. An empty string is treated as
	// "(devel)".
	Version string

	// Commands is the set of subcommands, looked up by Name().
	// Registering multiple Commands with the same Name is invalid and
	// causes NewApp to panic.
	Commands []Command
}

// NewApp builds an App from the subcommand set plus name and version.
//
// Contract:
//   - commands must contain at least one Command.
//   - if multiple Commands share the same Name, this function panics.
//     That is treated as an early configuration failure rather than a
//     runtime omission.
//   - the caller must assemble and include the help command via
//     [NewHelpCommand]. App does not embed help on its own in order to
//     preserve SRP.
func NewApp(name, version string, commands []Command) *App {
	if len(commands) == 0 {
		panic("cli: NewApp requires at least one command")
	}
	seen := make(map[string]struct{}, len(commands))
	for _, c := range commands {
		n := c.Name()
		if _, dup := seen[n]; dup {
			panic(fmt.Sprintf("cli: duplicate command name %q", n))
		}
		seen[n] = struct{}{}
	}
	return &App{Name: name, Version: version, Commands: commands}
}

// Execute routes Runtime.Args and calls Run on the selected Command.
//
// Processing order:
//  1. If rt.Args is empty, print the full subcommand list to stderr and
//     return ExitUsage.
//  2. If rt.Args[0] is "-h" or "--help", print the full subcommand list
//     to stdout and return ExitSuccess. This is the explicit-help path
//     and follows POSIX/GNU convention. Routing to the "help"
//     subcommand is treated as normal command dispatch and delegated to
//     help.Run.
//  3. If rt.Args[0] is "-v" or "--version", print Version to stdout and
//     return ExitSuccess.
//  4. Look up rt.Args[0] in Commands as the subcommand name. If it is
//     not found, return ErrUnknownCommand (ExitUsage).
//  5. Parse rt.Args[1:] with the selected Command's Flags(). If parsing
//     fails, return ErrUsage (ExitUsage).
//  6. Call Command.Run(ctx, rt, fs). Run returns only an error.
//  7. If Run returns a non-nil error, derive ExitCode via
//     errorToExitCode and write the error details to stderr. If Run
//     returns nil, return ExitSuccess.
//
// ctx exists to propagate external cancellation such as SIGINT.
// This package does not register signal handlers itself. Callers such as
// main or tests construct the context when needed.
func (a *App) Execute(ctx context.Context, rt *Runtime) ExitCode {
	// 1. empty args → usage error.
	if len(rt.Args) == 0 {
		printOverview(rt.Stderr, a.Commands)
		return ExitUsage
	}
	first := rt.Args[0]
	// 2. explicit help request → success (POSIX/GNU convention).
	if first == "-h" || first == "--help" {
		printOverview(rt.Stdout, a.Commands)
		return ExitSuccess
	}
	// 2. -v / --version → print version.
	if first == "-v" || first == "--version" {
		v := a.Version
		if v == "" {
			v = "(devel)"
		}
		_, _ = fmt.Fprintln(rt.Stdout, v)
		return ExitSuccess
	}
	// 3. routing.
	cmd := a.lookup(first)
	if cmd == nil {
		_, _ = fmt.Fprintf(rt.Stderr, "%s: unknown command %q\n", a.Name, first)
		return ExitUsage
	}
	// 4. parse flags.
	fs := cmd.Flags()
	if err := fs.Parse(rt.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprint(rt.Stdout, cmd.Help())
			return ExitSuccess
		}
		// flag.ContinueOnError already wrote a message to fs.Output() unless
		// suppressed; re-surface as ErrUsage for the mapping below.
		_, _ = fmt.Fprintf(rt.Stderr, "%s %s: %v\n", a.Name, cmd.Name(), err)
		return ExitUsage
	}
	// 5. run.
	if err := cmd.Run(ctx, rt, fs); err != nil {
		// Distinguish unknown-command (returned by help) so the operator
		// sees the precise sentinel name in the stderr message.
		if errors.Is(err, ErrUnknownCommand) {
			_, _ = fmt.Fprintf(rt.Stderr, "%s: %v\n", a.Name, err)
		} else {
			_, _ = fmt.Fprintf(rt.Stderr, "%s %s: %v\n", a.Name, cmd.Name(), err)
		}
		return errorToExitCode(err)
	}
	return ExitSuccess
}

// lookup returns the Command whose Name() matches the given subcommand, or
// nil when none does. The linear scan is fine: knit ships fewer than a
// dozen subcommands.
func (a *App) lookup(name string) Command {
	for _, c := range a.Commands {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
