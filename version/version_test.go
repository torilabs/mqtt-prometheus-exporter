package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func Test_commitFrom(t *testing.T) {
	tests := []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{name: "no vcs information", want: "unknown"},
		{
			name:     "short revision is kept",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}},
			want:     "abc123",
		},
		{
			name:     "long revision is shortened",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}},
			want:     "0123456789ab",
		},
		{
			name: "modified tree is flagged",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.modified", Value: "true"},
			},
			want: "abc123-dirty",
		},
		{
			name: "unmodified tree",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.modified", Value: "false"},
			},
			want: "abc123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commitFrom(&debug.BuildInfo{Settings: tt.settings}); got != tt.want {
				t.Errorf("commitFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = "v9.8.7"

	got := String()
	for _, want := range []string{"v9.8.7", "commit ", runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}

func TestVersion_DefaultsToDev(t *testing.T) {
	if Version == "" {
		t.Error("Version must never be empty")
	}
}
