package sdlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangelogFromCommitMessage(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want *Changelog
	}{
		{
			name: "should parse the commit message and return all fields in ChangeLog",
			args: args{
				msg: "feat(app/subapp)!: message with breaking change",
			},
			want: &Changelog{
				Type:     "feat",
				Scope:    "app/subapp",
				Message:  "message with breaking change",
				Breaking: true,
			},
		},
		{
			name: "should parse the commit message and return non breaking change in ChangeLog",
			args: args{
				msg: "feat(app/subapp): message with non breaking change",
			},
			want: &Changelog{
				Type:     "feat",
				Scope:    "app/subapp",
				Message:  "message with non breaking change",
				Breaking: false,
			},
		},
		{
			name: "should parse the commit message and return commit with no scope in ChangeLog",
			args: args{
				msg: "feat: message with no scope",
			},
			want: &Changelog{
				Type:     "feat",
				Message:  "message with no scope",
				Breaking: false,
			},
		},
		{
			name: "should parse the commit message and return commit with no separator in ChangeLog",
			args: args{
				msg: "feat(adsol) message with no separator",
			},
			want: &Changelog{
				Type:     "feat",
				Scope:    "adsol",
				Message:  "message with no separator",
				Breaking: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChangelogFromCommitMessage(tt.args.msg)

			assert.Equal(t, tt.want, got)
		})
	}
}
