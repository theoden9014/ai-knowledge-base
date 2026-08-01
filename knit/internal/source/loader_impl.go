package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"sigs.k8s.io/yaml"
)

// loader is the production implementation of Loader. It is composed with a
// Validator at construction time; LoadPack invokes the validator on every
// raw YAML payload it reads so that schema errors are reported with the
// source path attached.
type loader struct {
	validator Validator
}

// newLoader is the unexported constructor that builds a *loader bound to v.
// The exported entry point is NewLoader in loader.go, which simply delegates
// to this function; keeping the body here lets loader.go remain a thin
// API-only file while the implementation lives alongside loader itself.
func newLoader(v Validator) Loader {
	return &loader{validator: v}
}

// manifestRaw is the on-disk shape of manifest.yaml. It is intentionally
// minimal: schema validation happens on the raw bytes before this struct is
// populated.
type manifestRaw struct {
	Pack         string          `json:"pack"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	DefaultTools []string        `json:"default_tools,omitempty"`
	Entries      []manifestEntry `json:"entries"`
}

type manifestEntry struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// entryRaw is the on-disk shape of an entry frontmatter block.
type entryRaw struct {
	ID          string                   `json:"id"`
	Kind        string                   `json:"kind"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Tags        []string                 `json:"tags,omitempty"`
	Tools       map[string]toolConfigRaw `json:"tools,omitempty"`
	UsesSkills  []string                 `json:"uses_skills,omitempty"`
	Invocation  string                   `json:"invocation,omitempty"`
}

