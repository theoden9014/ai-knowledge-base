package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// listCommand implements the `knit list` command.
type listCommand struct {
	scopeFlag  *string
	targetFlag *string
}

// NewListCommand constructs the list subcommand.
func NewListCommand() Command {
	return &listCommand{}
}

// Name returns "list".
func (c *listCommand) Name() string { return "list" }

// Synopsis returns a one-line description.
func (c *listCommand) Synopsis() string {
	return "list installed knowledge packs"
}

// Help returns the detailed help text.
func (c *listCommand) Help() string {
	return `usage: knit list [--scope=user|project] [--target=claude|codex|gemini|all]

Lists Installations recorded under the requested scope/target inventories.
Columns: TARGET / SCOPE / ID / SOURCE_ENTRIES.
`
}

// Flags returns a FlagSet containing --scope and --target.
func (c *listCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	c.scopeFlag = registerScopeFlag(fs)
	c.targetFlag = registerTargetFlag(fs)
	return fs
}

// Run executes the flow described above.
func (c *listCommand) Run(ctx context.Context, rt *Runtime, fs *flag.FlagSet) error {
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: list takes no positional arguments", ErrUsage)
	}
	scope, targets, factory, err := buildFactory(rt, *c.scopeFlag, *c.targetFlag)
	if err != nil {
		return err
	}

	type row struct {
		target  string
		scope   string
		id      string
		entries string
	}
	var rows []row
	var failures []TargetFailure
	for _, target := range targets {
		lister, lerr := factory.Lister(target)
		if lerr != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: lerr})
			continue
		}
		insts, lerr := lister.List(ctx, scope)
		if lerr != nil {
			failures = append(failures, TargetFailure{Target: string(target), Err: lerr})
			continue
		}
		for _, inst := range insts {
			rows = append(rows, row{
				target:  string(inst.Label.Target),
				scope:   string(inst.Label.Scope),
				id:      string(inst.ID),
				entries: strings.Join(inst.Provenance.SourceEntryIDs, ","),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].target != rows[j].target {
			return rows[i].target < rows[j].target
		}
		if rows[i].scope != rows[j].scope {
			return rows[i].scope < rows[j].scope
		}
		return rows[i].id < rows[j].id
	})

	_, _ = fmt.Fprintf(rt.Stdout, "%-10s %-8s %-40s %s\n", "TARGET", "SCOPE", "ID", "SOURCE_ENTRIES")
	for _, r := range rows {
		_, _ = fmt.Fprintf(rt.Stdout, "%-10s %-8s %-40s %s\n", r.target, r.scope, r.id, r.entries)
	}
	return aggregateOrSingle(len(targets), failures)
}
