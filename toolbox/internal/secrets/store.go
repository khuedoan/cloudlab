package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/vault/api"
)

type Store struct {
	vault *api.Client
}

func NewStore(vault *api.Client) *Store {
	return &Store{vault: vault}
}

func (s *Store) Process(ctx context.Context, e Entry, generator *Generator) error {
	mount, path, err := parsePath(e.Path)
	if err != nil {
		return err
	}

	existing, err := s.vault.KVv2(mount).Get(ctx, path)
	if err != nil && !errors.Is(err, api.ErrSecretNotFound) {
		return fmt.Errorf("read existing secret: %w", err)
	}

	data := make(map[string]interface{})
	if existing != nil && existing.Data != nil {
		for k, v := range existing.Data {
			data[k] = v
		}
	}

	keysToCheck := []string{e.DataKey}
	if e.Settings.Type == "ssh" {
		pubKey := e.Settings.PublicKey
		if pubKey == "" {
			pubKey = e.DataKey + ".pub"
		}
		keysToCheck = append(keysToCheck, pubKey)
	}

	allExist := true
	for _, k := range keysToCheck {
		if _, exists := data[k]; !exists {
			allExist = false
			break
		}
	}
	if allExist {
		log.Info("secret already exists, skipping", "path", e.Path, "key", e.DataKey)
		return nil
	}

	newData, err := generator.Generate(e)
	if err != nil {
		return err
	}

	for k, v := range newData {
		data[k] = v
	}

	if _, err := s.vault.KVv2(mount).Put(ctx, path, data); err != nil {
		return fmt.Errorf("write to Vault: %w", err)
	}

	return nil
}

func parsePath(fullPath string) (mount, path string, err error) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path %q: expected format mount/path", fullPath)
	}
	return parts[0], parts[1], nil
}
