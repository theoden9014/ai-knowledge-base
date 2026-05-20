package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
)

// buildCommand implements the `knit build <pack> --target=<target>` command.
//
// Semantics:
//   - A debug-oriented command that does not distribute anything and only
//     produces the Builder's output artifacts.
//   - --target accepts only a single Target (`all` is rejected here).
//   - If -o <dir> is specified, artifacts are written under that directory.
//   - If -o is omitted, only the artifact list is printed to stdout
//     (without contents).
type buildCommand struct {
	targetFlag    *string
	outputDirFlag *string
}

// NewBuildCommand constructs the build subcommand.
func NewBuildCommand() Command {
	return &buildCommand{}
}

// Name returns "build".
func (c *buildCommand) Name() string { return "build" }

// Synopsis returns a one-line description.
func (c *buildCommand) Synopsis() string {
	return "build a pack's artifacts without distributing them (debug)"
}

// Help returns the detailed help text.
func (c *buildCommand) Help() string {
	return `usage: knit build <pack-or-path-or-url> --target=claude|codex|gemini [-o <dir>]

Builds the named pack for a single target. Without -o, prints the artifact
list to stdout. With -o, writes each artifact to <dir>/<artifact-path>.

<pack-or-path-or-url> accepts the same forms as 'knit install': a pack
name, a local directory path, or a remote git URL.
`
}

// Flags returns a FlagSet containing --target and -o.
// (build does not take scope because it does not distribute anything.)
func (c *buildCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	c.targetFlag = registerTargetFlag(fs)
	c.outputDirFlag = registerOutputDirFlag(fs)
	return fs
}

// Run executes the flow described above.
//
// The <pack> argument is triaged by [loadPackFromArg], just like install
// and update, and accepts pack names, local directory paths, and remote
// URLs. --target=all is rejected for this command.
func (c *buildCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	packArg, err := requirePackArg(fs)
	if err != nil {
		return err
	}
	if *c.targetFlag == "all" {
		return fmt.Errorf("%w: build requires a single --target (got %q)", ErrInvalidFlagValue, *c.targetFlag)
	}
	// reuse factory plumbing but with a placeholder userBase: build does
	// not touch the inventory, so empty roots are fine.
	factory := NewDistributionFactory("", "", "")
	targets, err := factory.ResolveTargets(*c.targetFlag)
	if err != nil {
		return err
	}
	if len(targets) != 1 {
		// defense-in-depth: ResolveTargets should not return multiple here.
		return fmt.Errorf("%w: build requires exactly one target, got %d", ErrInvalidFlagValue, len(targets))
	}
	target := targets[0]
	rp, err := loadPackFromArg(ctx, rt, packArg)
	if err != nil {
		return err
	}
	defer cleanupResolvedPack(rt, rp)
	builder, err := factory.Builder(target)
	if err != nil {
		return err
	}
	artifacts, err := builder.Build(ctx, rp.Pack)
	if err != nil {
		return err
	}
	if *c.outputDirFlag == "" {
		printArtifactSummary(rt.Stdout, artifacts)
		return nil
	}
	for _, art := range artifacts {
		if err := writeArtifactToDir(*c.outputDirFlag, art); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(rt.Stdout, "wrote %d artifacts to %s\n", len(artifacts), *c.outputDirFlag)
	return nil
}
