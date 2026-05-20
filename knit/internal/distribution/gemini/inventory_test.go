package gemini

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory/inventorytest"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

// newTestRoots creates and returns independent tempdirs for user / project
// inventory roots plus a LabelStore wired against a separate knit metadata
// root. The layout mirrors the cli factory.
func newTestRoots(t *testing.T) (userRoot, projectRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user", ".gemini")
	projectRoot = filepath.Join(base, "project", ".gemini")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	projectLabelsRoot := filepath.Join(base, "project", ".knit", "labels")
	if err := os.MkdirAll(userRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(userRoot): %v", err)
	}
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(projectRoot): %v", err)
	}
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, projectLabelsRoot)
	return userRoot, projectRoot, labels
}

// newTestRootsUserOnly returns a fixture where projectRoot is intentionally
// empty so callers can exercise ErrProjectRootNotConfigured paths.
func newTestRootsUserOnly(t *testing.T) (userRoot string, labels *inventory.SidecarLabelStore) {
	t.Helper()
	base := t.TempDir()
	userRoot = filepath.Join(base, "user", ".gemini")
	userLabelsRoot := filepath.Join(base, "user", ".knit", "labels")
	if err := os.MkdirAll(userRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(userRoot): %v", err)
	}
	labels = inventory.NewSidecarLabelStore(Target, userLabelsRoot, "")
	return userRoot, labels
}

func sampleSkillArtifact() source.Artifact {
	return source.Artifact{
		Target:         Target,
		Path:           "skills/sample/SKILL.md",
		Content:        []byte("---\nname: sample\ndescription: desc\n---\nbody\n"),
		SourceEntryIDs: []string{"p.skill.sample"},
	}
}

func TestInstaller_Contract(t *testing.T) {
	inventorytest.RunInstallerContract(t, func(t *testing.T) inventorytest.InstallerHarness {
		userRoot, projectRoot, labels := newTestRoots(t)
		ins := NewInstaller(userRoot, projectRoot, labels)
		un := NewUninstaller(userRoot, projectRoot, labels)
		return inventorytest.InstallerHarness{
			Installer:       ins,
			SupportedTarget: Target,
			SampleArtifact:  sampleSkillArtifact(),
			Uninstaller:     un,
		}
	})
}

func TestUninstaller_Contract(t *testing.T) {
	inventorytest.RunUninstallerContract(t, func(t *testing.T) inventorytest.UninstallerHarness {
		userRoot, projectRoot, labels := newTestRoots(t)
		ins := NewInstaller(userRoot, projectRoot, labels)
		un := NewUninstaller(userRoot, projectRoot, labels)
		return inventorytest.UninstallerHarness{
			Uninstaller:     un,
			SupportedTarget: Target,
			SeedInstalled: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				art := sampleSkillArtifact()
				inst, err := ins.Install(context.Background(), scope, art)
				if err != nil {
					t.Fatalf("seed Install() error = %v", err)
				}
				return inst
			},
			FabricateMissing: func(t *testing.T, scope inventory.Scope) inventory.Installation {
				return inventory.Installation{
					ID: inventory.InstallationID("skills/missing/SKILL.md"),
					Label: inventory.Label{
						Target: Target,
						Scope:  scope,
					},
				}
			},
		}
	})
}

func TestLister_Contract(t *testing.T) {
	inventorytest.RunListerContract(t, func(t *testing.T) inventorytest.ListerHarness {
		userRoot, projectRoot, labels := newTestRoots(t)
		ins := NewInstaller(userRoot, projectRoot, labels)
		ls := NewLister(userRoot, projectRoot, labels)
		return inventorytest.ListerHarness{
			Lister:          ls,
			SupportedTarget: Target,
			SeedInstalled: func(t *testing.T, scope inventory.Scope, n int) []inventory.Installation {
				out := make([]inventory.Installation, 0, n)
				for i := 0; i < n; i++ {
					art := source.Artifact{
						Target:         Target,
						Path:           fmt.Sprintf("skills/sk-%d/SKILL.md", i),
						Content:        []byte(fmt.Sprintf("body-%d\n", i)),
						SourceEntryIDs: []string{fmt.Sprintf("p.skill.sk-%d", i)},
					}
					inst, err := ins.Install(context.Background(), scope, art)
					if err != nil {
						t.Fatalf("seed Install() error = %v", err)
					}
					out = append(out, inst)
				}
				return out
			},
		}
	})
}

func TestInstaller_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	if got, want := ins.Target(), Target; got != want {
		t.Errorf("Installer.Target() = %q, want %q", got, want)
	}
}

func TestUninstaller_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	un := NewUninstaller(userRoot, projectRoot, labels)
	if got, want := un.Target(), Target; got != want {
		t.Errorf("Uninstaller.Target() = %q, want %q", got, want)
	}
}

func TestLister_Target(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ls := NewLister(userRoot, projectRoot, labels)
	if got, want := ls.Target(), Target; got != want {
		t.Errorf("Lister.Target() = %q, want %q", got, want)
	}
}

