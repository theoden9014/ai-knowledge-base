package remote

import (
	"fmt"
	"strings"
)

func (l *Locator) cloneURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", l.Host, l.Owner, l.Repo)
}

func parse(arg string) (*Locator, error) {
	if arg == "" {
		return nil, fmt.Errorf("%w: empty argument", ErrInvalidLocator)
	}
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return nil, fmt.Errorf("%w: URL scheme prefixes are not accepted (strip http(s):// before parsing)", ErrInvalidLocator)
	}
	if strings.ContainsAny(arg, "?#") {
		return nil, fmt.Errorf("%w: query strings and fragments are not accepted", ErrInvalidLocator)
	}
	if strings.Contains(arg, `\`) {
		return nil, fmt.Errorf("%w: backslashes are not accepted (use forward slashes)", ErrInvalidLocator)
	}
	if strings.Contains(arg, "//") {
		return nil, fmt.Errorf("%w: consecutive slashes are not accepted", ErrInvalidLocator)
	}
	if strings.HasPrefix(arg, "/") {
		return nil, fmt.Errorf("%w: leading slash is not accepted", ErrInvalidLocator)
	}
	if strings.HasSuffix(arg, "/") {
		return nil, fmt.Errorf("%w: trailing slash is not accepted", ErrInvalidLocator)
	}

	segments := strings.Split(arg, "/")
	if len(segments) < 3 {
		return nil, fmt.Errorf("%w: expected at least <host>/<owner>/<repo>, got %q", ErrInvalidLocator, arg)
	}

	host := strings.ToLower(segments[0])
	owner := segments[1]
	repo := strings.TrimSuffix(segments[2], ".git")
	if host == "" || owner == "" || repo == "" {
		return nil, fmt.Errorf("%w: host/owner/repo segments must be non-empty", ErrInvalidLocator)
	}

	loc := &Locator{
		Host:  host,
		Owner: owner,
		Repo:  repo,
	}
	if len(segments) > 3 {
		loc.Subpath = strings.Join(segments[3:], "/")
	}
	return loc, nil
}

func isRemoteArg(arg string) bool {
	if arg == "" {
		return false
	}
	// Absolute paths and "./" / "../" prefixes belong to the local
	// filesystem-path code path in the CLI layer; never remote.
	if strings.HasPrefix(arg, "/") {
		return false
	}
	if strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") {
		return false
	}
	if arg == "." || arg == ".." {
		return false
	}
	idx := strings.Index(arg, "/")
	if idx < 0 {
		// Bare single segment: a host name like "github.com" is not a
		// complete remote ref, so we leave it to fall through to the
		// pack-name path (where it will fail because "github.com" is not
		// a valid kebab-case pack name).
		return false
	}
	first := arg[:idx]
	return isHostLike(first)
}

// isHostLike returns true when s looks like a DNS host name: it contains
// at least one '.' with non-empty labels on both sides. Pure dot fragments
// ("." / "..") and labels that are entirely dots are rejected.
//
// IPv4 literals such as "1.2.3.4" satisfy this rule and therefore classify
// as host-like; that is intentional, since downstream Parse / Fetcher
// dispatch is responsible for deciding whether a given host is actually
// supported (the dispatch layer rejects unknown hosts with
// ErrUnsupportedHost). IPv6 literals (which contain ':' and brackets) are
// out of scope here.
func isHostLike(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return strings.Contains(s, ".")
}
