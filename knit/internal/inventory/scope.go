package inventory

// Scope represents the placement scope of an Installation.
// user means a user-wide area for the AI tool, and project means a local area
// for the current project.
type Scope string

const (
	// ScopeUser represents the user-wide scope.
	ScopeUser Scope = "user"
	// ScopeProject represents the project-local scope.
	ScopeProject Scope = "project"
)

// Valid reports whether Scope is one of the allowed constants
// (ScopeUser / ScopeProject).
func (s Scope) Valid() bool {
	switch s {
	case ScopeUser, ScopeProject:
		return true
	default:
		return false
	}
}

// Validate checks whether Scope is allowed and returns ErrInvalidScope when it
// is not. Callers that want to branch on an error should use this method.
func (s Scope) Validate() error {
	if !s.Valid() {
		return ErrInvalidScope
	}
	return nil
}

// String returns the string representation of Scope.
func (s Scope) String() string {
	return string(s)
}
