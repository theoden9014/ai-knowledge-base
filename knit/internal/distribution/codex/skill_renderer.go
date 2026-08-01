package codex

import (
	"fmt"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"sigs.k8s.io/yaml"
)

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
	var openAIMetadata []byte
	if e.Skill != nil {
		for _, a := range e.Skill.Assets() {
			if a.Path() == "agents/openai.yaml" && e.Skill.Invocation() == source.SkillInvocationManual {
				openAIMetadata = a.Content()
				continue
			}
			arts = append(arts, source.Artifact{
				Target:         Target,
				Path:           root + "/" + a.Path(),
				Content:        a.Content(),
				SourceEntryIDs: []string{e.ID},
			})
		}
		if e.Skill.Invocation() == source.SkillInvocationManual {
			metadata, err := renderManualInvocationMetadata(openAIMetadata)
			if err != nil {
				return nil, err
			}
			arts = append(arts, source.Artifact{
				Target:         Target,
				Path:           root + "/agents/openai.yaml",
				Content:        metadata,
				SourceEntryIDs: []string{e.ID},
			})
		}
	}
	return arts, nil
}

func renderManualInvocationMetadata(existing []byte) ([]byte, error) {
	metadata := map[string]any{}
	if len(existing) > 0 {
		var decoded any
		if err := yaml.Unmarshal(existing, &decoded); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSkillMetadata, err)
		}
		var ok bool
		metadata, ok = decoded.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: document must be a mapping", ErrInvalidSkillMetadata)
		}
	}
	policy, ok := metadata["policy"].(map[string]any)
	if !ok {
		if metadata["policy"] != nil {
			return nil, fmt.Errorf("%w: policy must be a mapping", ErrInvalidSkillMetadata)
		}
		policy = map[string]any{}
		metadata["policy"] = policy
	}
	policy["allow_implicit_invocation"] = false

	content, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSkillMetadata, err)
	}
	return content, nil
}
