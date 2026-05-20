package inventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

const testTarget source.Target = "claude"

func TestNewSidecarLabelStore(t *testing.T) {
	type args struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	tests := []struct {
		name string
		args args
		want *SidecarLabelStore
	}{
		{
			name: "all fields populated",
			args: args{
				target:            testTarget,
				userLabelsRoot:    "/home/u/.knit/labels",
				projectLabelsRoot: "/proj/.knit/labels",
			},
			want: &SidecarLabelStore{
				target:            testTarget,
				userLabelsRoot:    "/home/u/.knit/labels",
				projectLabelsRoot: "/proj/.knit/labels",
			},
		},
		{
			name: "project labels root empty",
			args: args{
				target:            testTarget,
				userLabelsRoot:    "/home/u/.knit/labels",
				projectLabelsRoot: "",
			},
			want: &SidecarLabelStore{
				target:            testTarget,
				userLabelsRoot:    "/home/u/.knit/labels",
				projectLabelsRoot: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSidecarLabelStore(tt.args.target, tt.args.userLabelsRoot, tt.args.projectLabelsRoot)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(SidecarLabelStore{})); diff != "" {
				t.Errorf("NewSidecarLabelStore() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSidecarLabelStore_labelsRootFor(t *testing.T) {
	type fields struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	type args struct {
		scope Scope
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "user scope returns userLabelsRoot",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "/u",
				projectLabelsRoot: "/p",
			},
			args:    args{scope: ScopeUser},
			want:    "/u",
			wantErr: false,
		},
		{
			name: "project scope returns projectLabelsRoot",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "/u",
				projectLabelsRoot: "/p",
			},
			args:    args{scope: ScopeProject},
			want:    "/p",
			wantErr: false,
		},
		{
			name: "user scope with empty userLabelsRoot -> not configured",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "",
				projectLabelsRoot: "/p",
			},
			args:    args{scope: ScopeUser},
			want:    "",
			wantErr: true,
		},
		{
			name: "project scope with empty projectLabelsRoot -> not configured",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "/u",
				projectLabelsRoot: "",
			},
			args:    args{scope: ScopeProject},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid scope -> ErrInvalidScope",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "/u",
				projectLabelsRoot: "/p",
			},
			args:    args{scope: Scope("system")},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarLabelStore{
				target:            tt.fields.target,
				userLabelsRoot:    tt.fields.userLabelsRoot,
				projectLabelsRoot: tt.fields.projectLabelsRoot,
			}
			got, err := s.labelsRootFor(tt.args.scope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SidecarLabelStore.labelsRootFor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("SidecarLabelStore.labelsRootFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSidecarLabelStore_labelsRootFor_ErrorIdentity(t *testing.T) {
	tests := []struct {
		name              string
		userLabelsRoot    string
		projectLabelsRoot string
		scope             Scope
		want              error
	}{
		{
			name:              "user scope empty user root -> ErrLabelsRootNotConfigured",
			userLabelsRoot:    "",
			projectLabelsRoot: "/p",
			scope:             ScopeUser,
			want:              ErrLabelsRootNotConfigured,
		},
		{
			name:              "project scope empty project root -> ErrLabelsRootNotConfigured",
			userLabelsRoot:    "/u",
			projectLabelsRoot: "",
			scope:             ScopeProject,
			want:              ErrLabelsRootNotConfigured,
		},
		{
			name:              "unknown scope -> ErrInvalidScope",
			userLabelsRoot:    "/u",
			projectLabelsRoot: "/p",
			scope:             Scope(""),
			want:              ErrInvalidScope,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSidecarLabelStore(testTarget, tt.userLabelsRoot, tt.projectLabelsRoot)
			_, err := s.labelsRootFor(tt.scope)
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want errors.Is = %v", err, tt.want)
			}
		})
	}
}

