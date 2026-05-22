package gemini

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

type skillRenderer struct{}

func (skillRenderer) Kind() source.Kind { return source.KindSkill }

func (skillRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	for k, v := range e.FrontmatterFor(Target) {
		fm[k] = v
	}
	content, err := composeYAMLFrontmatter(fm, e.Body)
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