type toolConfigRaw struct {
	Enabled     *bool          `json:"enabled,omitempty"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
}

func (l *loader) LoadPack(ctx context.Context, fsys fs.FS, packDir string) (*Pack, LoadInfo, error) {
	info := LoadInfo{PackDir: packDir}

	if err := ctx.Err(); err != nil {
		return nil, info, err
	}

	manifestPath := path.Join(packDir, "manifest.yaml")
	manifestBytes, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, info, fmt.Errorf("%w: %s", ErrManifestNotFound, manifestPath)
		}
		return nil, info, fmt.Errorf("source: read %s: %w", manifestPath, err)
	}

	if err := l.validator.ValidateManifest(manifestBytes); err != nil {
		return nil, info, fmt.Errorf("source: %s: %w", manifestPath, err)
	}

	var mr manifestRaw
	if err := yaml.Unmarshal(manifestBytes, &mr); err != nil {
		return nil, info, fmt.Errorf("source: parse %s: %w", manifestPath, err)
	}
	packDirName := path.Base(path.Clean(packDir))
	// Remote fetchers may root fsys at the pack itself and pass "."; there
	// is no directory-name identity to compare in that representation.
	if packDirName != "." && packDirName != mr.Pack {
		return nil, info, fmt.Errorf(
			"%w: directory=%s manifest=%s",
			ErrPackMismatch, packDirName, mr.Pack,
		)
	}

	seen := make(map[string]struct{}, len(mr.Entries))
	entries := make([]Entry, 0, len(mr.Entries))
	for _, me := range mr.Entries {
		if err := ctx.Err(); err != nil {
			return nil, info, err
		}
		if _, dup := seen[me.ID]; dup {
			return nil, info, fmt.Errorf("%w: %s", ErrDuplicateEntryID, me.ID)
		}
		seen[me.ID] = struct{}{}
		entryID, err := NewEntryID(me.ID)
		if err != nil {
			return nil, info, fmt.Errorf("source: parse manifest entry id %q: %w", me.ID, err)
		}
		if entryID.Pack() != mr.Pack {
			return nil, info, fmt.Errorf(
				"%w: manifest=%s entry=%s",
				ErrPackMismatch, mr.Pack, entryID.Pack(),
			)
		}
		if err := validateManifestEntryPathIdentity(entryID, me.Path); err != nil {
			return nil, info, err
		}

		// skill entries point at a directory whose body is the fixed
		// SKILL.md file; agent entries point at a single markdown
		// file. We pick the resolver accordingly and, for skills, also
		// collect sibling assets that live alongside SKILL.md.
		isSkill := kindFromManifestEntryID(me.ID) == KindSkill
		var (
			bodyPath   string
			assets     []SkillAsset
			resolveErr error
		)
		if isSkill {
			bodyPath, assets, resolveErr = resolveSkillEntrySource(fsys, packDir, me)
		} else {
			bodyPath, resolveErr = resolveFileEntrySource(fsys, packDir, me)
		}
		if resolveErr != nil {
			return nil, info, resolveErr
		}
		raw, err := fs.ReadFile(fsys, bodyPath)
		if err != nil {
			return nil, info, fmt.Errorf("source: read %s: %w", bodyPath, err)
		}

		fmBytes, body, err := splitFrontmatter(raw)
		if err != nil {
			return nil, info, fmt.Errorf("source: %s: %w", bodyPath, err)
		}

		if err := l.validator.ValidateEntryFrontmatter(fmBytes); err != nil {
			return nil, info, fmt.Errorf("source: %s: %w", bodyPath, err)
		}

		var er entryRaw
		if err := yaml.Unmarshal(fmBytes, &er); err != nil {
			return nil, info, fmt.Errorf("source: parse %s: %w", bodyPath, err)
		}

		if er.ID != me.ID {
			return nil, info, fmt.Errorf(
				"%w: manifest=%s frontmatter=%s (at %s)",
				ErrIDMismatch, me.ID, er.ID, bodyPath,
			)
		}
		if Kind(er.Kind) != entryID.Kind() {
			return nil, info, fmt.Errorf(
				"%w: id=%s frontmatter=%s (at %s)",
				ErrKindMismatch, entryID.Kind(), er.Kind, bodyPath,
			)
		}

		entry := Entry{
			ID:          er.ID,
			Kind:        Kind(er.Kind),
			Name:        er.Name,
			Description: er.Description,
			Tags:        er.Tags,
			Path:        me.Path,
			Body:        body,
		}
		if len(er.Tools) > 0 {
			entry.Tools = make(map[Target]ToolConfig, len(er.Tools))
			for k, v := range er.Tools {
				entry.Tools[Target(k)] = ToolConfig(v)
			}
		}
		if Kind(er.Kind) == KindAgent && len(er.UsesSkills) > 0 {
			entry.Agent = &AgentMeta{UsesSkills: er.UsesSkills}
		}
		if Kind(er.Kind) == KindSkill {
			meta, mErr := NewSkillMetaWithInvocation(me.Path, assets, SkillInvocation(er.Invocation))
			if mErr != nil {
				return nil, info, fmt.Errorf("source: build skill meta %q: %w", me.Path, mErr)
			}
			entry.Skill = meta
		}
		entries = append(entries, entry)
	}

	defaults := make([]Target, 0, len(mr.DefaultTools))
	for _, t := range mr.DefaultTools {
		defaults = append(defaults, Target(t))
	}

	pack := &Pack{
		Name:         mr.Pack,
		Version:      mr.Version,
		Description:  mr.Description,
		DefaultTools: defaults,
		Entries:      entries,
	}
	return pack, info, nil
}

func validateManifestEntryPathIdentity(id EntryID, entryPath string) error {
	var pathName string
	switch id.Kind() {
	case KindSkill:
		pathName = path.Base(entryPath)
	case KindAgent:
		pathName = path.Base(entryPath)
		if !strings.HasSuffix(pathName, ".md") {
			return fmt.Errorf(
				"%w: id=%s path=%s",
				ErrPathMismatch, id.String(), entryPath,
			)
		}
		pathName = strings.TrimSuffix(pathName, ".md")
	default:
		return fmt.Errorf("%w: %s", ErrInvalidKind, id.Kind())
	}
	if pathName != id.Name() {
		return fmt.Errorf(
			"%w: id=%s path=%s",
			ErrPathMismatch, id.String(), entryPath,
		)
	}
	return nil
}

// splitFrontmatter separates the leading YAML frontmatter block (delimited
// by lines containing only "---") from the markdown body.
//
// The opening "---" must be the first non-empty line of the document. If no
// frontmatter delimiters are found, the whole document is treated as body
// and an empty frontmatter byte slice is returned, which the schema check
// will reject downstream.
//
// The returned body is byte-exact: every byte after the closing delimiter
// line (including trailing newlines and any inner whitespace) is preserved
// verbatim, so Builder implementations can rely on Entry.Body matching the
// source file from frontmatter end to EOF.
func splitFrontmatter(raw []byte) (frontmatter, body []byte, err error) {
	rest := raw
	// Allow an optional UTF-8 BOM.
	rest = bytes.TrimPrefix(rest, []byte{0xEF, 0xBB, 0xBF})

	// Find the opening delimiter (first line).
	line, after, ok := readLine(rest)
	if !ok || !isDelimiter(line) {
		return nil, raw, nil
	}
	fm := after
	for {
		line, next, ok := readLine(fm)
		if !ok {
			return nil, nil, fmt.Errorf("unterminated frontmatter block")
		}
		if isDelimiter(line) {
			fmEnd := len(after) - len(fm)
			return after[:fmEnd], next, nil
		}
		fm = next
	}
}

func readLine(b []byte) (line, rest []byte, ok bool) {
	if len(b) == 0 {
		return nil, nil, false
	}
	idx := bytes.IndexByte(b, '\n')
	if idx < 0 {
		return b, nil, true
	}
	return b[:idx], b[idx+1:], true
}

func isDelimiter(line []byte) bool {
	return bytes.Equal(bytes.TrimRight(line, "\r"), []byte("---"))
}
