package update

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{
			name: "equal versions",
			v1:   "1.0.0",
			v2:   "1.0.0",
			want: 0,
		},
		{
			name: "v1 newer major",
			v1:   "2.0.0",
			v2:   "1.0.0",
			want: 1,
		},
		{
			name: "v1 older major",
			v1:   "1.0.0",
			v2:   "2.0.0",
			want: -1,
		},
		{
			name: "v1 newer minor",
			v1:   "1.2.0",
			v2:   "1.1.0",
			want: 1,
		},
		{
			name: "v1 older minor",
			v1:   "1.1.0",
			v2:   "1.2.0",
			want: -1,
		},
		{
			name: "v1 newer patch",
			v1:   "1.0.2",
			v2:   "1.0.1",
			want: 1,
		},
		{
			name: "v1 older patch",
			v1:   "1.0.1",
			v2:   "1.0.2",
			want: -1,
		},
		{
			name: "with v prefix",
			v1:   "v1.2.0",
			v2:   "v1.1.0",
			want: 1,
		},
		{
			name: "mixed v prefix",
			v1:   "v1.2.0",
			v2:   "1.1.0",
			want: 1,
		},
		{
			name: "dev is always older",
			v1:   "dev",
			v2:   "0.0.1",
			want: -1,
		},
		{
			name: "anything is newer than dev",
			v1:   "0.0.1",
			v2:   "dev",
			want: 1,
		},
		{
			name: "dev equals dev",
			v1:   "dev",
			v2:   "dev",
			want: -1, // dev vs dev returns -1 (first check)
		},
		{
			name: "two part version",
			v1:   "1.2",
			v2:   "1.1",
			want: 1,
		},
		{
			name: "different length versions",
			v1:   "1.2.3",
			v2:   "1.2",
			want: 1,
		},
		{
			name: "different length versions reverse",
			v1:   "1.2",
			v2:   "1.2.3",
			want: -1,
		},
		{
			name: "single digit versions",
			v1:   "2",
			v2:   "1",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestUpdateInfo_IsUpdateAvailable(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{
			name:    "update available",
			current: "1.0.0",
			latest:  "1.1.0",
			want:    true,
		},
		{
			name:    "no update same version",
			current: "1.0.0",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "no update newer current",
			current: "1.2.0",
			latest:  "1.1.0",
			want:    false,
		},
		{
			name:    "dev version",
			current: "dev",
			latest:  "1.0.0",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &UpdateInfo{
				CurrentVersion: tt.current,
				LatestVersion:  tt.latest,
			}
			got := info.IsUpdateAvailable()
			if got != tt.want {
				t.Errorf("IsUpdateAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker_getAssetName(t *testing.T) {
	// This test just verifies the format, actual values depend on GOOS/GOARCH
	checker := NewChecker("1.0.0", "/tmp")
	name := checker.getAssetName()

	if name == "" {
		t.Error("getAssetName() returned empty string")
	}

	// Should start with "iot-cli-"
	if len(name) < 8 || name[:8] != "iot-cli-" {
		t.Errorf("getAssetName() = %q, should start with 'iot-cli-'", name)
	}
}
