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

// updateCommand implements the `knit update <pack>` command.
//
// Semantics:
//   - Re-distribute existing artifacts from the latest source by running
//     "uninstall -> install" as a single transaction.
//   - Targets with no existing Installation are skipped by this command.
//
// Rollback policy on failure:
//   - "Best-effort, one-way": if uninstall succeeds and install fails,
//     the Installation remains removed. Full 2PC support is future work.
type updateCommand struct {
	scopeFlag  *string
	targetFlag *string
}

// NewUpdateCommand constructs the update subcommand.
func NewUpdateCommand() Command {
	return &updateCommand{}
}

// Name returns "update".
func (c *updateCommand) Name() string { return "update" }

// Synopsis returns a one-line description.
func (c *updateCommand) Synopsis() string {
	return "re-install a pack with the latest knowledge sources"
}

// Help returns the detailed help text.
func (c *updateCommand) Help() string {
	return `usage: knit update <pack-or-path-or-url> [--scope=user|project] [--target=claude|codex|gemini|all]

Re-distributes the named pack by uninstalling existing installations and
installing freshly-built artifacts. Targets without any prior installation
are skipped (use 'knit install' for first-time installs).

<pack-or-path-or-url> accepts the same forms as 'knit install': a pack
name, a local directory path, or a remote git URL.
`
}

// Flags returns a FlagSet containing --scope and --target.
func (c *updateCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	c.scopeFlag = registerScopeFlag(fs)
	c.targetFlag = registerTargetFlag(fs)
	return fs
}

// Run executes the flow described above.
//
// The <pack> argument is interpreted the same way as install and
// supports both local pack names and remote URLs via [loadPackFromArg].
// Prior installations are matched using [resolvedPack.Name]
// (= Pack.Name = the `pack:` field in manifest.yaml), which allows the
// same key to be used for both local and remote sources.
func (c *updateCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
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
		if err := c.updateForTarget(ctx, rt, factory, rp.Pack, target, scope, rp.Name); err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
		}
	}
	return aggregateOrSingle(len(targets), failures)
}

func (c *updateCommand) updateForTarget(
	ctx context.Context,
	rt *Runtime,
	factory *DistributionFactory,
	pack *source.Pack,
	target source.Target,
	scope inventory.Scope,
	packName string,
) error {
	lister, err := factory.Lister(target)
	if err != nil {
		return err
	}
	uninstaller, err := factory.Uninstaller(target)
	if err != nil {
		return err
	}
	installer, err := factory.Installer(target)
	if err != nil {
		return err
	}
	builder, err := factory.Builder(target)
	if err != nil {
		return err
	}

	insts, err := lister.List(ctx, scope)
	if err != nil {
		return err
	}
	matchingPriors := 0
	for _, inst := range insts {
		if !installationBelongsToPack(inst, packName) {
			continue
		}
		matchingPriors++
		if err := uninstaller.Uninstall(ctx, inst); err != nil {
			if errors.Is(err, inventory.ErrInstallationNotFound) {
				continue
			}
			return fmt.Errorf("update/uninstall %s/%s: %w", target, inst.ID, err)
		}
	}
	if matchingPriors == 0 {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: nothing to update for %s/%s (no prior installation of %q)\n", target, scope, packName)
		return nil
	}

	artifacts, err := builder.Build(ctx, pack)
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		// Prior installations were already removed above (matchingPriors >0)
		// but the new pack content has no artifacts for this target. Be
		// explicit so the user sees this as an intentional skip, not a
		// silent regression.
		_, _ = fmt.Fprintf(rt.Stdout, "skipped %s/%s (no artifacts for this pack; prior installations removed)\n", target, scope)
		return nil
	}
	for _, art := range artifacts {
		if _, err := installer.Install(ctx, scope, art); err != nil {
			return fmt.Errorf("update/install %s/%s: %w (re-run to recover)", target, art.Path, err)
		}
	}
	_, _ = fmt.Fprintf(rt.Stdout, "updated %d artifacts in %s/%s\n", len(artifacts), target, scope)
	return nil
}
