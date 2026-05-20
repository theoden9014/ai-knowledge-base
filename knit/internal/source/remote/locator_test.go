package remote

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLocator_CloneURL(t *testing.T) {
	type fields struct {
		Host    string
		Owner   string
		Repo    string
		Subpath string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "github clone url ignores subpath",
			fields: fields{Host: "github.com", Owner: "o", Repo: "r"},
			want:   "https://github.com/o/r.git",
		},
		{
			name:   "subpath is omitted from clone url",
			fields: fields{Host: "github.com", Owner: "o", Repo: "r", Subpath: "knowledge/pack"},
			want:   "https://github.com/o/r.git",
		},
		{
			name:   "host preserved verbatim",
			fields: fields{Host: "gitlab.com", Owner: "o", Repo: "r"},
			want:   "https://gitlab.com/o/r.git",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Locator{
				Host:    tt.fields.Host,
				Owner:   tt.fields.Owner,
				Repo:    tt.fields.Repo,
				Subpath: tt.fields.Subpath,
			}
			if got := l.CloneURL(); got != tt.want {
				t.Errorf("Locator.CloneURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		arg string
	}
	tests := []struct {
		name    string
		args    args
		want    *Locator
		wantErr bool
	}{
		{
			name: "host/owner/repo with no subpath",
			args: args{arg: "github.com/theoden9014/ai-knowledge-base"},
			want: &Locator{Host: "github.com", Owner: "theoden9014", Repo: "ai-knowledge-base"},
		},
		{
			name: "host/owner/repo with subpath",
			args: args{arg: "github.com/theoden9014/ai-knowledge-base/knowledge/structure-behavior-design"},
			want: &Locator{
				Host:    "github.com",
				Owner:   "theoden9014",
				Repo:    "ai-knowledge-base",
				Subpath: "knowledge/structure-behavior-design",
			},
		},
		{
			name: "trailing .git on repo is stripped",
			args: args{arg: "github.com/o/r.git"},
			want: &Locator{Host: "github.com", Owner: "o", Repo: "r"},
		},
		{
			name: "trailing .git stripped with subpath present",
			args: args{arg: "github.com/o/r.git/sub"},
			want: &Locator{Host: "github.com", Owner: "o", Repo: "r", Subpath: "sub"},
		},
		{
			name: "host is lower-cased",
			args: args{arg: "GitHub.COM/o/r"},
			want: &Locator{Host: "github.com", Owner: "o", Repo: "r"},
		},
		{
			name:    "rejects http:// scheme",
			args:    args{arg: "http://github.com/o/r"},
			wantErr: true,
		},
		{
			name:    "rejects https:// scheme",
			args:    args{arg: "https://github.com/o/r"},
			wantErr: true,
		},
		{
			name:    "rejects query string",
			args:    args{arg: "github.com/o/r?ref=main"},
			wantErr: true,
		},
		{
			name:    "rejects fragment",
			args:    args{arg: "github.com/o/r#frag"},
			wantErr: true,
		},
		{
			name:    "rejects consecutive slashes",
			args:    args{arg: "github.com//o/r"},
			wantErr: true,
		},
		{
			name:    "rejects consecutive slashes in subpath",
			args:    args{arg: "github.com/o/r//sub"},
			wantErr: true,
		},
		{
			name:    "rejects backslashes",
			args:    args{arg: `github.com\o\r`},
			wantErr: true,
		},
		{
			name:    "rejects trailing slash",
			args:    args{arg: "github.com/o/r/"},
			wantErr: true,
		},
		{
			name:    "rejects empty string",
			args:    args{arg: ""},
			wantErr: true,
		},
		{
			name:    "rejects missing repo segment",
			args:    args{arg: "github.com/o"},
			wantErr: true,
		},
		{
			name:    "rejects single segment",
			args:    args{arg: "github.com"},
			wantErr: true,
		},
		{
			name:    "rejects local pack name without dot",
			args:    args{arg: "structure-behavior-design"},
			wantErr: true,
		},
		{
			name:    "rejects leading slash",
			args:    args{arg: "/github.com/o/r"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidLocator) {
					t.Errorf("Parse() error = %v, want errors.Is ErrInvalidLocator", err)
				}
				return
			}
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Parse() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestIsRemoteArg(t *testing.T) {
	type args struct {
		arg string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "github url is remote", args: args{arg: "github.com/o/r"}, want: true},
		{name: "gitlab url is remote", args: args{arg: "gitlab.com/o/r/sub"}, want: true},
		{name: "kebab-case pack name is local", args: args{arg: "structure-behavior-design"}, want: false},
		{name: "single word pack is local", args: args{arg: "mypack"}, want: false},
		{name: "empty string is local", args: args{arg: ""}, want: false},
		{name: "absolute path is local not remote", args: args{arg: "/foo/bar"}, want: false},
		{name: "dot prefix relative path is local not remote", args: args{arg: "./foo"}, want: false},
		{name: "dotdot prefix relative path is local not remote", args: args{arg: "../foo"}, want: false},
		{name: "bare dot is local not remote", args: args{arg: "."}, want: false},
		{name: "bare dotdot is local not remote", args: args{arg: ".."}, want: false},
		{name: "dot only after first segment is local", args: args{arg: "foo/bar.com"}, want: false},
		{name: "bare host without path is not remote", args: args{arg: "github.com"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRemoteArg(tt.args.arg); got != tt.want {
				t.Errorf("IsRemoteArg() = %v, want %v", got, tt.want)
			}
		})
	}
}
