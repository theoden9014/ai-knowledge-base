package claude

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// skillRenderer produces skills/<name>/SKILL.md plus one artifact per
// sibling asset carried by the entry's SkillMeta.
type skillRenderer struct{}

// Kind returns source.KindSkill.
func (skillRenderer) Kind() source.Kind { return source.KindSkill }

// Render generates the SKILL.md artifact and any sibling-asset artifacts
// collected by the loader, merging tools.claude.frontmatter on top of the
// neutral name/description frontmatter for the body.
func (skillRenderer) Render(e *source.Entry, _ *source.Pack) ([]source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	for k, v := range e.FrontmatterFor(Target) {
		fm[k] = v
	}
	content, err := frontmatterRenderer.Render(fm, e.Body)
	if err != nil {
		return nil, err
	}
	root := "skills/" + e.Name
	arts := []source.Artifact{{
		Target:         Target,
		Path:           root + "/" + source.SkillBodyFileName,
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}}
	if e.Skill != nil {
		for _, a := range e.Skill.Assets() {
			arts = append(arts, source.Artifact{
				Target:         Target,
				Path:           root + "/" + a.Path(),
				Content:        a.Content(),
				SourceEntryIDs: []string{e.ID},
			})
		}
	}
	return arts, nil
}
