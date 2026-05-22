package source

// Pack is a fully loaded knowledge pack: the manifest metadata together with
// every Entry referenced by the manifest. A Pack is the unit of input for
// Builder and is treated as immutable by every consumer.
//
// Pack does not carry filesystem provenance (such as the directory the pack
// was read from). That information belongs to LoadInfo, returned alongside
// Pack by Loader.LoadPack, so consumers cannot inadvertently use Pack to
// drive additional I/O.
type Pack struct {
	// Name is the pack name (kebab-case), matching the directory name and
	// the manifest's pack field.
	Name string

	// Version is the pack semver.
	Version string

	// Description is the pack-level human-readable summary.
	Description string

	// DefaultTools lists the targets enabled by default for entries that
	// omit Tools[target].Enabled.
	DefaultTools []Target

	// Entries holds the loaded entries in manifest order.
	Entries []Entry
}

// EntriesFor returns the entries that resolve to enabled for the given
// target. The resolution rule is the single source of truth for enabled
// state: Entry.Tools[target].Enabled wins when present, otherwise the entry
// is enabled iff target appears in Pack.DefaultTools. The returned pointers
// are read-only handles into Pack.Entries; consumers must not mutate the
// pointees.
func (p *Pack) EntriesFor(target Target) []*Entry {
	var out []*Entry
	for i := range p.Entries {
		if p.isEnabled(&p.Entries[i], target) {
			out = append(out, &p.Entries[i])
		}
	}
	return out
}

func (p *Pack) isEnabled(e *Entry, target Target) bool {
	if cfg, ok := e.Tools[target]; ok && cfg.Enabled != nil {
		return *cfg.Enabled
	}
	for _, t := range p.DefaultTools {
		if t == target {
			return true
		}
	}
	return false
}
