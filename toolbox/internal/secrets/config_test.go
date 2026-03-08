package secrets

import (
	"strings"
	"testing"
)

func TestParseAndValidateRejectsNegativeRandomLength(t *testing.T) {
	config := &Config{
		Secrets: map[string]map[string]SecretSettings{
			"secret/path": {
				"PASSWORD": {
					Type:   "random",
					Length: -1,
				},
			},
		},
	}

	_, err := ParseAndValidate(config)
	if err == nil {
		t.Fatal("expected validation error for negative length, got nil")
	}
	if !strings.Contains(err.Error(), "length must be >= 0") {
		t.Fatalf("expected length validation error, got %v", err)
	}
}

func TestParseAndValidateAllowsZeroRandomLength(t *testing.T) {
	config := &Config{
		Secrets: map[string]map[string]SecretSettings{
			"secret/path": {
				"PASSWORD": {
					Type:   "random",
					Length: 0,
				},
			},
		},
	}

	entries, err := ParseAndValidate(config)
	if err != nil {
		t.Fatalf("expected zero length to be accepted as default, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}
