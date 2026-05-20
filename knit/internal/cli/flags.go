package cli

import "flag"

// Helpers for assembling standard flag sets shared across subcommands.
//
// SRP: flag names and short usage strings are centralized here.
// Each Command implementation registers only the flags it needs via the
// registerXxx helpers and reads the parsed values afterward.

// registerScopeFlag registers --scope on fs and returns the destination
// string pointer. The default is "user". The usage string documents the
// "--scope=user|project" form.
func registerScopeFlag(fs *flag.FlagSet) *string {
	return fs.String("scope", "user", "configuration scope (user|project)")
}

// registerTargetFlag registers --target on fs and returns the
// destination string pointer. The default is "all". The usage string
// documents the "--target=claude|codex|gemini|all" form.
func registerTargetFlag(fs *flag.FlagSet) *string {
	return fs.String("target", "all", "distribution target (claude|codex|gemini|all)")
}

// registerOutputDirFlag registers -o / --output on fs and returns the
// destination string pointer. It is specific to the build subcommand,
// where output means "write built artifacts here instead of
// distributing them". The default is "", meaning build.Run will list
// artifacts on stdout without writing file contents.
//
// Both -o and --output write into the same pointer.
func registerOutputDirFlag(fs *flag.FlagSet) *string {
	p := fs.String("output", "", "output directory for build artifacts (when empty, lists artifacts to stdout)")
	fs.StringVar(p, "o", "", "shorthand for --output")
	return p
}
