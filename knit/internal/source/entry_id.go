package source

import (
	"errors"
	"strings"
)

// ErrInvalidEntryID is returned when a string is rejected as an EntryID because
// it does not match <pack>.<kind>.<name> where pack and name are kebab-case and
// kind is one of the four valid Kinds.
var ErrInvalidEntryID = errors.New("invalid entry id")

// EntryID is the identity of an Entry in the form "<pack>.<kind>.<name>".
//
// The zero value represents "no id" (IsZero reports true). All non-zero values
// have passed structural validation in NewEntryID.
type EntryID struct {
	pack string
	kind Kind
	name string
}

// NewEntryID parses and validates an entry id string. The string must consist
// of exactly three dot-separated components: a kebab-case pack name, one of
// the four Kinds, and a kebab-case entry name.
func NewEntryID(s string) (EntryID, error) {
	pack, kindStr, name, ok := splitEntryID(s)
	if !ok {
		return EntryID{}, ErrInvalidEntryID
	}
	if !isKebabCase(pack) {
		return EntryID{}, ErrInvalidEntryID
	}
	kind := Kind(kindStr)
	if !kind.IsValid() {
		return EntryID{}, ErrInvalidEntryID
	}
	if !isKebabCase(name) {
		return EntryID{}, ErrInvalidEntryID
	}
	return EntryID{pack: pack, kind: kind, name: name}, nil
}

// Pack returns the pack component. Returns "" for the zero value.
func (id EntryID) Pack() string { return id.pack }

// Kind returns the kind component. Returns the zero Kind for the zero value.
func (id EntryID) Kind() Kind { return id.kind }

// Name returns the entry name component. Returns "" for the zero value.
func (id EntryID) Name() string { return id.name }

// String returns "<pack>.<kind>.<name>". Returns "" for the zero value.
func (id EntryID) String() string {
	if id.IsZero() {
		return ""
	}
	return id.pack + "." + id.kind.String() + "." + id.name
}

// IsZero reports whether id is the zero value.
func (id EntryID) IsZero() bool { return id.pack == "" && id.kind == "" && id.name == "" }

// splitEntryID partitions s into (pack, kind, name) at the FIRST and LAST dot.
// This handles cases where the pack name itself contains dots... but kebab-case
// forbids dots, so the only valid split is exactly two dots. We enforce that
// by counting.
func splitEntryID(s string) (pack, kind, name string, ok bool) {
	if s == "" {
		return "", "", "", false
	}
	first := strings.IndexByte(s, '.')
	if first < 0 {
		return "", "", "", false
	}
	last := strings.LastIndexByte(s, '.')
	if last == first {
		return "", "", "", false
	}
	pack = s[:first]
	kind = s[first+1 : last]
	name = s[last+1:]
	if pack == "" || kind == "" || name == "" {
		return "", "", "", false
	}
	return pack, kind, name, true
}

// isKebabCase reports whether s matches ^[a-z0-9]+(-[a-z0-9]+)*$.
func isKebabCase(s string) bool {
	if s == "" {
		return false
	}
	prevHyphen := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			prevHyphen = false
		case c == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}
	return !prevHyphen
}
