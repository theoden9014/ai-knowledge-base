package source

import (
	"fmt"
	"slices"
	"strings"
)

// SkillMeta is the kind-specific sub-structure attached to an Entry whose
// Kind is KindSkill. It pairs the skill's root (a pack-relative directory)
// with the set of sibling assets the loader collected from that directory.
//
// Entry.Skill is non-nil exactly when Entry.Kind == KindSkill; consumers
// reading skill-specific data must go through SkillMeta rather than the
// legacy Entry.Path field, which carries the same root value only as a
// compatibility copy.
type SkillMeta struct {
	root   string
	assets []SkillAsset
}

// NewSkillMeta validates root and assets and returns a SkillMeta that owns
// a defensive copy of the assets slice. root must be a non-empty,
// non-absolute, forward-slash, pack-relative directory path without a
// trailing slash, backslash, or ".." segment. assets must not contain two
// elements with the same Path.
func NewSkillMeta(root string, assets []SkillAsset) (*SkillMeta, error) {
	if err := validateSkillRoot(root); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(assets))
	for _, a := range assets {
		if _, dup := seen[a.Path()]; dup {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateSkillAsset, a.Path())
		}
		seen[a.Path()] = struct{}{}
	}
	copied := make([]SkillAsset, len(assets))
	copy(copied, assets)
	return &SkillMeta{root: root, assets: copied}, nil
}

// Root returns the pack-relative skill root directory path (no trailing
// slash, forward-slash separated).
func (m *SkillMeta) Root() string { return m.root }

// Assets returns a defensive copy of the skill's sibling assets.
func (m *SkillMeta) Assets() []SkillAsset {
	out := make([]SkillAsset, len(m.assets))
	copy(out, m.assets)
	return out
}

func validateSkillRoot(root string) error {
	if root == "" {
		return fmt.Errorf("%w: empty root", ErrInvalidSkillRoot)
	}
	if strings.HasPrefix(root, "/") {
		return fmt.Errorf("%w: absolute path: %q", ErrInvalidSkillRoot, root)
	}
	if strings.HasSuffix(root, "/") {
		return fmt.Errorf("%w: trailing slash: %q", ErrInvalidSkillRoot, root)
	}
	if strings.ContainsRune(root, '\\') {
		return fmt.Errorf("%w: backslash not allowed: %q", ErrInvalidSkillRoot, root)
	}
	if slices.Contains(strings.Split(root, "/"), "..") {
		return fmt.Errorf("%w: parent traversal: %q", ErrInvalidSkillRoot, root)
	}
	return nil
}
