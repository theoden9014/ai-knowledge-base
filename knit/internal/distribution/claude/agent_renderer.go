package claude

import "github.com/theoden9014/ai-knowledge-base/knit/internal/source"

// agentRenderer produces agents/<name>.md from a KindAgent entry.
type agentRenderer struct{}

// Kind returns source.KindAgent.
func (agentRenderer) Kind() source.Kind { return source.KindAgent }

// Render generates the agent artifact. uses_skills entries become a
// frontmatter `skills:` array using the "<pack>-<entry>" short form.
// tools.claude.frontmatter overrides any same-name keys produced from
// neutral metadata.
func (agentRenderer) Render(e *source.Entry, _ *source.Pack) ([]source.Artifact, error) {
	fm := map[string]any{
		"name":        e.Name,
		"description": e.Description,
	}
	if e.Agent != nil && len(e.Agent.UsesSkills) > 0 {
		skills := make([]string, 0, len(e.Agent.UsesSkills))
		for _, id := range e.Agent.UsesSkills {
			skills = append(skills, neutralIDToShortName(id))
		}
		fm["skills"] = skills
	}
	for k, v := range e.FrontmatterFor(Target) {
		fm[k] = v
	}
	content, err := frontmatterRenderer.Render(fm, e.Body)
	if err != nil {
		return nil, err
	}
	return []source.Artifact{{
		Target:         Target,
		Path:           "agents/" + e.Name + ".md",
		Content:        content,
		SourceEntryIDs: []string{e.ID},
	}}, nil
}
