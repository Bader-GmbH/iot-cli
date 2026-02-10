package file

import (
	"testing"
)

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDevice string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "simple path",
			input:      "device-1:/var/log/app.log",
			wantDevice: "device-1",
			wantPath:   "/var/log/app.log",
			wantErr:    false,
		},
		{
			name:       "root path",
			input:      "my-device:/",
			wantDevice: "my-device",
			wantPath:   "/",
			wantErr:    false,
		},
		{
			name:       "directory path",
			input:      "sensor-001:/etc/myapp/",
			wantDevice: "sensor-001",
			wantPath:   "/etc/myapp/",
			wantErr:    false,
		},
		{
			name:       "uuid device id",
			input:      "550e8400-e29b-41d4-a716-446655440000:/data/file.txt",
			wantDevice: "550e8400-e29b-41d4-a716-446655440000",
			wantPath:   "/data/file.txt",
			wantErr:    false,
		},
		{
			name:    "missing colon",
			input:   "device-1/var/log/app.log",
			wantErr: true,
		},
		{
			name:    "empty device",
			input:   ":/var/log/app.log",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   "device-1:",
			wantErr: true,
		},
		{
			name:    "relative path",
			input:   "device-1:var/log/app.log",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRemotePath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRemotePath(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRemotePath(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result.DeviceID != tt.wantDevice {
				t.Errorf("ParseRemotePath(%q).DeviceID = %q, want %q", tt.input, result.DeviceID, tt.wantDevice)
			}

			if result.Path != tt.wantPath {
				t.Errorf("ParseRemotePath(%q).Path = %q, want %q", tt.input, result.Path, tt.wantPath)
			}
		})
	}
}

func TestIsRemotePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"device-1:/var/log/app.log", true},
		{"device:/path", true},
		{"./local/path", false},
		{"/absolute/local/path", false},
		{"C:\\Windows\\path", false}, // Windows drive letter
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsRemotePath(tt.input)
			if got != tt.want {
				t.Errorf("IsRemotePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/var/log/app.log", "app.log"},
		{"/etc/myapp/", "myapp"},
		{"/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/", "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := BaseName(tt.input)
			if got != tt.want {
				t.Errorf("BaseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/var/log/", true},
		{"/var/log", false},
		{"/", true},
		{"./dir/", true},
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsDirectory(tt.input)
			if got != tt.want {
				t.Errorf("IsDirectory(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestJoinRemotePath(t *testing.T) {
	tests := []struct {
		base string
		name string
		want string
	}{
		{"/var/log/", "app.log", "/var/log/app.log"},
		{"/var/log", "app.log", "/var/log/app.log"},
		{"/", "file.txt", "/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.base+"+"+tt.name, func(t *testing.T) {
			got := JoinRemotePath(tt.base, tt.name)
			if got != tt.want {
				t.Errorf("JoinRemotePath(%q, %q) = %q, want %q", tt.base, tt.name, got, tt.want)
			}
		})
	}
}

func TestResolveLocalDestination(t *testing.T) {
	tests := []struct {
		name       string
		remotePath string
		localDest  string
		want       string
	}{
		{
			name:       "empty dest uses current dir",
			remotePath: "/var/log/app.log",
			localDest:  "",
			want:       "./app.log",
		},
		{
			name:       "directory dest appends filename",
			remotePath: "/var/log/app.log",
			localDest:  "./logs/",
			want:       "logs/app.log",
		},
		{
			name:       "explicit filename",
			remotePath: "/var/log/app.log",
			localDest:  "./myfile.log",
			want:       "./myfile.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveLocalDestination(tt.remotePath, tt.localDest)
			if got != tt.want {
				t.Errorf("ResolveLocalDestination(%q, %q) = %q, want %q",
					tt.remotePath, tt.localDest, got, tt.want)
			}
		})
	}
}

func TestResolveRemoteDestination(t *testing.T) {
	tests := []struct {
		name       string
		localPath  string
		remotePath string
		want       string
	}{
		{
			name:       "directory dest appends filename",
			localPath:  "./script.sh",
			remotePath: "/opt/",
			want:       "/opt/script.sh",
		},
		{
			name:       "explicit remote path",
			localPath:  "./script.sh",
			remotePath: "/opt/myscript.sh",
			want:       "/opt/myscript.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRemoteDestination(tt.localPath, tt.remotePath)
			if got != tt.want {
				t.Errorf("ResolveRemoteDestination(%q, %q) = %q, want %q",
					tt.localPath, tt.remotePath, got, tt.want)
			}
		})
	}
}
