package gemini

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

type skillRenderer struct{}

func (skillRenderer) Kind() source.Kind { return source.KindSkill }

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
