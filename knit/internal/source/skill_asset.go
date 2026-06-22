package source

import (
	"fmt"
	"slices"
	"strings"
)

// SkillAsset is one sibling file carried alongside a skill's SKILL.md body.
// Its path is relative to the skill root directory; SKILL.md itself is not
// representable as a SkillAsset.
type SkillAsset struct {
	path    string
	content []byte
}

// NewSkillAsset validates relPath and returns a SkillAsset that owns a
// defensive copy of content. relPath must be a non-empty, non-absolute,
// forward-slash relative path that does not traverse with "..", does not
// contain backslashes, and is not exactly "SKILL.md".
func NewSkillAsset(relPath string, content []byte) (SkillAsset, error) {
	if relPath == "" {
		return SkillAsset{}, fmt.Errorf("%w: empty path", ErrInvalidSkillAssetPath)
	}
	if relPath == SkillBodyFileName {
		return SkillAsset{}, fmt.Errorf("%w: %q is the skill body, not an asset", ErrInvalidSkillAssetPath, relPath)
	}
	if strings.ContainsRune(relPath, '\\') {
		return SkillAsset{}, fmt.Errorf("%w: backslash not allowed: %q", ErrInvalidSkillAssetPath, relPath)
	}
	if strings.HasPrefix(relPath, "/") {
		return SkillAsset{}, fmt.Errorf("%w: absolute path: %q", ErrInvalidSkillAssetPath, relPath)
	}
	if slices.Contains(strings.Split(relPath, "/"), "..") {
		return SkillAsset{}, fmt.Errorf("%w: parent traversal: %q", ErrInvalidSkillAssetPath, relPath)
	}
	buf := make([]byte, len(content))
	copy(buf, content)
	return SkillAsset{path: relPath, content: buf}, nil
}

// Path returns the skill-root-relative path of the asset.
func (a SkillAsset) Path() string { return a.path }

// Content returns a defensive copy of the asset bytes.
func (a SkillAsset) Content() []byte {
	out := make([]byte, len(a.content))
	copy(out, a.content)
	return out
}

// IsZero reports whether the receiver is the zero value.
func (a SkillAsset) IsZero() bool {
	return a.path == "" && a.content == nil
}

// SkillBodyFileName is the fixed file name that carries a skill's body
// (frontmatter + markdown) under the skill root directory. Loader uses it
// to locate the body file; distribution renderers use it to compose the
// target-side artifact path so the two stay in sync.
const SkillBodyFileName = "SKILL.md"
