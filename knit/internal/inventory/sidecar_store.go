package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// SidecarLabelStore is the default LabelStore implementation. It persists
// Labels under the knit root as one JSON sidecar file per Label.
//
// Directory layout:
//
//	<labelsRoot>/<target>/<scope>/<encoded-id>.json
//
// labelsRoot is selected per scope:
//   - ScopeUser   : userLabelsRoot    (typically "$HOME/.knit/labels")
//   - ScopeProject: projectLabelsRoot (typically "<project>/.knit/labels")
//
// This knit root is kept separate from the roots of the AI tools themselves
// (~/.claude, ~/.codex, ~/.gemini for Claude, Codex, Gemini, etc.). That
// avoids mixing knit-managed metadata into the tool directories and lets users
// continue managing only the AI tool configuration files separately.
//
// ID encoding:
//   - '/' inside InstallationID is replaced with a single separator that is
//     safe for use as a base name. The exact replacement rule is internal to
//     this store, and List returns the restored InstallationID as-is.
//   - Base names that cannot be decoded (file names that violate the rule)
//     are skipped.
//
// Atomicity:
//   - Set uses a two-step "tmp file in the same directory -> rename" flow,
//     and non-existence of the destination path is guaranteed by an
//     O_EXCL-equivalent pre-check.
//   - Delete is a plain os.Remove, and a missing target maps to
//     ErrInstallationNotFound.
type SidecarLabelStore struct {
	target            source.Target
	userLabelsRoot    string
	projectLabelsRoot string
}

// sidecarSeparator is the substitution character used to flatten an
// InstallationID containing '/' into a single base name. The choice mirrors
// the existing distribution-level sidecar encoding: knowledge-format pack and
// entry names are kebab-case ([a-z0-9-]), so '_' never appears inside a valid
// InstallationID and the encoding round-trips unambiguously.
const sidecarSeparator = "_"

// sidecarExt is the on-disk extension for a label sidecar.
const sidecarExt = ".json"

// sidecarTmpPattern is the os.CreateTemp pattern used in the same directory as
// the destination so the subsequent rename stays atomic on POSIX.
const sidecarTmpPattern = ".knit-label-*.json.tmp"

// NewSidecarLabelStore constructs a SidecarLabelStore with fixed
// (target, userLabelsRoot, projectLabelsRoot).
//
// Contract:
//   - target is the source.Target handled by this store. Set/Get/Delete/List
//     determine the storage directory by combining the received scope with
//     this target.
//   - userLabelsRoot is the absolute path of the ScopeUser label-sidecar root
//     directory (for example "$HOME/.knit/labels"). If an empty string is
//     passed, every ScopeUser operation returns ErrLabelsRootNotConfigured.
//   - projectLabelsRoot is the absolute path of the ScopeProject label-sidecar
//     root directory (for example "<project>/.knit/labels"). If an empty
//     string is passed, every ScopeProject operation returns
//     ErrLabelsRootNotConfigured. This makes it explicit when the cli factory
//     could not discover the project root.
//
// Embedding the scope -> labelsRoot mapping inside the store lets callers
// (the cli factory) pass a single LabelStore per target into each distribution,
// instead of asking Installer/Uninstaller/Lister to branch between user and
// project stores.
//
// This constructor performs no I/O. Directories under each labelsRoot are
// created on demand by Set.
func NewSidecarLabelStore(target source.Target, userLabelsRoot, projectLabelsRoot string) *SidecarLabelStore {
	return &SidecarLabelStore{
		target:            target,
		userLabelsRoot:    userLabelsRoot,
		projectLabelsRoot: projectLabelsRoot,
	}
}

// labelsRootFor returns the absolute labels-root directory for scope.
//
// Error precedence:
//  1. ErrInvalidScope            (scope is neither ScopeUser nor ScopeProject)
//  2. ErrLabelsRootNotConfigured (the corresponding labels-root is empty,
//     for example because the cli factory could not discover the project root)
//
// This helper centralizes scope -> labelsRoot resolution so that
// Set/Get/Delete/List share a single validation point.
func (s *SidecarLabelStore) labelsRootFor(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		if s.userLabelsRoot == "" {
			return "", ErrLabelsRootNotConfigured
		}
		return s.userLabelsRoot, nil
	case ScopeProject:
		if s.projectLabelsRoot == "" {
			return "", ErrLabelsRootNotConfigured
		}
		return s.projectLabelsRoot, nil
	default:
		return "", ErrInvalidScope
	}
}

// scopeDir returns the absolute directory that stores sidecars for the given
// scope: <labelsRoot>/<target>/<scope>.
func (s *SidecarLabelStore) scopeDir(scope Scope) (string, error) {
	root, err := s.labelsRootFor(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, string(s.target), string(scope)), nil
}