func TestSidecarLabelStore_Set(t *testing.T) {
	type fields struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	type args struct {
		ctx   context.Context
		scope Scope
		id    InstallationID
		data  LabelData
	}
	tmpUser := t.TempDir()
	tmpProject := t.TempDir()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful set under user scope",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    filepath.Join(tmpUser, "set-user-ok"),
				projectLabelsRoot: tmpProject,
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    InstallationID("skills/foo/SKILL.md"),
				data: LabelData{
					SchemaVersion:  1,
					ArtifactPath:   "skills/foo/SKILL.md",
					SourceEntryIDs: []string{"core.skill.foo"},
				},
			},
			wantErr: false,
		},
		{
			name: "successful set under project scope",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: filepath.Join(tmpProject, "set-project-ok"),
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeProject,
				id:    InstallationID("AGENTS.md"),
				data: LabelData{
					SchemaVersion:  1,
					ArtifactPath:   "AGENTS.md",
					SourceEntryIDs: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid scope -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: tmpProject,
			},
			args: args{
				ctx:   context.Background(),
				scope: Scope("bogus"),
				id:    InstallationID("x"),
				data:  LabelData{SchemaVersion: 1, ArtifactPath: "x"},
			},
			wantErr: true,
		},
		{
			name: "user labels root not configured -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "",
				projectLabelsRoot: tmpProject,
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    InstallationID("x"),
				data:  LabelData{SchemaVersion: 1, ArtifactPath: "x"},
			},
			wantErr: true,
		},
		{
			name: "canceled context -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    filepath.Join(tmpUser, "set-ctx-cancel"),
				projectLabelsRoot: tmpProject,
			},
			args: args{
				ctx:   canceledCtx(),
				scope: ScopeUser,
				id:    InstallationID("x"),
				data:  LabelData{SchemaVersion: 1, ArtifactPath: "x"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarLabelStore{
				target:            tt.fields.target,
				userLabelsRoot:    tt.fields.userLabelsRoot,
				projectLabelsRoot: tt.fields.projectLabelsRoot,
			}
			if err := s.Set(tt.args.ctx, tt.args.scope, tt.args.id, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("SidecarLabelStore.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSidecarLabelStore_Set_Duplicate verifies the ErrLabelAlreadyExists
// contract that is invariant across Set call sequences.
func TestSidecarLabelStore_Set_Duplicate(t *testing.T) {
	tmpUser := t.TempDir()
	s := NewSidecarLabelStore(testTarget, tmpUser, "")
	id := InstallationID("skills/foo/SKILL.md")
	data := LabelData{SchemaVersion: 1, ArtifactPath: "skills/foo/SKILL.md"}
	if err := s.Set(context.Background(), ScopeUser, id, data); err != nil {
		t.Fatalf("first Set: unexpected error: %v", err)
	}
	err := s.Set(context.Background(), ScopeUser, id, data)
	if !errors.Is(err, ErrLabelAlreadyExists) {
		t.Errorf("second Set: error = %v, want errors.Is = %v", err, ErrLabelAlreadyExists)
	}
}

func TestSidecarLabelStore_Get(t *testing.T) {
	type fields struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	type args struct {
		ctx   context.Context
		scope Scope
		id    InstallationID
	}
	tmpUser := t.TempDir()
	// Seed: set a label under user scope so Get can read it.
	seed := NewSidecarLabelStore(testTarget, tmpUser, "")
	seedID := InstallationID("skills/foo/SKILL.md")
	seedData := LabelData{
		SchemaVersion:  1,
		ArtifactPath:   "skills/foo/SKILL.md",
		SourceEntryIDs: []string{"core.skill.foo"},
	}
	if err := seed.Set(context.Background(), ScopeUser, seedID, seedData); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    LabelData
		wantErr bool
	}{
		{
			name: "existing label is returned",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    seedID,
			},
			want:    seedData,
			wantErr: false,
		},
		{
			name: "absent label -> ErrInstallationNotFound",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    InstallationID("skills/nope/SKILL.md"),
			},
			want:    LabelData{},
			wantErr: true,
		},
		{
			name: "invalid scope -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: Scope("system"),
				id:    seedID,
			},
			want:    LabelData{},
			wantErr: true,
		},
		{
			name: "labels root not configured -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "",
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    seedID,
			},
			want:    LabelData{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarLabelStore{
				target:            tt.fields.target,
				userLabelsRoot:    tt.fields.userLabelsRoot,
				projectLabelsRoot: tt.fields.projectLabelsRoot,
			}
			got, err := s.Get(tt.args.ctx, tt.args.scope, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SidecarLabelStore.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SidecarLabelStore.Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestSidecarLabelStore_Get_NotFoundIdentity verifies that the absence case
// maps to ErrInstallationNotFound via errors.Is.
func TestSidecarLabelStore_Get_NotFoundIdentity(t *testing.T) {
	tmpUser := t.TempDir()
	s := NewSidecarLabelStore(testTarget, tmpUser, "")
	_, err := s.Get(context.Background(), ScopeUser, InstallationID("missing"))
	if !errors.Is(err, ErrInstallationNotFound) {
		t.Errorf("Get(absent) error = %v, want errors.Is = %v", err, ErrInstallationNotFound)
	}
}

func TestSidecarLabelStore_Delete(t *testing.T) {
	type fields struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	type args struct {
		ctx   context.Context
		scope Scope
		id    InstallationID
	}
	tmpUser := t.TempDir()
	// Seed
	seed := NewSidecarLabelStore(testTarget, tmpUser, "")
	seedID := InstallationID("skills/foo/SKILL.md")
	if err := seed.Set(context.Background(), ScopeUser, seedID, LabelData{SchemaVersion: 1, ArtifactPath: "skills/foo/SKILL.md"}); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "delete existing label",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    seedID,
			},
			wantErr: false,
		},
		{
			name: "delete absent label -> ErrInstallationNotFound",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    InstallationID("skills/nope/SKILL.md"),
			},
			wantErr: true,
		},
		{
			name: "invalid scope -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpUser,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: Scope("nope"),
				id:    seedID,
			},
			wantErr: true,
		},
		{
			name: "labels root not configured -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "",
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
				id:    seedID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarLabelStore{
				target:            tt.fields.target,
				userLabelsRoot:    tt.fields.userLabelsRoot,
				projectLabelsRoot: tt.fields.projectLabelsRoot,
			}
			if err := s.Delete(tt.args.ctx, tt.args.scope, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("SidecarLabelStore.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSidecarLabelStore_Delete_RemovesFile checks the side effect that Delete
// physically removes the underlying sidecar so a subsequent Get returns
// ErrInstallationNotFound.
func TestSidecarLabelStore_Delete_RemovesFile(t *testing.T) {
	tmpUser := t.TempDir()
	s := NewSidecarLabelStore(testTarget, tmpUser, "")
	id := InstallationID("skills/foo/SKILL.md")
	if err := s.Set(context.Background(), ScopeUser, id, LabelData{SchemaVersion: 1, ArtifactPath: "skills/foo/SKILL.md"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.Delete(context.Background(), ScopeUser, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(context.Background(), ScopeUser, id); !errors.Is(err, ErrInstallationNotFound) {
		t.Errorf("Get after Delete: err=%v, want errors.Is=ErrInstallationNotFound", err)
	}
}

func TestSidecarLabelStore_List(t *testing.T) {
	type fields struct {
		target            source.Target
		userLabelsRoot    string
		projectLabelsRoot string
	}
	type args struct {
		ctx   context.Context
		scope Scope
	}

	// Seeded environment: two user-scope labels.
	tmpSeeded := t.TempDir()
	seed := NewSidecarLabelStore(testTarget, tmpSeeded, "")
	idA := InstallationID("skills/aaa/SKILL.md")
	dataA := LabelData{
		SchemaVersion:  1,
		ArtifactPath:   "skills/aaa/SKILL.md",
		SourceEntryIDs: []string{"core.skill.aaa"},
	}
	idB := InstallationID("skills/bbb/SKILL.md")
	dataB := LabelData{
		SchemaVersion:  1,
		ArtifactPath:   "skills/bbb/SKILL.md",
		SourceEntryIDs: []string{"core.skill.bbb"},
	}
	if err := seed.Set(context.Background(), ScopeUser, idA, dataA); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if err := seed.Set(context.Background(), ScopeUser, idB, dataB); err != nil {
		t.Fatalf("seed B: %v", err)
	}
	// Drop an unrelated file beside the labels to verify base-name decoding skips it.
	junkDir := filepath.Join(tmpSeeded, string(testTarget), string(ScopeUser))
	if err := os.WriteFile(filepath.Join(junkDir, "not-a-label.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("seed junk: %v", err)
	}

	tmpEmpty := t.TempDir()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []LabelEntry
		wantErr bool
	}{
		{
			name: "lists user-scope labels sorted by encoded base name",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpSeeded,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
			},
			want: []LabelEntry{
				{ID: idA, Data: dataA},
				{ID: idB, Data: dataB},
			},
			wantErr: false,
		},
		{
			name: "empty directory -> nil result without error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpEmpty,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid scope -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    tmpSeeded,
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: Scope("x"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "labels root not configured -> error",
			fields: fields{
				target:            testTarget,
				userLabelsRoot:    "",
				projectLabelsRoot: "",
			},
			args: args{
				ctx:   context.Background(),
				scope: ScopeUser,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarLabelStore{
				target:            tt.fields.target,
				userLabelsRoot:    tt.fields.userLabelsRoot,
				projectLabelsRoot: tt.fields.projectLabelsRoot,
			}
			got, err := s.List(tt.args.ctx, tt.args.scope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SidecarLabelStore.List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SidecarLabelStore.List() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
