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
// From an SRP perspective:
//   - Each subcommand (install / uninstall / list / update / build)
//     translates only "user intent -> domain operation" and does not
//     know about target-specific concrete types
//     (claude.NewInstaller, codex.NewInstaller, ...).
//   - All mapping from a target string to a concrete type is centralized
//     in this factory.
//   - Adding a new Target is completed by a one-line addition here plus
//     the corresponding implementation in distribution/<target>.
//
// Dependency direction:
//   - cli -> distribution/<target>. distribution does not know cli.
//   - This factory must be updated whenever a new Target is added. That
//     is a deliberate design choice that prefers a clear aggregation
//     point over strict OCP.
//
// Lifecycle:
//   - DistributionFactory is built per subcommand execution.
//   - userBase / projectRoot are passed in from scopeResolver. Building
//     target-specific subdirectories (`.claude` / `.codex` / `.gemini`)
//     happens inside this factory, so the convention for assembling the
//     userRoot / projectRoot arguments passed to each distribution's
//     NewXxx is closed over here.
type DistributionFactory struct {
	// userBase is the base directory for scope=user (usually $HOME).
	// projectRoot is the base directory for scope=project. If it is an
	// empty string, scope=project operations are not detected at factory
	// return time and instead fall through to the path where each
	// distribution returns ErrProjectRootNotConfigured.
	userBase    string
	projectRoot string

	// codexHome is the value of the $CODEX_HOME environment variable
	// (empty string means unset). Because only Codex has the convention of
	// using $CODEX_HOME as the ScopeUser Inventory root, this factory
	// decides whether to override the default here.
	codexHome string
}

// NewDistributionFactory constructs the factory from userBase /
// projectRoot resolved by scopeResolver and codexHome retrieved from
// Runtime.Getenv.
//
// Arguments:
//   - userBase: the starting directory for scope=user (usually $HOME).
//     If it is an empty string, behavior is undefined for distributions
//     that require scope=user. The caller (each command) is responsible
//     for passing this factory only after scope=user resolution has
//     succeeded.
//   - projectRoot: the starting directory for scope=project
//     (empty string means unresolved).
//   - codexHome: the value of $CODEX_HOME (empty string means unset).
func NewDistributionFactory(userBase, projectRoot, codexHome string) *DistributionFactory {
	return &DistributionFactory{
		userBase:    userBase,
		projectRoot: projectRoot,
		codexHome:   codexHome,
	}
}

// SupportedTargets returns the Targets recognized by the factory in
// kebab-case alphabetical order. It is used for expanding
// `--target=all` and for validating unknown targets.
func (f *DistributionFactory) SupportedTargets() []source.Target {
	return []source.Target{claude.Target, codex.Target, gemini.Target}
}

// ResolveTargets expands the --target flag value ("claude" / "codex" /
// "gemini" / "all") into a Target slice.
//
// Contract:
//   - "all" is equivalent to SupportedTargets().
//   - Unknown strings return ErrInvalidFlagValue.
//   - The result preserves the stable order of SupportedTargets().
func (f *DistributionFactory) ResolveTargets(flagValue string) ([]source.Target, error) {
	if flagValue == "all" {
		return f.SupportedTargets(), nil
	}
	for _, t := range f.SupportedTargets() {
		if string(t) == flagValue {
			return []source.Target{t}, nil
		}
	}
	return nil, fmt.Errorf("%w: target=%q (want claude|codex|gemini|all)", ErrInvalidFlagValue, flagValue)
}

// Builder returns the source.Builder for target.
// Unknown targets return ErrInvalidFlagValue.
func (f *DistributionFactory) Builder(target source.Target) (source.Builder, error) {
	switch target {
	case claude.Target:
		return claude.NewBuilder(), nil
	case codex.Target:
		return codex.NewBuilder(), nil
	case gemini.Target:
		return gemini.NewBuilder(), nil
	default:
		return nil, fmt.Errorf("%w: target=%q", ErrInvalidFlagValue, target)
	}
}

// Installer returns the inventory.Installer for target.
// Unknown targets return ErrInvalidFlagValue.
func (f *DistributionFactory) Installer(target source.Target) (inventory.Installer, error) {
	switch target {
	case claude.Target:
		return claude.NewInstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case codex.Target:
		return codex.NewInstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case gemini.Target:
		return gemini.NewInstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	default:
		return nil, fmt.Errorf("%w: target=%q", ErrInvalidFlagValue, target)
	}
}

