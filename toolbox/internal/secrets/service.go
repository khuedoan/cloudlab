package secrets

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type Service struct {
	store     *Store
	generator *Generator
}

func NewService(vault *api.Client, prompter Prompter) *Service {
	return &Service{
		store:     NewStore(vault),
		generator: NewGenerator(prompter),
	}
}

func (s *Service) Run(ctx context.Context, entries []Entry) error {
	var autoEntries, manualEntries []Entry
	for _, e := range entries {
		if e.Settings.Type == "manual" {
			manualEntries = append(manualEntries, e)
		} else {
			autoEntries = append(autoEntries, e)
		}
	}

	for _, e := range autoEntries {
		if err := s.store.Process(ctx, e, s.generator); err != nil {
			return fmt.Errorf("process secret %s#%s: %w", e.Path, e.DataKey, err)
		}
	}

	for _, e := range manualEntries {
		if err := s.store.Process(ctx, e, s.generator); err != nil {
			return fmt.Errorf("process secret %s#%s: %w", e.Path, e.DataKey, err)
		}
	}

	return nil
}
