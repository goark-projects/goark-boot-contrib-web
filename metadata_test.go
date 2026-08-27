package gbcweb

import "testing"

func TestModuleMetadata(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "module path", got: ModulePath, want: "goark.dev/gbc-web"},
		{name: "repository", got: Repository, want: "goark-boot-contrib-web"},
		{name: "starter id", got: StarterID, want: "goark.boot.contrib.web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("metadata mismatch: got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
