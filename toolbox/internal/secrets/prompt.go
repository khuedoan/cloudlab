package secrets

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

type HuhPrompter struct{}

func (HuhPrompter) PromptSecret(description string) (string, error) {
	var value string

	err := huh.NewInput().
		Title(description).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		Run()
	if err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}

	if value == "" {
		return "", fmt.Errorf("no input provided")
	}

	return value, nil
}
