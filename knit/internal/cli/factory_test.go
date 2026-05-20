package cli

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/claude"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/codex"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/gemini"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestNewDistributionFactory(t *testing.T) {
	tests := []struct {
		name        string
		userBase    string
		projectRoot string
		codexHome   string
	}{
		{name: "all set", userBase: "/home/u", projectRoot: "/home/u/proj", codexHome: "/home/u/.codex"},
		{name: "no project root", userBase: "/home/u", projectRoot: "", codexHome: ""},
		{name: "no codex home", userBase: "/home/u", projectRoot: "/home/u/proj", codexHome: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory(tt.userBase, tt.projectRoot, tt.codexHome)
			if f == nil {
				t.Fatalf("NewDistributionFactory returned nil")
			}
			if f.userBase != tt.userBase || f.projectRoot != tt.projectRoot || f.codexHome != tt.codexHome {
				t.Errorf("fields mismatch: got userBase=%q projectRoot=%q codexHome=%q",
					f.userBase, f.projectRoot, f.codexHome)
			}
		})
	}
}

func TestDistributionFactory_SupportedTargets(t *testing.T) {
	f := NewDistributionFactory("/h", "/h/p", "")
	got := f.SupportedTargets()
	want := []source.Target{claude.Target, codex.Target, gemini.Target}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SupportedTargets() mismatch (-want +got):\n%s", diff)
	}
}

func TestDistributionFactory_ResolveTargets(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		want      []source.Target
		wantErr   error
	}{
		{name: "claude", flagValue: "claude", want: []source.Target{claude.Target}},
		{name: "codex", flagValue: "codex", want: []source.Target{codex.Target}},
		{name: "gemini", flagValue: "gemini", want: []source.Target{gemini.Target}},
		{
			name:      "all",
			flagValue: "all",
			want:      []source.Target{claude.Target, codex.Target, gemini.Target},
		},
		{name: "empty rejected", flagValue: "", wantErr: ErrInvalidFlagValue},
		{name: "unknown rejected", flagValue: "ollama", wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", "/h/p", "")
			got, err := f.ResolveTargets(tt.flagValue)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDistributionFactory_Builder(t *testing.T) {
	tests := []struct {
		name    string
		target  source.Target
		wantErr error
	}{
		{name: "claude", target: claude.Target},
		{name: "codex", target: codex.Target},
		{name: "gemini", target: gemini.Target},
		{name: "unknown", target: source.Target("ollama"), wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", "/h/p", "")
			got, err := f.Builder(tt.target)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Target() != tt.target {
				t.Errorf("Target() = %q, want %q", got.Target(), tt.target)
			}
		})
	}
}

func TestDistributionFactory_Installer(t *testing.T) {
	tests := []struct {
		name    string
		target  source.Target
		wantErr error
	}{
		{name: "claude", target: claude.Target},
		{name: "codex", target: codex.Target},
		{name: "gemini", target: gemini.Target},
		{name: "unknown", target: source.Target("ollama"), wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", "/h/p", "")
			got, err := f.Installer(tt.target)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Target() != tt.target {
				t.Errorf("Target() = %q, want %q", got.Target(), tt.target)
			}
		})
	}
}

func TestDistributionFactory_Uninstaller(t *testing.T) {
	tests := []struct {
		name    string
		target  source.Target
		wantErr error
	}{
		{name: "claude", target: claude.Target},
		{name: "codex", target: codex.Target},
		{name: "gemini", target: gemini.Target},
		{name: "unknown", target: source.Target("ollama"), wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", "/h/p", "")
			got, err := f.Uninstaller(tt.target)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Target() != tt.target {
				t.Errorf("Target() = %q, want %q", got.Target(), tt.target)
			}
		})
	}
}

func TestDistributionFactory_Lister(t *testing.T) {
	tests := []struct {
		name    string
		target  source.Target
		wantErr error
	}{
		{name: "claude", target: claude.Target},
		{name: "codex", target: codex.Target},
		{name: "gemini", target: gemini.Target},
		{name: "unknown", target: source.Target("ollama"), wantErr: ErrInvalidFlagValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", "/h/p", "")
			got, err := f.Lister(tt.target)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Target() != tt.target {
				t.Errorf("Target() = %q, want %q", got.Target(), tt.target)
			}
		})
	}
}

func TestDistributionFactory_userRootFor(t *testing.T) {
	tests := []struct {
		name      string
		userBase  string
		codexHome string
		target    source.Target
		want      string
	}{
		{name: "claude", userBase: "/home/u", codexHome: "", target: claude.Target, want: "/home/u/.claude"},
		{name: "codex no CODEX_HOME → fallback to userBase/.codex", userBase: "/home/u", codexHome: "", target: codex.Target, want: "/home/u/.codex"},
		{name: "codex with CODEX_HOME prioritized", userBase: "/home/u", codexHome: "/custom/codex", target: codex.Target, want: "/custom/codex"},
		{name: "gemini", userBase: "/home/u", codexHome: "", target: gemini.Target, want: "/home/u/.gemini"},
		{name: "unknown target → empty", userBase: "/home/u", codexHome: "", target: source.Target("ollama"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory(tt.userBase, "", tt.codexHome)
			if got := f.userRootFor(tt.target); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDistributionFactory_projectRootFor(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		target      source.Target
		want        string
	}{
		{name: "claude", projectRoot: "/p", target: claude.Target, want: "/p/.claude"},
		{name: "codex", projectRoot: "/p", target: codex.Target, want: "/p/.codex"},
		{name: "gemini", projectRoot: "/p", target: gemini.Target, want: "/p/.gemini"},
		{name: "no project root → empty", projectRoot: "", target: claude.Target, want: ""},
		{name: "unknown target → empty", projectRoot: "/p", target: source.Target("ollama"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", tt.projectRoot, "")
			if got := f.projectRootFor(tt.target); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDistributionFactory_userKnitRoot(t *testing.T) {
	tests := []struct {
		name     string
		userBase string
		want     string
	}{
		{name: "userBase set joins .knit", userBase: "/home/u", want: "/home/u/.knit"},
		{name: "empty userBase propagates as empty", userBase: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory(tt.userBase, "", "")
			if got := f.userKnitRoot(); got != tt.want {
				t.Errorf("userKnitRoot() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDistributionFactory_projectKnitRoot(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		want        string
	}{
		{name: "projectRoot set joins .knit", projectRoot: "/p", want: "/p/.knit"},
		{name: "empty projectRoot propagates as empty", projectRoot: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDistributionFactory("/h", tt.projectRoot, "")
			if got := f.projectKnitRoot(); got != tt.want {
				t.Errorf("projectKnitRoot() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_labelsRootOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		knitRoot string
		want     string
	}{
		{name: "non-empty knit root maps to <knitRoot>/labels", knitRoot: "/home/u/.knit", want: "/home/u/.knit/labels"},
		{name: "empty knit root preserved as empty sentinel", knitRoot: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := labelsRootOrEmpty(tt.knitRoot); got != tt.want {
				t.Errorf("labelsRootOrEmpty(%q) = %q, want %q", tt.knitRoot, got, tt.want)
			}
		})
	}
}
