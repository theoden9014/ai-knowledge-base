package cli

import "context"

// Execute is the single entry point called from knit's main.
//
// Responsibilities:
//  1. Build DefaultRuntime when rt is nil. The single source of truth
//     for args is [Runtime.Args], so Execute does not take args as a
//     parameter.
//  2. Construct an [App] with the standard command set
//     ([install / uninstall / list / update / build / help]), embedding
//     the binary name and version string.
//  3. Call App.Execute(ctx, rt) and return its result unchanged.
//
// Design intent:
//   - main.go remains a one-line wrapper that only passes this
//     function's return value to os.Exit. Command registration and DI
//     are centralized here.
//   - rt exists as an injection point for tests. Production callers pass
//     nil and let DefaultRuntime() be built internally.
//   - ctx exists as the path for injecting a context built externally,
//     such as by a signal handler. When ctx is nil,
//     context.Background() is used.
//   - name is the binary name, typically "knit". It is injectable so
//     tests can verify help output for alternate binary names.
func Execute(ctx context.Context, rt *Runtime, name, version string) ExitCode {
	if ctx == nil {
		ctx = context.Background()
	}
	if rt == nil {
		rt = DefaultRuntime()
	}
	base := defaultCommands()
	help := NewHelpCommand(base)
	commands := append(base, help)
	app := NewApp(name, version, commands)
	return app.Execute(ctx, rt)
}

// defaultCommands returns the subcommand set provided by this package by
// default. The order must remain stable so help output stays stable.
//
// The set consists of the five primary commands:
//   - install / uninstall / list / update / build
//
// help is not included here because it needs a reference to the full
// final command slice in order to display the list and Synopsis of all
// other commands. [Execute] injects it separately via [NewHelpCommand].
func defaultCommands() []Command {
	return []Command{
		NewInstallCommand(),
		NewUninstallCommand(),
		NewListCommand(),
		NewUpdateCommand(),
		NewBuildCommand(),
	}
}
