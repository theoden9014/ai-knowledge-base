package cli

import (
	"flag"
	"io"
	"testing"
)

// registerFlagSpec captures the contract for each register*Flag helper:
// flag name, default value (when no args are parsed), and overridden
// value (when "--<name>=<override>" is parsed).
type registerFlagSpec struct {
	name     string
	flagName string
	register func(fs *flag.FlagSet) *string
	def      string
	override string
}

func TestRegisterFlagHelpers(t *testing.T) {
	specs := []registerFlagSpec{
		{name: "scope flag", flagName: "scope", register: registerScopeFlag, def: "user", override: "project"},
		{name: "target flag", flagName: "target", register: registerTargetFlag, def: "all", override: "claude"},
		{name: "output-dir flag (--output)", flagName: "output", register: registerOutputDirFlag, def: "", override: "/tmp/out"},
	}
	for _, sp := range specs {
		sp := sp
		t.Run(sp.name+"/default value", func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			ptr := sp.register(fs)
			if ptr == nil {
				t.Fatalf("register returned nil pointer")
			}
			if got := *ptr; got != sp.def {
				t.Errorf("default value = %q, want %q", got, sp.def)
			}
			if fs.Lookup(sp.flagName) == nil {
				t.Errorf("flag %q not registered", sp.flagName)
			}
		})
		t.Run(sp.name+"/parses override", func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			ptr := sp.register(fs)
			if err := fs.Parse([]string{"--" + sp.flagName + "=" + sp.override}); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := *ptr; got != sp.override {
				t.Errorf("override value = %q, want %q", got, sp.override)
			}
		})
	}
}

// Output flag must also be accessible via the short alias "-o" per build
// command convention. Verified separately because the alias is only
// relevant to registerOutputDirFlag.
func Test_registerOutputDirFlag_shortAlias(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ptr := registerOutputDirFlag(fs)
	if err := fs.Parse([]string{"-o", "/tmp/short"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := *ptr; got != "/tmp/short" {
		t.Errorf("short alias -o = %q, want %q", got, "/tmp/short")
	}
}
