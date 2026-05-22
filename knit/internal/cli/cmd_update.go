package cli

import (
	"context"
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
//   - A bare pack name refreshes only packs installed from a recorded remote
//     source. Local-source installs must be updated with an explicit path.
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

Re-distributes an installed pack by uninstalling existing artifacts and
installing freshly-built artifacts.

When a pack name is given, knit uses the remote URL recorded at install
time. Packs installed from local sources must be updated with an explicit
local path. Targets without any prior installation are skipped (use
'knit install' for first-time installs).

<pack-or-path-or-url> accepts:
  - a pack name for remote-installed packs (e.g. "structure-behavior-design")
  - a local directory path pointing directly at a pack directory
  - a remote git URL
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
// For a bare pack name, update first looks for a recorded remote source on
// existing installations. Local-source installations are rejected so the user
// must pass the source path explicitly. Prior installations are matched using
// [resolvedPack.Name] (= Pack.Name = the `pack:` field in manifest.yaml),
// which allows explicit local paths and remote URLs to use the same key.
func (c *updateCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	packArg, err := requirePackArg(fs)
	if err != nil {
		return err
	}
	scope, targets, factory, err := buildFactory(rt, *c.scopeFlag, *c.targetFlag)
	if err != nil {
		return err
	}
	triaged := TriageArg(packArg)
	if triaged.Kind == ArgKindPackName {
		ref, ok, err := c.remoteSourceForPackName(ctx, factory, targets, scope, triaged.Cleaned)
		if err != nil {
			return err
		}
		if !ok {
			for _, target := range targets {
				_, _ = fmt.Fprintf(rt.Stderr, "warning: nothing to update for %s/%s (no prior installation of %q)\n", target, scope, triaged.Cleaned)
			}
			return nil
		}
		packArg = ref.Value
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
		if err := c.updateForTarget(ctx, rt, factory, rp.Pack, rp.Source, target, scope, rp.Name); err != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: err})
		}
	}
	return aggregateOrSingle(len(targets), failures)
}

func (c *updateCommand) remoteSourceForPackName(
	ctx context.Context,
	factory *DistributionFactory,
	targets []source.Target,
	scope inventory.Scope,
	packName string,
) (source.SourceRef, bool, error) {
	var found source.SourceRef
	hasMatch := false
	for _, target := range targets {
		lister, err := factory.Lister(target)
		if err != nil {
			return source.SourceRef{}, false, err
		}
		insts, err := lister.List(ctx, scope)
		if err != nil {
			return source.SourceRef{}, false, err
		}
		for _, inst := range insts {
			if !inst.Provenance.BelongsToPack(packName) {
				continue
			}
			hasMatch = true
			ref := inst.Provenance.SourceRef
			switch {
			case ref.Kind == source.SourceRefRemoteURL && ref.Value != "":
				if found.IsZero() {
					found = ref
					continue
				}
				if found != ref {
					return source.SourceRef{}, false, fmt.Errorf(
						"%w: multiple remote sources recorded for %q; pass the source URL explicitly",
						ErrUsage, packName,
					)
				}
			case ref.Kind == source.SourceRefLocalPath:
				return source.SourceRef{}, false, fmt.Errorf(
					"%w: pack %q was installed from a local source; pass the source path explicitly",
					ErrUsage, packName,
				)
			default:
				return source.SourceRef{}, false, fmt.Errorf(
					"%w: pack %q has no recorded remote source; pass the source path or URL explicitly",
					ErrUsage, packName,
				)
			}
		}
	}
	if !hasMatch {
		return source.SourceRef{}, false, nil
	}
	if found.IsZero() {
		return source.SourceRef{}, false, fmt.Errorf(
			"%w: pack %q has no recorded remote source; pass the source path or URL explicitly",
			ErrUsage, packName,
		)
	}
	return found, true, nil
}

func (c *updateCommand) updateForTarget(
	ctx context.Context,
	rt *Runtime,
	factory *DistributionFactory,
	pack *source.Pack,
	ref source.SourceRef,
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
	reinstaller, err := inventory.NewReinstaller(installer, uninstaller, lister)
	if err != nil {
		return err
	}

	artifacts, err := builder.Build(ctx, pack)
	if err != nil {
		return err
	}
	for i := range artifacts {
		artifacts[i].SourceRef = ref
	}

	report, err := reinstaller.Reinstall(ctx, scope, packName, artifacts)
	if err != nil {
		return fmt.Errorf("update %s/%s: %w", target, scope, err)
	}
	switch {
	case report.NoPriorInstallation:
		_, _ = fmt.Fprintf(rt.Stderr, "warning: nothing to update for %s/%s (no prior installation of %q)\n", target, scope, packName)
	case len(artifacts) == 0:
		// Prior installations were already removed but the new pack
		// content has no artifacts for this target. Surface the skip so
		// the user sees it as intentional rather than a silent regression.
		_, _ = fmt.Fprintf(rt.Stdout, "skipped %s/%s (no artifacts for this pack; prior installations removed)\n", target, scope)
	default:
		_, _ = fmt.Fprintf(rt.Stdout, "updated %d artifacts in %s/%s\n", report.InstalledCount, target, scope)
	}
	return nil
}
