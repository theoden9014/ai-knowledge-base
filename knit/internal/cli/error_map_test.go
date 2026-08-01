package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/claude"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/codex"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/distribution/gemini"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/inventory"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source"
	"github.com/theoden9014/ai-knowledge-base/knit/internal/source/remote"
)

func Test_errorToExitCode(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want ExitCode
	}{
		// nil
		{name: "nil → ExitSuccess", args: args{err: nil}, want: ExitSuccess},

		// cli sentinels
		{name: "ErrUsage → ExitUsage", args: args{err: ErrUsage}, want: ExitUsage},
		{name: "ErrUnknownCommand → ExitUsage", args: args{err: ErrUnknownCommand}, want: ExitUsage},
		{name: "ErrMissingArgument → ExitUsage", args: args{err: ErrMissingArgument}, want: ExitUsage},
		{name: "ErrInvalidFlagValue → ExitUsage", args: args{err: ErrInvalidFlagValue}, want: ExitUsage},
		{name: "ErrHomeNotSet → ExitConfig", args: args{err: ErrHomeNotSet}, want: ExitConfig},
		{name: "ErrProjectRootNotFound → ExitConfig", args: args{err: ErrProjectRootNotFound}, want: ExitConfig},
		{name: "ErrKnowledgeDirNotFound → ExitConfig", args: args{err: ErrKnowledgeDirNotFound}, want: ExitConfig},
		{name: "ErrPartialFailure → ExitPartial", args: args{err: ErrPartialFailure}, want: ExitPartial},

		// AggregateError matches ErrPartialFailure via Is
		{
			name: "AggregateError → ExitPartial",
			args: args{err: &AggregateError{Failures: []TargetFailure{
				{Target: "claude", Err: errors.New("x")},
			}}},
			want: ExitPartial,
		},

		// inventory sentinels
		{name: "inventory.ErrInvalidScope → ExitUsage", args: args{err: inventory.ErrInvalidScope}, want: ExitUsage},
		{name: "inventory.ErrTargetMismatch → ExitGeneral", args: args{err: inventory.ErrTargetMismatch}, want: ExitGeneral},
		{name: "inventory.ErrAlreadyInstalled → ExitConflict", args: args{err: inventory.ErrAlreadyInstalled}, want: ExitConflict},
		{name: "inventory.ErrInstallationNotFound → ExitNotFound", args: args{err: inventory.ErrInstallationNotFound}, want: ExitNotFound},

		// source sentinels
		{name: "source.ErrManifestNotFound → ExitNotFound", args: args{err: source.ErrManifestNotFound}, want: ExitNotFound},
		{name: "source.ErrEntryNotFound → ExitNotFound", args: args{err: source.ErrEntryNotFound}, want: ExitNotFound},
		{name: "source.ErrSchemaViolation → ExitGeneral", args: args{err: source.ErrSchemaViolation}, want: ExitGeneral},
		{name: "source.ErrIDMismatch → ExitGeneral", args: args{err: source.ErrIDMismatch}, want: ExitGeneral},
		{name: "source.ErrDuplicateEntryID → ExitGeneral", args: args{err: source.ErrDuplicateEntryID}, want: ExitGeneral},
		{name: "source.ErrInvalidKind → ExitGeneral", args: args{err: source.ErrInvalidKind}, want: ExitGeneral},

		// distribution sentinels (per package)
		{name: "claude.ErrProjectRootNotConfigured → ExitConfig", args: args{err: claude.ErrProjectRootNotConfigured}, want: ExitConfig},
		{name: "claude.ErrUnmanagedArtifactExists → ExitConflict", args: args{err: claude.ErrUnmanagedArtifactExists}, want: ExitConflict},
		{name: "claude.ErrInvalidArtifactPath → ExitGeneral", args: args{err: claude.ErrInvalidArtifactPath}, want: ExitGeneral},
		{name: "codex.ErrProjectRootNotConfigured → ExitConfig", args: args{err: codex.ErrProjectRootNotConfigured}, want: ExitConfig},
		{name: "codex.ErrUnmanagedArtifactExists → ExitConflict", args: args{err: codex.ErrUnmanagedArtifactExists}, want: ExitConflict},
		{name: "codex.ErrInvalidArtifactPath → ExitGeneral", args: args{err: codex.ErrInvalidArtifactPath}, want: ExitGeneral},
		{name: "codex.ErrInvalidSkillMetadata → ExitGeneral", args: args{err: codex.ErrInvalidSkillMetadata}, want: ExitGeneral},
		{name: "gemini.ErrProjectRootNotConfigured → ExitConfig", args: args{err: gemini.ErrProjectRootNotConfigured}, want: ExitConfig},
		{name: "gemini.ErrUnmanagedArtifactExists → ExitConflict", args: args{err: gemini.ErrUnmanagedArtifactExists}, want: ExitConflict},
		{name: "gemini.ErrInvalidArtifactPath → ExitGeneral", args: args{err: gemini.ErrInvalidArtifactPath}, want: ExitGeneral},
		{name: "gemini.ErrUnsupportedSkillInvocation → ExitGeneral", args: args{err: gemini.ErrUnsupportedSkillInvocation}, want: ExitGeneral},

		// remote sentinels (Wave4)
		{name: "remote.ErrInvalidLocator → ExitUsage", args: args{err: remote.ErrInvalidLocator}, want: ExitUsage},
		{name: "remote.ErrUnsupportedHost → ExitConfig", args: args{err: remote.ErrUnsupportedHost}, want: ExitConfig},
		{name: "remote.ErrCloneFailed → ExitGeneral", args: args{err: remote.ErrCloneFailed}, want: ExitGeneral},
		{name: "remote.ErrCleanupFailed → ExitGeneral", args: args{err: remote.ErrCleanupFailed}, want: ExitGeneral},
		{
			name: "wrapped remote.ErrUnsupportedHost → ExitConfig",
			args: args{err: fmt.Errorf("dispatch: %w", remote.ErrUnsupportedHost)},
			want: ExitConfig,
		},

		// wrapped error preserves mapping
		{
			name: "wrapped inventory.ErrAlreadyInstalled → ExitConflict",
			args: args{err: fmt.Errorf("install failed: %w", inventory.ErrAlreadyInstalled)},
			want: ExitConflict,
		},

		// unrelated error → fallback ExitGeneral
		{name: "unknown error → ExitGeneral", args: args{err: errors.New("boom")}, want: ExitGeneral},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorToExitCode(tt.args.err); got != tt.want {
				t.Errorf("errorToExitCode() = %v, want %v", got, tt.want)
			}
		})
	}
}
