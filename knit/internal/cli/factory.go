package cli

import (
	"fmt"
	"path/filepath"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/claude"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/codex"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/gemini"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// DistributionFactory is the factory that builds the Builder /
// Installer / Uninstaller / Lister for the Target selected by the
// `--target` flag value.
//
// SRP / SOLID notes:
//   - Each subcommand (install / uninstall / list / update / build)
//     translates only "user intent -> domain operation" and never imports
//     a target-specific package directly.
//   - The Target -> implementation mapping lives in exactly one place
//     (the providers table in this file). Adding a new Target is one
//     entry in providers plus the corresponding distribution package.
//   - Dependency direction: cli -> distribution/<target>. distribution
//     does not know cli.
//
// Lifecycle:
//   - DistributionFactory is built per subcommand execution.
//   - userBase / projectRoot / codexHome are passed in from
//     scopeResolver and Runtime.Getenv. Each provider closes over the
//     subdirectory and env-override conventions that turn those bases
//     into the absolute roots passed to NewInstaller / NewUninstaller /
//     NewLister.
type DistributionFactory struct {
	// userBase is the base directory for scope=user (usually $HOME).
	// projectRoot is the base directory for scope=project. An empty
	// projectRoot falls through to each distribution's
	// ErrProjectRootNotConfigured path at call time.
	userBase    string
	projectRoot string

	// codexHome holds the value of $CODEX_HOME (empty means unset). Only
	// the Codex provider consults it.
	codexHome string
}

// NewDistributionFactory constructs the factory.
func NewDistributionFactory(userBase, projectRoot, codexHome string) *DistributionFactory {
	return &DistributionFactory{
		userBase:    userBase,
		projectRoot: projectRoot,
		codexHome:   codexHome,
	}
}

// targetProvider bundles everything cli needs to talk to one distribution
// target: how to assemble its scope roots, and how to construct each role.
// Adding a new target = adding one entry to providers.
type targetProvider struct {
	target source.Target

	// userSubdir is appended to userBase to produce the ScopeUser
	// inventory root (e.g. ".claude").
	userSubdir string

	// projectSubdir is appended to projectRoot to produce the
	// ScopeProject inventory root.
	projectSubdir string

	// userRoot lets a provider override the default
	// `<userBase>/<userSubdir>` assembly. It receives the factory and
	// returns the absolute userRoot, or "" when it cannot be resolved.
	// Nil means "use the default".
	userRoot func(f *DistributionFactory) string

	newBuilder     func() source.Builder
	newInstaller   func(*DistributionFactory, inventory.LabelStore) (inventory.Installer, error)
	newUninstaller func(*DistributionFactory, inventory.LabelStore) (inventory.Uninstaller, error)
	newLister      func(*DistributionFactory, inventory.LabelStore) (inventory.Lister, error)
}

// providers is the single source of truth for "which distribution
// packages cli knows about". Order here defines SupportedTargets order.
var providers = []targetProvider{
	{
		target:        claude.Target,
		userSubdir:    ".claude",
		projectSubdir: ".claude",
		newBuilder:    func() source.Builder { return claude.NewBuilder() },
		newInstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Installer, error) {
			return claude.NewInstaller(joinRoot(f.userBase, ".claude"), joinRoot(f.projectRoot, ".claude"), l)
		},
		newUninstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Uninstaller, error) {
			return claude.NewUninstaller(joinRoot(f.userBase, ".claude"), joinRoot(f.projectRoot, ".claude"), l)
		},
		newLister: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Lister, error) {
			return claude.NewLister(joinRoot(f.userBase, ".claude"), joinRoot(f.projectRoot, ".claude"), l)
		},
	},
	{
		target:        codex.Target,
		userSubdir:    ".codex",
		projectSubdir: ".codex",
		userRoot: func(f *DistributionFactory) string {
			// Codex honors $CODEX_HOME as the absolute user root when set;
			// otherwise it falls back to the default `<userBase>/.codex`.
			if f.codexHome != "" {
				return f.codexHome
			}
			if f.userBase == "" {
				return ""
			}
			return filepath.Join(f.userBase, ".codex")
		},
		newBuilder: func() source.Builder { return codex.NewBuilder() },
		newInstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Installer, error) {
			return codex.NewInstallerWithRoots(codex.DefaultRoots(f.userBase, f.projectRoot, f.codexHome), l)
		},
		newUninstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Uninstaller, error) {
			return codex.NewUninstallerWithRoots(codex.DefaultRoots(f.userBase, f.projectRoot, f.codexHome), l)
		},
		newLister: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Lister, error) {
			return codex.NewListerWithRoots(codex.DefaultRoots(f.userBase, f.projectRoot, f.codexHome), l)
		},
	},
	{
		target:        gemini.Target,
		userSubdir:    ".gemini",
		projectSubdir: ".gemini",
		newBuilder:    func() source.Builder { return gemini.NewBuilder() },
		newInstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Installer, error) {
			return gemini.NewInstaller(joinRoot(f.userBase, ".gemini"), joinRoot(f.projectRoot, ".gemini"), l)
		},
		newUninstaller: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Uninstaller, error) {
			return gemini.NewUninstaller(joinRoot(f.userBase, ".gemini"), joinRoot(f.projectRoot, ".gemini"), l)
		},
		newLister: func(f *DistributionFactory, l inventory.LabelStore) (inventory.Lister, error) {
			return gemini.NewLister(joinRoot(f.userBase, ".gemini"), joinRoot(f.projectRoot, ".gemini"), l)
		},
	},
}

