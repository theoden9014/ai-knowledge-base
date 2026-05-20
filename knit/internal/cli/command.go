package cli

import (
	"context"
	"flag"
)

// Command is the interface implemented by each knit subcommand.
//
// Separation of responsibilities:
//   - Name / Synopsis / Help: static metadata used for help output.
//   - Flags: returns the command-specific flag set as a fresh
//     flag.FlagSet. The router parses args with this FlagSet before
//     calling Run.
//   - Run: receives the parsed flag.FlagSet and shared Runtime and
//     performs the actual work. Run returns only an error; the caller
//     determines ExitCode uniquely via [errorToExitCode].
//
// SRP intent:
//   - Command implementations focus only on translating user intent
//     (CLI input) into domain-operation calls. They do not own domain
//     logic such as Builder or Installer behavior.
//   - Flag parsing and usage display are handled by the router (App) and
//     shared helpers.
//   - ExitCode selection is centralized in errorToExitCode. Commands do
//     not return ExitCode directly, which keeps that responsibility from
//     being scattered.
//
// # Positional Arguments
//
// Parsed positional arguments, such as "<pack>" in install <pack>, are
// read from inside Run via fs.Args(), fs.Arg(i), and fs.NArg(). Run does
// not receive positional arguments as a separate parameter so that fs
// remains the single source of truth.
type Command interface {
	// Name is the unique subcommand name, for example "install".
	Name() string

	// Synopsis is the one-line description shown in `knit help`.
	Synopsis() string

	// Help is the detailed usage text shown by `knit help <name>`.
	// It may be multi-line and must include a trailing newline.
	Help() string

	// Flags constructs and returns a fresh flag.FlagSet dedicated to this
	// command. The router parses args against the returned FlagSet with
	// flag.ContinueOnError behavior.
	//
	// Note: FlagSet must be created fresh on every call rather than
	// reused. This allows the same Command to be exercised concurrently in
	// tests.
	Flags() *flag.FlagSet

	// Run executes the subcommand's main behavior.
	//
	// Contract:
	//   - fs is the same FlagSet returned by Flags(), passed after the
	//     router has parsed args. Flag values come from fs, and positional
	//     arguments come from fs.Args() / fs.Arg(i).
	//   - rt is the Runtime abstraction for I/O and environment access.
	//   - ctx is used for cancellation propagation. Long-running I/O is
	//     expected to respect ctx.Err().
	//
	// Return value:
	//   - error represents execution failure. It should wrap CLI-specific,
	//     source, inventory, or distribution sentinels so callers can
	//     inspect them via errors.Is. Run does not choose ExitCode;
	//     App.Execute derives it uniquely through errorToExitCode.
	Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error
}
