package source

import (
	"errors"
	"testing"
)

func newValidatorForTest(t *testing.T) Validator {
	t.Helper()
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	return v
}

func TestJsonSchemaValidator_ValidateManifest(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantKind error
	}{
		{
			name: "valid manifest",
			args: args{raw: []byte(`
pack: structure-behavior-design
version: 0.1.0
description: test pack
default_tools: [claude]
entries:
  - id: structure-behavior-design.skill.orchestrator
    path: skills/orchestrator
`)},
			wantErr: false,
		},
		{
			name: "missing required pack field",
			args: args{raw: []byte(`
version: 0.1.0
description: test pack
entries:
  - id: p.skill.a
    path: skills/a
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "pack name not kebab-case",
			args: args{raw: []byte(`
pack: StructureBehaviorDesign
version: 0.1.0
description: test pack
entries:
  - id: structure-behavior-design.skill.orchestrator
    path: skills/orchestrator
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "empty entries list",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries: []
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name:     "invalid yaml",
			args:     args{raw: []byte("pack: [unclosed")},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "additional property is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
unknown_field: surprise
entries:
  - id: p.skill.a
    path: skills/a
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "legacy skill path with SKILL.md is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a/SKILL.md
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "skill path with trailing slash is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: skills/a/
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "agent path as directory is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.agent.x
    path: agents/x
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "skill id with agent path shape is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.skill.a
    path: agents/a.md
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "legacy rule entry is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.rule.x
    path: rules/x.md
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "legacy prompt entry is rejected",
			args: args{raw: []byte(`
pack: p
version: 0.1.0
description: d
entries:
  - id: p.prompt.x
    path: prompts/x.md
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newValidatorForTest(t)
			err := v.ValidateManifest(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateManifest() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, tt.wantKind) {
				t.Errorf("ValidateManifest() err=%v, want errors.Is %v", err, tt.wantKind)
			}
		})
	}
}

func TestJsonSchemaValidator_ValidateEntryFrontmatter(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantKind error
	}{
		{
			name: "valid skill entry",
			args: args{raw: []byte(`
id: structure-behavior-design.skill.orchestrator
kind: skill
name: structure-behavior-design-orchestrator
description: orchestrator skill
`)},
			wantErr: false,
		},
		{
			name: "valid manual skill entry",
			args: args{raw: []byte(`
id: p.skill.cleanup
kind: skill
name: cleanup
description: cleanup development environment
invocation: manual
`)},
			wantErr: false,
		},
		{
			name: "valid both skill entry",
			args: args{raw: []byte(`
id: p.skill.cleanup
kind: skill
name: cleanup
description: cleanup development environment
invocation: both
`)},
			wantErr: false,
		},
		{
			name: "valid agent entry with uses_skills",
			args: args{raw: []byte(`
id: structure-behavior-design.agent.solid-reviewer
kind: agent
name: structure-behavior-design-solid-reviewer
description: solid reviewer
uses_skills:
  - structure-behavior-design.skill.orchestrator
`)},
			wantErr: false,
		},
		{
			name: "unknown kind reports ErrInvalidKind",
			args: args{raw: []byte(`
id: p.skill.a
kind: skil
name: p-a
description: typo in kind
`)},
			wantErr:  true,
			wantKind: ErrInvalidKind,
		},
		{
			name: "uses_skills on non-agent is rejected",
			args: args{raw: []byte(`
id: p.skill.a
kind: skill
name: p-a
description: misuse
uses_skills:
  - p.skill.x
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "invocation on agent is rejected",
			args: args{raw: []byte(`
id: p.agent.a
kind: agent
name: p-a
description: misuse
invocation: manual
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "unknown invocation is rejected",
			args: args{raw: []byte(`
id: p.skill.a
kind: skill
name: p-a
description: misuse
invocation: automatic
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "legacy rule kind reports ErrInvalidKind",
			args: args{raw: []byte(`
id: p.rule.a
kind: rule
name: p-a
description: legacy
`)},
			wantErr:  true,
			wantKind: ErrInvalidKind,
		},
		{
			name: "legacy prompt kind reports ErrInvalidKind",
			args: args{raw: []byte(`
id: p.prompt.a
kind: prompt
name: p-a
description: legacy
`)},
			wantErr:  true,
			wantKind: ErrInvalidKind,
		},
		{
			name: "name not kebab-case",
			args: args{raw: []byte(`
id: p.skill.a
kind: skill
name: PackA
description: bad name
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "id format mismatch",
			args: args{raw: []byte(`
id: pack/skill/a
kind: skill
name: p-a
description: bad id
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "missing required description",
			args: args{raw: []byte(`
id: p.skill.a
kind: skill
name: p-a
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
		{
			name: "additional property is rejected",
			args: args{raw: []byte(`
id: p.skill.a
kind: skill
name: p-a
description: d
unexpected: surprise
`)},
			wantErr:  true,
			wantKind: ErrSchemaViolation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newValidatorForTest(t)
			err := v.ValidateEntryFrontmatter(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEntryFrontmatter() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, tt.wantKind) {
				t.Errorf("ValidateEntryFrontmatter() err=%v, want errors.Is %v", err, tt.wantKind)
			}
		})
	}
}
