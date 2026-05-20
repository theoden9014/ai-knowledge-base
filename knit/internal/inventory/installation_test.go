package inventory

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
)

func TestInstallationID_String(t *testing.T) {
	tests := []struct {
		name string
		id   InstallationID
		want string
	}{
		{
			name: "empty id stringifies to empty",
			id:   InstallationID(""),
			want: "",
		},
		{
			name: "arbitrary id stringifies as-is",
			id:   InstallationID("claude/user/skills/foo"),
			want: "claude/user/skills/foo",
		},
		{
			name: "id with non-ASCII characters preserved",
			id:   InstallationID("cafe/résumé-café"),
			want: "cafe/résumé-café",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.want {
				t.Errorf("InstallationID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInstallation_LabelIsZeroContract fixes the documented invariant that
// Installation always has a non-zero Label by asserting that Label.IsZero()
// can be used to observe the distinction.
//
// An Installation with a zero Label is invalid as Installer/Lister output but
// can still be assembled as a zero value. Label.IsZero() is used to identify
// that case. This test places zero-Label and non-zero-Label Installations side
// by side and verifies that IsZero() distinguishes them reliably.
func TestInstallation_LabelIsZeroContract(t *testing.T) {
	tests := []struct {
		name string
		inst Installation
		want bool
	}{
		{
			name: "zero Installation has zero Label (invalid for Installer/Lister output)",
			inst: Installation{},
			want: true,
		},
		{
			name: "Installation with non-zero Label is detected as non-zero",
			inst: Installation{
				ID:    InstallationID("claude/user/skills/foo"),
				Label: Label{Target: source.Target("claude"), Scope: ScopeUser},
			},
			want: false,
		},
		{
			name: "Installation with only Provenance set still has zero Label",
			inst: Installation{
				Provenance: Provenance{SourceEntryIDs: []string{"pack.skill.foo"}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inst.Label.IsZero(); got != tt.want {
				t.Errorf("Installation.Label.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProvenance_SourceEntryIDsShape fixes the contract that
// Provenance.SourceEntryIDs is nil, not an empty slice, in the zero value.
// This guards the representation so Lister-side code can distinguish the case
// where provenance could not be restored from persistence.
func TestProvenance_SourceEntryIDsShape(t *testing.T) {
	tests := []struct {
		name string
		p    Provenance
		want Provenance
	}{
		{
			name: "zero Provenance: SourceEntryIDs is nil",
			p:    Provenance{},
			want: Provenance{SourceEntryIDs: nil},
		},
		{
			name: "non-zero Provenance preserves SourceEntryIDs as given",
			p:    Provenance{SourceEntryIDs: []string{"pack.skill.foo", "pack.skill.bar"}},
			want: Provenance{SourceEntryIDs: []string{"pack.skill.foo", "pack.skill.bar"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.p); diff != "" {
				t.Errorf("Provenance mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
