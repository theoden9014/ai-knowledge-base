package gemini

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

type agentRenderer struct{}

func (agentRenderer) Kind() source.Kind { return source.KindAgent }

func (agentRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
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
		Path:           "agents/" + e.Name + ".md",
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}