// providerByTarget indexes providers by Target for O(1) lookup. Built
// once at package init so adding a target only requires updating the
// providers slice.
var providerByTarget = func() map[source.Target]*targetProvider {
	m := make(map[source.Target]*targetProvider, len(providers))
	for i := range providers {
		m[providers[i].target] = &providers[i]
	}
	return m
}()

func (f *DistributionFactory) providerFor(target source.Target) (*targetProvider, error) {
	if p, ok := providerByTarget[target]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("%w: target=%q", ErrInvalidFlagValue, target)
}

// SupportedTargets returns the Targets recognized by the factory in the
// order declared in providers.
func (f *DistributionFactory) SupportedTargets() []source.Target {
	out := make([]source.Target, 0, len(providers))
	for _, p := range providers {
		out = append(out, p.target)
	}
	return out
}

// ResolveTargets expands the --target flag value ("claude" / "codex" /
// "gemini" / "all") into a Target slice.
func (f *DistributionFactory) ResolveTargets(flagValue string) ([]source.Target, error) {
	if flagValue == "all" {
		return f.SupportedTargets(), nil
	}
	for _, p := range providers {
		if string(p.target) == flagValue {
			return []source.Target{p.target}, nil
		}
	}
	return nil, fmt.Errorf("%w: target=%q (want claude|codex|gemini|all)", ErrInvalidFlagValue, flagValue)
}

// Builder returns the source.Builder for target.
func (f *DistributionFactory) Builder(target source.Target) (source.Builder, error) {
	p, err := f.providerFor(target)
	if err != nil {
		return nil, err
	}
	return p.newBuilder(), nil
}

// Installer returns the inventory.Installer for target.
func (f *DistributionFactory) Installer(target source.Target) (inventory.Installer, error) {
	p, err := f.providerFor(target)
	if err != nil {
		return nil, err
	}
	return p.newInstaller(f, f.labelStoreFor(target))
}

// Uninstaller returns the inventory.Uninstaller for target.
func (f *DistributionFactory) Uninstaller(target source.Target) (inventory.Uninstaller, error) {
	p, err := f.providerFor(target)
	if err != nil {
		return nil, err
	}
	return p.newUninstaller(f, f.labelStoreFor(target))
}

// Lister returns the inventory.Lister for target.
func (f *DistributionFactory) Lister(target source.Target) (inventory.Lister, error) {
	p, err := f.providerFor(target)
	if err != nil {
		return nil, err
	}
	return p.newLister(f, f.labelStoreFor(target))
}

// userRootFor returns the absolute ScopeUser Inventory root path for the
// given target by consulting the provider. Unknown targets resolve to "".
func (f *DistributionFactory) userRootFor(target source.Target) string {
	p, ok := providerByTarget[target]
	if !ok {
		return ""
	}
	if p.userRoot != nil {
		return p.userRoot(f)
	}
	if f.userBase == "" {
		return ""
	}
	return filepath.Join(f.userBase, p.userSubdir)
}

// projectRootFor returns the absolute ScopeProject Inventory root path
// for the given target by consulting the provider. Empty projectRoot
// short-circuits to "".
func (f *DistributionFactory) projectRootFor(target source.Target) string {
	if f.projectRoot == "" {
		return ""
	}
	p, ok := providerByTarget[target]
	if !ok {
		return ""
	}
	return filepath.Join(f.projectRoot, p.projectSubdir)
}

// userKnitRoot returns the absolute knit metadata root path for scope=user
// (typically "<userBase>/.knit"). Shared across all targets so knit-managed
// metadata stays out of any tool's own home directory.
func (f *DistributionFactory) userKnitRoot() string {
	if f.userBase == "" {
		return ""
	}
	return filepath.Join(f.userBase, ".knit")
}

// projectKnitRoot returns the absolute knit metadata root path for
// scope=project (typically "<projectRoot>/.knit"). Empty here causes
// ScopeProject operations to return ErrProjectRootNotConfigured.
func (f *DistributionFactory) projectKnitRoot() string {
	if f.projectRoot == "" {
		return ""
	}
	return filepath.Join(f.projectRoot, ".knit")
}

// labelStoreFor returns the inventory.LabelStore that the given target's
// Installer / Uninstaller / Lister should use for Label persistence.
func (f *DistributionFactory) labelStoreFor(target source.Target) inventory.LabelStore {
	return inventory.NewSidecarLabelStore(
		target,
		labelsRootOrEmpty(f.userKnitRoot()),
		labelsRootOrEmpty(f.projectKnitRoot()),
	)
}

func labelsRootOrEmpty(knitRoot string) string {
	if knitRoot == "" {
		return ""
	}
	return filepath.Join(knitRoot, "labels")
}

func joinRoot(base, subdir string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(base, subdir)
}
