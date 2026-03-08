package secrets

import "testing"

func TestGenerateRandomStringRejectsNonPositiveLength(t *testing.T) {
	if _, err := generateRandomString(0); err == nil {
		t.Fatal("expected error for zero length, got nil")
	}

	if _, err := generateRandomString(-1); err == nil {
		t.Fatal("expected error for negative length, got nil")
	}
}

func TestGeneratorUsesDefaultLengthWhenRandomLengthIsZero(t *testing.T) {
	generator := NewGenerator(nil)
	entry := Entry{
		Path:    "secret/path",
		DataKey: "PASSWORD",
		Settings: SecretSettings{
			Type:   "random",
			Length: 0,
		},
	}

	data, err := generator.Generate(entry)
	if err != nil {
		t.Fatalf("expected generation to succeed, got %v", err)
	}

	value, ok := data["PASSWORD"].(string)
	if !ok {
		t.Fatalf("expected generated value to be string, got %T", data["PASSWORD"])
	}

	if len(value) != defaultKeyLength {
		t.Fatalf("expected generated value length %d, got %d", defaultKeyLength, len(value))
	}
}
