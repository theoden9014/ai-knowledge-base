package codex

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

type agentRenderer struct{}

func (agentRenderer) Kind() source.Kind { return source.KindAgent }

func (agentRenderer) Render(e *source.Entry, _ *source.Pack) ([]source.Artifact, error) {
	table := map[string]any{
		"name":                   e.Name,
		"description":            e.Description,
		"developer_instructions": string(e.Body),
	}
	for k, v := range e.FrontmatterFor(Target) {
		table[k] = v
	}
	buf, err := toml.Marshal(table)
	if err != nil {
		return nil, fmt.Errorf("codex: marshal agent toml for %s: %w", e.ID, err)
	}
	return []source.Artifact{{
		Target:         Target,
		Path:           fmt.Sprintf("agents/%s.toml", e.Name),
		Content:        buf,
		SourceEntryIDs: []string{e.ID},
	}}, nil
}
