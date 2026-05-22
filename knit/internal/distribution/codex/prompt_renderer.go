package codex

import (
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type promptRenderer struct{}

func (promptRenderer) Kind() source.Kind { return source.KindPrompt }

func (promptRenderer) Render(e *source.Entry, _ *source.Pack) (source.Artifact, error) {
	fm := map[string]any{
		"description": e.Description,
	}
	for k, v := range e.FrontmatterFor(Target) {
		fm[k] = v
	}
	content, err := writeMarkdownWithFrontmatter(fm, e.Body)
	if err != nil {
		return source.Artifact{}, err
	}
	return source.Artifact{
		Target:         Target,
		Path:           fmt.Sprintf("prompts/%s.md", e.Name),
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}, nil
}
