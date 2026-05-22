package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// uninstallCommand implements the `knit uninstall <pack>` command.
//
// Processing flow (Run's responsibility):
//  1. Extract a single <pack> from fs.Args() (same rule as install).
//  2. Parse --scope / --target using the same rules as install.
//  3. Resolve userBase / projectRoot via scopeResolver.
//  4. For each expanded Target:
//     a. Enumerate all Installations for the requested scope with
//     factory.Lister(target).List(ctx, scope).
//     b. Inspect Installation.Provenance.SourceEntryIDs and select
//     entries whose <pack> segment matches the input argument.
//     c. Call Uninstaller.Uninstall for each selected Installation.
//     Failures are aggregated with AggregateError just like install.
//  5. inventory.ErrInstallationNotFound is absorbed within this command
//     and downgraded to a warning on stderr to preserve idempotency. If
//     no matching installations are found at all, emit a warning to
//     stderr and exit with ExitSuccess.
type uninstallCommand struct {
	scopeFlag  *string
	targetFlag *string
}

// NewUninstallCommand constructs the uninstall subcommand.
func NewUninstallCommand() Command {
	return &uninstallCommand{}
}

// Name returns "uninstall".
func (c *uninstallCommand) Name() string { return "uninstall" }

// Synopsis returns a one-line description.
func (c *uninstallCommand) Synopsis() string {
	return "remove a previously installed knowledge pack"
}

// Help returns the detailed help text.
func (c *uninstallCommand) Help() string {
	return `usage: knit uninstall <pack-or-path> [--scope=user|project] [--target=claude|codex|gemini|all]

Removes the artifacts previously installed for the given pack, identified
by the Installation labels written at install time.

<pack-or-path> accepts:
  - a kebab-case pack name (used verbatim; knowledge/ does not have to
    exist, so packs previously installed from a now-unreachable source
    can still be uninstalled).
  - a local directory path (absolute or relative; must contain "/" or
    start with "./", "../", or "/") pointing at a pack directory whose
    manifest.yaml is read only to recover the canonical Pack.Name.
  - remote URLs (e.g. github.com/owner/repo) are rejected with ErrUsage.

Idempotency: when no installations are found for the requested pack/target
pair this command emits a warning to stderr and exits with ExitSuccess.
`
}

// Flags returns a FlagSet containing --scope and --target.
// (uninstall does not register --knowledge-dir because it does not read
// knowledge/.)
func (c *uninstallCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	c.scopeFlag = registerScopeFlag(fs)
	c.targetFlag = registerTargetFlag(fs)
	return fs
}

// Run executes the flow described above.
//
// The <pack> argument accepts **local pack names only**. Passing a
// remote URL returns ErrUsage.
//
// Design decision:
//   - uninstall matches by extracting the <pack> segment from
//     Provenance.SourceEntryIDs of the form <pack>.<kind>.<entry> on the
//     Installations returned by Lister.
//   - The normalization rule from a remote URL to this <pack> name is
//     not yet defined in the current phase, so remote URLs are not
//     allowed to avoid unintended deletions caused by false matches.
//   - If support is added later, the command should read Pack.Name via
//     loadPackFromArg and use that as the key. Pack.Name comes from the
//     manifest.yaml `pack:` field and is uniquely determined for both
//     local and remote inputs, so that value can be passed to
//     installationBelongsToPack.
func (c *uninstallCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	arg, err := requirePackArg(fs)
	if err != nil {
		return err
	}
	triaged := TriageArg(arg)
	// Remote URL arguments are rejected unconditionally for uninstall —
	// the URL → local pack name normalization is future work (see the
	// Run godoc above for the rationale).
	if triaged.Kind == ArgKindRemoteURL {
		return fmt.Errorf("%w: uninstall accepts a local pack name or directory path only, not a remote URL", ErrUsage)
	}
	// Determine the canonical pack name to match against installation
	// provenance. For kebab-case pack names, use the argument verbatim so
	// the command does not require knowledge/ to exist (uninstall must
	// work even when the source pack is no longer reachable, e.g. for
	// previously remote-installed packs). For local directory paths,
	// load the manifest only to read Pack.Name.
	var packName string
	switch triaged.Kind {
	case ArgKindLocalPath:
		pack, err := loadPackFromLocalDir(ctx, rt, triaged.Cleaned)
		if err != nil {
			return err
		}
		packName = pack.Name
	default:
		packName = triaged.Cleaned
	}

	scope, targets, factory, err := buildFactory(rt, *c.scopeFlag, *c.targetFlag)
	if err != nil {
		return err
	}

	var failures []TargetFailure
	totalRemoved := 0
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
			continue
		}
		removed, err := c.uninstallForTarget(ctx, rt, factory, target, scope, packName)
		if err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
			continue
		}
		totalRemoved += removed
	}
	if totalRemoved == 0 && len(failures) == 0 {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: no installations of %q found\n", packName)
	}
	return aggregateOrSingle(len(targets), failures)
}

func (c *uninstallCommand) uninstallForTarget(
	ctx context.Context,
	rt *Runtime,
	factory *DistributionFactory,
	target source.Target,
	scope inventory.Scope,
	packName string,
) (int, error) {
	lister, err := factory.Lister(target)
	if err != nil {
		return 0, err
	}
	uninstaller, err := factory.Uninstaller(target)
	if err != nil {
		return 0, err
	}
	installations, err := lister.List(ctx, scope)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, inst := range installations {
		if !inst.Provenance.BelongsToPack(packName) {
			continue
		}
		if err := uninstaller.Uninstall(ctx, inst); err != nil {
			if errors.Is(err, inventory.ErrInstallationNotFound) {
				_, _ = fmt.Fprintf(rt.Stderr, "warning: %s/%s already gone\n", target, inst.ID)
				continue
			}
			return removed, fmt.Errorf("uninstall %s/%s: %w", target, inst.ID, err)
		}
		removed++
	}
	if removed > 0 {
		_, _ = fmt.Fprintf(rt.Stdout, "removed %d installations from %s/%s\n", removed, target, scope)
	}
	return removed, nil
}
