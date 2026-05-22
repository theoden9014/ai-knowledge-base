package cli

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTriageArg(t *testing.T) {
	type args struct {
		arg string
	}
	tests := []struct {
		name string
		args args
		want TriagedArg
	}{
		{
			name: "empty arg falls through to pack-name with empty Cleaned",
			args: args{arg: ""},
			want: TriagedArg{Kind: ArgKindPackName, Cleaned: ""},
		},
		{
			name: "kebab-case pack name",
			args: args{arg: "structure-behavior-design"},
			want: TriagedArg{Kind: ArgKindPackName, Cleaned: "structure-behavior-design"},
		},
		{
			name: "single token containing dot stays pack-name (caller rejects with ErrUsage)",
			args: args{arg: "foo.bar"},
			want: TriagedArg{Kind: ArgKindPackName, Cleaned: "foo.bar"},
		},
		{
			name: "bare host-like token without slash stays pack-name",
			args: args{arg: "github.com"},
			want: TriagedArg{Kind: ArgKindPackName, Cleaned: "github.com"},
		},
		{
			name: "absolute path is local-path",
			args: args{arg: "/Users/alice/work/pack"},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "/Users/alice/work/pack"},
		},
		{
			name: "dot-slash relative path is local-path",
			args: args{arg: "./knowledge/pack"},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "knowledge/pack"},
		},
		{
			name: "dot-dot-slash relative path is local-path",
			args: args{arg: "../packs/my-pack"},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "../packs/my-pack"},
		},
		{
			name: "single dot is local-path",
			args: args{arg: "."},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "."},
		},
		{
			name: "double dot is local-path",
			args: args{arg: ".."},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: ".."},
		},
		{
			name: "implicit relative path with slash (no host shape) is local-path",
			args: args{arg: "knowledge/pack"},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "knowledge/pack"},
		},
		{
			name: "github host with owner and repo is remote-url",
			args: args{arg: "github.com/theoden9014/ai-knowledge-base"},
			want: TriagedArg{Kind: ArgKindRemoteURL, Cleaned: "github.com/theoden9014/ai-knowledge-base"},
		},
		{
			name: "github host with subpath is remote-url",
			args: args{arg: "github.com/theoden9014/ai-knowledge-base/knowledge/structure-behavior-design"},
			want: TriagedArg{Kind: ArgKindRemoteURL, Cleaned: "github.com/theoden9014/ai-knowledge-base/knowledge/structure-behavior-design"},
		},
		{
			name: "https scheme is stripped and classified as remote-url",
			args: args{arg: "https://github.com/owner/repo"},
			want: TriagedArg{Kind: ArgKindRemoteURL, Cleaned: "github.com/owner/repo"},
		},
		{
			name: "http scheme is stripped and classified as remote-url",
			args: args{arg: "http://github.com/owner/repo"},
			want: TriagedArg{Kind: ArgKindRemoteURL, Cleaned: "github.com/owner/repo"},
		},
		{
			name: "gitlab-style host is remote-url",
			args: args{arg: "gitlab.com/group/project/sub"},
			want: TriagedArg{Kind: ArgKindRemoteURL, Cleaned: "gitlab.com/group/project/sub"},
		},
		{
			name: "dot-slash prefix is local even when remainder looks host-like",
			args: args{arg: "./github.com/owner/repo"},
			want: TriagedArg{Kind: ArgKindLocalPath, Cleaned: "github.com/owner/repo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TriageArg(tt.args.arg); !cmp.Equal(tt.want, got) {
				t.Errorf("TriageArg() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}
