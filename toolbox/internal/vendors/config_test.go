package vendors

import (
	"strings"
	"testing"
)

func TestParseAndValidate(t *testing.T) {
	cases := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{"missing chart versions", &Config{Items: map[string]Vendor{
			"vendor/charts/dex": {Kind: "chart", Chart: "dex", RepoURL: "https://charts.dexidp.io"},
		}}, "versions is required"},
		{"missing image kind", &Config{Items: map[string]Vendor{
			"vendor/charts/dex": {Versions: []string{"0.23.0"}, Chart: "dex", RepoURL: "https://charts.dexidp.io"},
		}}, "kind is required"},
		{"image with source and no versions", &Config{Items: map[string]Vendor{
			"vendor/images/dex": {Kind: "image", Source: "ghcr.io/dexidp/dex"},
		}}, "versions is required"},
		{"valid image versions", &Config{Items: map[string]Vendor{
			"vendor/images/dex": {Kind: "image", Source: "ghcr.io/dexidp/dex", Versions: []string{"v2.43.1"}},
		}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAndValidate(tc.config)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected validation error, got nil")
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected validation error %q, got %v", tc.wantErr, err)
			}
		})
	}
}
