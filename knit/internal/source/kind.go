package source

// Kind classifies a knowledge entry into one of the four neutral categories
// defined by the knowledge-format specification.
type Kind string

// Kind values recognized by the knowledge format.
const (
	KindSkill  Kind = "skill"
	KindAgent  Kind = "agent"
	KindRule   Kind = "rule"
	KindPrompt Kind = "prompt"
)

// IsValid reports whether the receiver is one of the recognized Kind values.
// Validator implementations should wrap ErrInvalidKind when reporting a value
// that fails this check.
func (k Kind) IsValid() bool {
	switch k {
	case KindSkill, KindAgent, KindRule, KindPrompt:
		return true
	default:
		return false
	}
}

// String returns the string form of the Kind.
func (k Kind) String() string {
	return string(k)
}