func TestInstaller_Install_writesContent(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	art := sampleSkillArtifact()
	inst, err := ins.Install(context.Background(), inventory.ScopeUser, art)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(userRoot, "skills/sample/SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile(content): %v", err)
	}
	if string(got) != string(art.Content) {
		t.Errorf("content mismatch: got=%q want=%q", got, art.Content)
	}
	stored, err := labels.Get(context.Background(), inventory.ScopeUser, inst.ID)
	if err != nil {
		t.Fatalf("LabelStore.Get: %v", err)
	}
	if stored.ArtifactPath != art.Path {
		t.Errorf("label ArtifactPath = %q, want %q", stored.ArtifactPath, art.Path)
	}
	if inst.Label.Target != Target {
		t.Errorf("Label.Target = %q, want %q", inst.Label.Target, Target)
	}
	if inst.Label.Scope != inventory.ScopeUser {
		t.Errorf("Label.Scope = %q, want %q", inst.Label.Scope, inventory.ScopeUser)
	}
	if string(inst.ID) != "skills/sample/SKILL.md" {
		t.Errorf("Installation.ID = %q, want %q", inst.ID, "skills/sample/SKILL.md")
	}
}

func TestInstaller_Install_projectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTestRootsUserOnly(t)
	ins := NewInstaller(userRoot, "", labels)
	_, err := ins.Install(context.Background(), inventory.ScopeProject, sampleSkillArtifact())
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("Install(ScopeProject) err = %v, want ErrProjectRootNotConfigured", err)
	}
}

func TestInstaller_Install_invalidArtifactPath(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	art := source.Artifact{
		Target:  Target,
		Path:    "hooks/foo.md",
		Content: []byte("x"),
	}
	_, err := ins.Install(context.Background(), inventory.ScopeUser, art)
	if !errors.Is(err, ErrInvalidArtifactPath) {
		t.Errorf("Install() err = %v, want ErrInvalidArtifactPath", err)
	}
}

func TestInstaller_Install_unmanagedArtifactExists(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	target := filepath.Join(userRoot, "skills/sample/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(target, []byte("preexisting"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ins.Install(context.Background(), inventory.ScopeUser, sampleSkillArtifact())
	if !errors.Is(err, ErrUnmanagedArtifactExists) {
		t.Errorf("Install() err = %v, want ErrUnmanagedArtifactExists", err)
	}
}

func TestUninstaller_Uninstall_projectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTestRootsUserOnly(t)
	un := NewUninstaller(userRoot, "", labels)
	inst := inventory.Installation{
		ID: inventory.InstallationID("skills/x/SKILL.md"),
		Label: inventory.Label{
			Target: Target,
			Scope:  inventory.ScopeProject,
		},
	}
	err := un.Uninstall(context.Background(), inst)
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("Uninstall() err = %v, want ErrProjectRootNotConfigured", err)
	}
}

func TestLister_List_projectRootNotConfigured(t *testing.T) {
	userRoot, labels := newTestRootsUserOnly(t)
	ls := NewLister(userRoot, "", labels)
	_, err := ls.List(context.Background(), inventory.ScopeProject)
	if !errors.Is(err, ErrProjectRootNotConfigured) {
		t.Errorf("List(ScopeProject) err = %v, want ErrProjectRootNotConfigured", err)
	}
}

func TestLister_List_excludesOrphanLabel(t *testing.T) {
	// A label whose artifact file is missing must be excluded from List.
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	ls := NewLister(userRoot, projectRoot, labels)
	inst, err := ins.Install(context.Background(), inventory.ScopeUser, sampleSkillArtifact())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := os.Remove(filepath.Join(userRoot, "skills/sample/SKILL.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	got, err := ls.List(context.Background(), inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() returned %d items (with orphan label), want 0; first ID=%q", len(got), inst.ID)
	}
}

func TestRoundtrip_Install_List_Uninstall(t *testing.T) {
	userRoot, projectRoot, labels := newTestRoots(t)
	ins := NewInstaller(userRoot, projectRoot, labels)
	ls := NewLister(userRoot, projectRoot, labels)
	un := NewUninstaller(userRoot, projectRoot, labels)
	ctx := context.Background()

	art := sampleSkillArtifact()
	inst, err := ins.Install(ctx, inventory.ScopeUser, art)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	list, err := ls.List(ctx, inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List returned %d items, want 1", len(list))
	}
	if list[0].ID != inst.ID {
		t.Errorf("List ID = %q, want %q", list[0].ID, inst.ID)
	}
	if list[0].Label.Target != Target || list[0].Label.Scope != inventory.ScopeUser {
		t.Errorf("List Label = %v, want target=%q scope=user", list[0].Label, Target)
	}

	if err := un.Uninstall(ctx, inst); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(userRoot, "skills/sample/SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("content still exists after Uninstall: err=%v", err)
	}
	if _, err := labels.Get(ctx, inventory.ScopeUser, inst.ID); !errors.Is(err, inventory.ErrInstallationNotFound) {
		t.Errorf("label still present after Uninstall: err=%v, want ErrInstallationNotFound", err)
	}
	list2, err := ls.List(ctx, inventory.ScopeUser)
	if err != nil {
		t.Fatalf("List after Uninstall: %v", err)
	}
	if len(list2) != 0 {
		t.Errorf("List after Uninstall returned %d items, want 0", len(list2))
	}
}
