package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// installCommand implements the `knit install <pack>` command.
//
// Processing flow (Run's responsibility):
//  1. Extract a single <pack-or-path-or-url> from fs.Args(). If it is
//     missing, return ErrMissingArgument. If extra positional args are
//     present, return ErrUsage.
//  2. Normalize --scope via validateScope.
//  3. Expand --target via DistributionFactory.ResolveTargets
//     ("all" -> all Targets).
//  4. Resolve userBase / projectRoot via scopeResolver.
//     - scope=user with HOME unset -> ErrHomeNotSet
//     - scope=project with project root not found -> ErrProjectRootNotFound
//  5. Load the pack via [loadPackFromArg], which performs the
//     remote-URL / local-path / pack-name triage internally.
//  6. For each expanded Target:
//     a. Generate artifacts with factory.Builder(target).Build(ctx, pack)
//     b. Execute factory.Installer(target).Install(ctx, scope, artifact)
//     for each artifact
//     Failures for individual Targets are aggregated into AggregateError
//     and returned as ErrPartialFailure at the end
//     (--target=all with partial failure).
//     If only a single Target was requested, return the error directly
//     instead of aggregating it.
//  7. On success, write the installed item count for each Target / Scope
//     to stdout.
type installCommand struct {
	scopeFlag  *string
	targetFlag *string
}

// NewInstallCommand constructs the install subcommand.
func NewInstallCommand() Command {
	return &installCommand{}
}

// Name returns "install".
func (c *installCommand) Name() string { return "install" }

// Synopsis returns a one-line description.
func (c *installCommand) Synopsis() string {
	return "install a knowledge pack into one or more AI tools"
}

// Help returns the detailed help text.
func (c *installCommand) Help() string {
	return `usage: knit install <pack-or-path-or-url> [--scope=user|project] [--target=claude|codex|gemini|all]

Builds the named pack and installs the resulting artifacts under the
configured inventory for each target.

<pack-or-path-or-url> accepts:
  - a kebab-case pack name (e.g. "structure-behavior-design"),
    resolved against the auto-detected knowledge/ directory.
  - a local directory path (absolute or relative; must contain "/" or
    start with "./", "../", or "/") pointing directly at a pack
    directory containing manifest.yaml.
  - a remote git URL (github.com/<owner>/<repo>[/<subpath>], optionally
    with an http(s):// prefix). The repository is cloned to a temp dir
    via git and removed when the command exits.

flags:
  --scope   configuration scope (default: user)
  --target  distribution target (default: all)
`
}

// Flags returns a FlagSet containing --scope and --target.
func (c *installCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	c.scopeFlag = registerScopeFlag(fs)
	c.targetFlag = registerTargetFlag(fs)
	return fs
}

// Run executes the flow described above.
//
// The <pack> argument is triaged by [loadPackFromArg] and supports both
// local pack names and remote URLs (for example,
// "github.com/owner/repo[/subpath]" or "https://github.com/...").
// For remote sources, the repository is cloned into a temporary
// directory and removed by the deferred Cleanup() at the end of this
// function.
func (c *installCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	packArg, err := requirePackArg(fs)
	if err != nil {
		return err
	}
	scope, targets, factory, err := buildFactory(rt, *c.scopeFlag, *c.targetFlag)
	if err != nil {
		return err
	}
	rp, err := loadPackFromArg(ctx, rt, packArg)
	if err != nil {
		return err
	}
	defer cleanupResolvedPack(rt, rp)

	var failures []TargetFailure
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
			continue
		}
		if err := c.installForTarget(ctx, rt, factory, rp.Pack, rp.Source, target, scope); err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
		}
	}
	return aggregateOrSingle(len(targets), failures)
}

func (c *installCommand) installForTarget(
	ctx context.Context,
	rt *Runtime,
	factory *DistributionFactory,
	pack *source.Pack,
	ref source.SourceRef,
	target source.Target,
	scope inventory.Scope,
) error {
	builder, err := factory.Builder(target)
	if err != nil {
		return err
	}
	artifacts, err := builder.Build(ctx, pack)
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		// The pack has no Entry for this target (false in both
		// default_tools and the per-entry override). This is not a
		// failure; report it explicitly to stdout as "not applicable".
		_, _ = fmt.Fprintf(rt.Stdout, "skipped %s/%s (no artifacts for this pack)\n", target, scope)
		return nil
	}
	installer, err := factory.Installer(target)
	if err != nil {
		return err
	}
	for _, art := range artifacts {
		art.SourceRef = ref
		if _, err := installer.Install(ctx, scope, art); err != nil {
			return fmt.Errorf("install %s/%s: %w", target, art.Path, err)
		}
	}
	_, _ = fmt.Fprintf(rt.Stdout, "installed %d artifacts to %s/%s\n", len(artifacts), target, scope)
	return nil
}
