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
Columns: TARGET / SCOPE / PACK / ENTRY_ID / PATH.
  - PACK is the knowledge pack name (e.g., "structure-behavior-design").
  - ENTRY_ID is the neutral <pack>.<kind>.<name> identifier; if an Installation
    aggregates multiple Entries (such as a folded rule file), the values are
    comma-separated.
  - PATH is the artifact placement path relative to the Inventory root.
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
		pack    string
		entryID string
		path    string
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
				pack:    strings.Join(inst.Provenance.Packs(), ","),
				entryID: strings.Join(inst.Provenance.SourceEntryIDs, ","),
				path:    string(inst.ID),
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
		if rows[i].pack != rows[j].pack {
			return rows[i].pack < rows[j].pack
		}
		return rows[i].entryID < rows[j].entryID
	})

	_, _ = fmt.Fprintf(rt.Stdout, "%-10s %-8s %-30s %-60s %s\n", "TARGET", "SCOPE", "PACK", "ENTRY_ID", "PATH")
	for _, r := range rows {
		_, _ = fmt.Fprintf(rt.Stdout, "%-10s %-8s %-30s %-60s %s\n", r.target, r.scope, r.pack, r.entryID, r.path)
	}
	return aggregateOrSingle(len(targets), failures)
}
