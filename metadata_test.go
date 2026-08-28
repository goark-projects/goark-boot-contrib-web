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
		{name: "deployment bean", got: BeanNameDeployment, want: "goark.boot.web.deployment"},
		{name: "http client builder bean", got: BeanNameHTTPClientBuilder, want: "goark.boot.web.clientBuilder"},
		{name: "http client bean", got: BeanNameHTTPClient, want: "goark.boot.web.client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("metadata mismatch: got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
