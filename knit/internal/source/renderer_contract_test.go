package source

import (
	"context"
	"errors"
	"testing"
)

type stubRenderer struct {
	kind Kind
}

func (s stubRenderer) Kind() Kind { return s.kind }
func (s stubRenderer) Render(_ *Entry, _ *Pack) ([]Artifact, error) {
	return nil, nil
}

func TestRendererRegistry_emptyRenderResultIsContractViolation(t *testing.T) {
	t.Parallel()
	r := NewRendererRegistry(Target("any"))
	r.Register(stubRenderer{kind: KindAgent})
	pack := &Pack{
		Name:         "p",
		DefaultTools: []Target{Target("any")},
		Entries: []Entry{
			{
				ID:   "p.agent.x",
				Kind: KindAgent,
				Name: "p-x",
				Path: "agents/x.md",
			},
		},
	}
	_, err := r.Build(context.Background(), pack)
	if err == nil {
		t.Fatal("Build returned nil error for empty renderer result")
	}
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("err = %v, want errors.Is ErrUnsupportedKind", err)
	}
}
