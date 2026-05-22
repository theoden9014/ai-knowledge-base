package claude

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// skillRenderer produces skills/<name>/SKILL.md from a KindSkill entry.
type skillRenderer struct{}

// Kind returns source.KindSkill.
func (skillRenderer) Kind() source.Kind { return source.KindSkill }

// Render generates the SKILL.md artifact, merging tools.claude.frontmatter
// on top of the neutral name/description frontmatter.
func (skillRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	mergeClaudeFrontmatter(fm, e)
	content, err := renderWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           "skills/" + e.Name + "/SKILL.md",
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}
