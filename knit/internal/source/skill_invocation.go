package source

// SkillInvocation controls whether a skill may be selected implicitly by a
// target harness or must be invoked explicitly by the user.
type SkillInvocation string

const (
	// SkillInvocationBoth adds no neutral restriction on explicit or
	// implicit invocation. Target-specific metadata may narrow it.
	SkillInvocationBoth SkillInvocation = "both"

	// SkillInvocationManual permits explicit invocation only.
	SkillInvocationManual SkillInvocation = "manual"
)

// NormalizeSkillInvocation applies the neutral-format default and validates
// the result. An omitted value defaults to SkillInvocationBoth.
func NormalizeSkillInvocation(v SkillInvocation) (SkillInvocation, error) {
	if v == "" {
		return SkillInvocationBoth, nil
	}
	switch v {
	case SkillInvocationBoth, SkillInvocationManual:
		return v, nil
	default:
		return "", ErrInvalidSkillInvocation
	}
}