// Uninstaller returns the inventory.Uninstaller for target.
// Unknown targets return ErrInvalidFlagValue.
func (f *DistributionFactory) Uninstaller(target source.Target) (inventory.Uninstaller, error) {
	switch target {
	case claude.Target:
		return claude.NewUninstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case codex.Target:
		return codex.NewUninstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case gemini.Target:
		return gemini.NewUninstaller(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	default:
		return nil, fmt.Errorf("%w: target=%q", ErrInvalidFlagValue, target)
	}
}

// Lister returns the inventory.Lister for target.
// Unknown targets return ErrInvalidFlagValue.
func (f *DistributionFactory) Lister(target source.Target) (inventory.Lister, error) {
	switch target {
	case claude.Target:
		return claude.NewLister(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case codex.Target:
		return codex.NewLister(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	case gemini.Target:
		return gemini.NewLister(f.userRootFor(target), f.projectRootFor(target), f.labelStoreFor(target))
	default:
		return nil, fmt.Errorf("%w: target=%q", ErrInvalidFlagValue, target)
	}
}

// userRootFor returns the absolute ScopeUser Inventory root path for the
// given target.
// (Example: claude -> "<userBase>/.claude", codex -> codexHome if set,
// otherwise "<userBase>/.codex", gemini -> "<userBase>/.gemini")
//
// This method is an internal helper used by Builder / Installer /
// Uninstaller / Lister and is not exported.
func (f *DistributionFactory) userRootFor(target source.Target) string {
	switch target {
	case claude.Target:
		if f.userBase == "" {
			return ""
		}
		return filepath.Join(f.userBase, ".claude")
	case codex.Target:
		if f.codexHome != "" {
			return f.codexHome
		}
		if f.userBase == "" {
			return ""
		}
		return filepath.Join(f.userBase, ".codex")
	case gemini.Target:
		if f.userBase == "" {
			return ""
		}
		return filepath.Join(f.userBase, ".gemini")
	default:
		return ""
	}
}

// projectRootFor returns the absolute ScopeProject Inventory root path
// for the given target. If projectRoot is an empty string, it returns an
// empty string and delegates to each distribution's
// ErrProjectRootNotConfigured path.
//
// Example: claude -> "<projectRoot>/.claude",
// codex -> "<projectRoot>/.codex",
// gemini -> "<projectRoot>/.gemini"
func (f *DistributionFactory) projectRootFor(target source.Target) string {
	if f.projectRoot == "" {
		return ""
	}
	switch target {
	case claude.Target:
		return filepath.Join(f.projectRoot, ".claude")
	case codex.Target:
		return filepath.Join(f.projectRoot, ".codex")
	case gemini.Target:
		return filepath.Join(f.projectRoot, ".gemini")
	default:
		return ""
	}
}

// userKnitRoot returns the absolute knit metadata root path for scope=user
// (typically "<userBase>/.knit"). Unlike userRootFor, this is shared across
// all targets because knit-managed metadata is intentionally kept out of any
// individual AI tool's home directory.
//
// Returns an empty string when userBase is empty; in that case each
// distribution's behavior matches userRoot being empty (undefined for
// ScopeUser; caller responsibility).
func (f *DistributionFactory) userKnitRoot() string {
	if f.userBase == "" {
		return ""
	}
	return filepath.Join(f.userBase, ".knit")
}

// projectKnitRoot returns the absolute knit metadata root path for
// scope=project (typically "<projectRoot>/.knit"). Like projectRootFor, an
// empty string here causes ScopeProject operations to return
// ErrProjectRootNotConfigured inside the distribution layer.
func (f *DistributionFactory) projectKnitRoot() string {
	if f.projectRoot == "" {
		return ""
	}
	return filepath.Join(f.projectRoot, ".knit")
}

// labelStoreFor returns the inventory.LabelStore that the given target's
// Installer / Uninstaller / Lister should use for Label persistence.
//
// The default backend is inventory.SidecarLabelStore with both
// "<userKnitRoot>/labels" and "<projectKnitRoot>/labels" wired in. The store
// selects between them per call based on the scope argument, so callers do
// not need to branch between user / project stores.
//
// If userKnitRoot or projectKnitRoot is empty (for example because the CLI
// could not discover the project root), the corresponding scope operation
// returns ErrLabelsRootNotConfigured from inside the store.
func (f *DistributionFactory) labelStoreFor(target source.Target) inventory.LabelStore {
	return inventory.NewSidecarLabelStore(
		target,
		labelsRootOrEmpty(f.userKnitRoot()),
		labelsRootOrEmpty(f.projectKnitRoot()),
	)
}

// labelsRootOrEmpty maps a knit root path to its `<knitRoot>/labels`
// subdirectory, preserving the empty-string sentinel so the store can return
// ErrLabelsRootNotConfigured for unconfigured scopes.
func labelsRootOrEmpty(knitRoot string) string {
	if knitRoot == "" {
		return ""
	}
	return filepath.Join(knitRoot, "labels")
}
