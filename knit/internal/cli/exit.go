package cli

// ExitCode represents the process exit code for knit.
// The range follows POSIX convention and uses 0..125 (126 and 127 are
// reserved by the shell).
//
// Range design:
//   - 0: success
//   - 1: general execution failure (runtime I/O error, dependency-layer error)
//   - 2: usage error (unknown subcommand, invalid flag, missing required argument)
//   - 3: missing configuration (HOME unset / project root not found / knowledge dir not found)
//   - 4: target object not found (missing Pack or Installation)
//   - 5: already exists / conflict (duplicate install, unmanaged file collision)
//   - 6: aggregate failure across multiple Targets (some branch of --target=all failed)
//
// Detailed sentinel-error to ExitCode mapping is centralized in
// [errorToExitCode].
type ExitCode int

const (
	// ExitSuccess represents successful completion.
	ExitSuccess ExitCode = 0
	// ExitGeneral represents a general execution failure.
	ExitGeneral ExitCode = 1
	// ExitUsage represents incorrect subcommand, flag, or argument usage.
	ExitUsage ExitCode = 2
	// ExitConfig represents missing required configuration such as HOME,
	// project root, or knowledge dir.
	ExitConfig ExitCode = 3
	// ExitNotFound represents a missing target object such as a Pack or
	// Installation.
	ExitNotFound ExitCode = 4
	// ExitConflict represents a conflict such as an existing destination.
	ExitConflict ExitCode = 5
	// ExitPartial represents a run across multiple Targets such as
	// --target=all in which some Targets failed.
	ExitPartial ExitCode = 6
)

// Int returns ExitCode as an int for passing to os.Exit from main.
func (c ExitCode) Int() int { return int(c) }