// sidecarPath returns the absolute file path of a sidecar for (scope, id).
func (s *SidecarLabelStore) sidecarPath(scope Scope, id InstallationID) (string, error) {
	dir, err := s.scopeDir(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, encodeBaseName(id)+sidecarExt), nil
}

// labelFile is the on-disk JSON representation of LabelData. Field names are
// pinned here so renaming Go identifiers does not change the wire format.
type labelFile struct {
	SchemaVersion  int      `json:"schema_version"`
	ArtifactPath   string   `json:"artifact_path"`
	SourceEntryIDs []string `json:"source_entry_ids,omitempty"`
}

func (f labelFile) toData() LabelData {
	return LabelData{
		SchemaVersion:  f.SchemaVersion,
		ArtifactPath:   f.ArtifactPath,
		SourceEntryIDs: append([]string(nil), f.SourceEntryIDs...),
	}
}

func toLabelFile(data LabelData) labelFile {
	return labelFile{
		SchemaVersion:  data.SchemaVersion,
		ArtifactPath:   data.ArtifactPath,
		SourceEntryIDs: append([]string(nil), data.SourceEntryIDs...),
	}
}

// Set implements LabelStore.Set. See LabelStore.Set for the detailed contract.
func (s *SidecarLabelStore) Set(ctx context.Context, scope Scope, id InstallationID, data LabelData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	abs, err := s.sidecarPath(scope, id)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("inventory: create labels dir: %w", err)
	}
	if _, statErr := os.Stat(abs); statErr == nil {
		return ErrLabelAlreadyExists
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("inventory: stat label: %w", statErr)
	}
	tmp, err := os.CreateTemp(dir, sidecarTmpPattern)
	if err != nil {
		return fmt.Errorf("inventory: create tmp label: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(toLabelFile(data)); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("inventory: encode label: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("inventory: sync tmp label: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("inventory: close tmp label: %w", err)
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		cleanup()
		return fmt.Errorf("inventory: rename label: %w", err)
	}
	return nil
}

// Get implements LabelStore.Get. See LabelStore.Get for the detailed contract.
func (s *SidecarLabelStore) Get(ctx context.Context, scope Scope, id InstallationID) (LabelData, error) {
	if err := ctx.Err(); err != nil {
		return LabelData{}, err
	}
	abs, err := s.sidecarPath(scope, id)
	if err != nil {
		return LabelData{}, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return LabelData{}, fmt.Errorf("%w: %s", ErrInstallationNotFound, id)
		}
		return LabelData{}, fmt.Errorf("inventory: read label: %w", err)
	}
	var lf labelFile
	if err := json.Unmarshal(b, &lf); err != nil {
		return LabelData{}, fmt.Errorf("inventory: decode label: %w", err)
	}
	return lf.toData(), nil
}

// Delete implements LabelStore.Delete. See LabelStore.Delete for the detailed contract.
func (s *SidecarLabelStore) Delete(ctx context.Context, scope Scope, id InstallationID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	abs, err := s.sidecarPath(scope, id)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrInstallationNotFound, id)
		}
		return fmt.Errorf("inventory: remove label: %w", err)
	}
	return nil
}

// List implements LabelStore.List. See LabelStore.List for the detailed contract.
//
// Enumeration order is stabilized by sorting file names (that is, encoded IDs)
// in ascending order.
func (s *SidecarLabelStore) List(ctx context.Context, scope Scope) ([]LabelEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := s.scopeDir(scope)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inventory: read labels dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var out []LabelEntry
	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(name, sidecarExt) {
			continue
		}
		base := strings.TrimSuffix(name, sidecarExt)
		id, ok := decodeBaseName(base)
		if !ok {
			continue
		}
		abs := filepath.Join(dir, name)
		b, rErr := os.ReadFile(abs)
		if rErr != nil {
			continue
		}
		var lf labelFile
		if jErr := json.Unmarshal(b, &lf); jErr != nil {
			continue
		}
		out = append(out, LabelEntry{ID: id, Data: lf.toData()})
	}
	return out, nil
}

// encodeBaseName flattens an InstallationID into a base name by replacing
// '/' with sidecarSeparator. See type doc for the rationale.
func encodeBaseName(id InstallationID) string {
	return strings.ReplaceAll(string(id), "/", sidecarSeparator)
}

// decodeBaseName restores an InstallationID from a sidecar base name (without
// extension). Reports false when the base name is empty so the caller can skip
// it during List.
func decodeBaseName(base string) (InstallationID, bool) {
	if base == "" {
		return "", false
	}
	return InstallationID(strings.ReplaceAll(base, sidecarSeparator, "/")), true
}

// Static type check for early detection of signature drift.
var _ LabelStore = (*SidecarLabelStore)(nil)
