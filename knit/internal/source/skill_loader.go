package source

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// kindFromManifestEntryID extracts the kind segment from a manifest entry
// id of the form "<pack>.<kind>.<name>". The schema validator rejects
// malformed ids before this point, so an out-of-shape id here yields an
// empty Kind that no kind branch will match.
func kindFromManifestEntryID(id string) Kind {
	parts := strings.Split(id, ".")
	if len(parts) < 3 {
		return ""
	}
	return Kind(parts[1])
}

// resolveFileEntrySource validates that the manifest path of a non-skill
// entry points at an existing file and returns its pack-rooted path.
func resolveFileEntrySource(fsys fs.FS, packDir string, me manifestEntry) (string, error) {
	bodyPath := path.Join(packDir, me.Path)
	if _, statErr := fs.Stat(fsys, bodyPath); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrEntryNotFound, bodyPath)
		}
		return "", fmt.Errorf("source: stat %s: %w", bodyPath, statErr)
	}
	return bodyPath, nil
}

// resolveSkillEntrySource validates the skill root directory, locates the
// SKILL.md body file, and collects every sibling asset. Returns the body
// path and the asset slice.
//
// Errors map to dedicated sentinels in the order documented in
// docs/skill-directory-interface-design.md:
//   - ErrSkillPathNotFound: skill root is absent
//   - ErrSkillPathNotDirectory: skill root exists but is a file
//   - ErrSkillBodyNotFound: skill root has no SKILL.md
//
// All three sentinels also satisfy errors.Is(..., ErrSkillResolution).
func resolveSkillEntrySource(fsys fs.FS, packDir string, me manifestEntry) (string, []SkillAsset, error) {
	rootPath := path.Join(packDir, me.Path)
	info, statErr := fs.Stat(fsys, rootPath)
	switch {
	case statErr == nil && info.IsDir():
		// fall through to body + assets collection
	case statErr == nil:
		return "", nil, fmt.Errorf("%w: %s", ErrSkillPathNotDirectory, rootPath)
	case errors.Is(statErr, fs.ErrNotExist):
		return "", nil, fmt.Errorf("%w: %s", ErrSkillPathNotFound, rootPath)
	default:
		return "", nil, fmt.Errorf("source: stat %s: %w", rootPath, statErr)
	}

	bodyPath := path.Join(rootPath, SkillBodyFileName)
	if _, bErr := fs.Stat(fsys, bodyPath); bErr != nil {
		if errors.Is(bErr, fs.ErrNotExist) {
			return "", nil, fmt.Errorf("%w: %s", ErrSkillBodyNotFound, bodyPath)
		}
		return "", nil, fmt.Errorf("source: stat %s: %w", bodyPath, bErr)
	}

	collected, cErr := collectSkillAssets(fsys, rootPath)
	if cErr != nil {
		return "", nil, fmt.Errorf("source: collect skill assets %q: %w", rootPath, cErr)
	}
	return bodyPath, collected, nil
}

// collectSkillAssets walks the skill root and returns one SkillAsset per
// regular file found beneath it, excluding the SKILL.md body file.
// Directories, symbolic links, and other non-regular files (sockets,
// devices, ...) are silently skipped because the requirements scope
// (skill-directory-requirements.md EE3) excludes them from siblings.
// ReadFile failures are wrapped with the offending pack-rooted path so
// callers can identify the file even after the loader rewraps the error.
func collectSkillAssets(fsys fs.FS, root string) ([]SkillAsset, error) {
	var out []SkillAsset
	walkErr := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == root || d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel := strings.TrimPrefix(p, root+"/")
		if rel == p {
			return fmt.Errorf("source: walk produced %q outside %q", p, root)
		}
		if rel == SkillBodyFileName {
			return nil
		}
		body, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return fmt.Errorf("source: read skill asset %s: %w", p, readErr)
		}
		asset, aErr := NewSkillAsset(rel, body)
		if aErr != nil {
			return fmt.Errorf("source: build skill asset %s: %w", p, aErr)
		}
		out = append(out, asset)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}
