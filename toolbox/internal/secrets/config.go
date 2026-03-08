package secrets

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure.
// Format:
//
//	secrets:
//	  secret/path:
//	    KEY_NAME:
//	      type: random|ssh|manual
//	      ...
type Config struct {
	Secrets map[string]map[string]SecretSettings `yaml:"secrets"`
}

type SecretSettings struct {
	Type        string `yaml:"type"`
	Length      int    `yaml:"length,omitempty"`
	Algorithm   string `yaml:"algorithm,omitempty"`
	PublicKey   string `yaml:"public_key,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Entry struct {
	Path     string
	DataKey  string
	Settings SecretSettings
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &config, nil
}

func ParseAndValidate(config *Config) ([]Entry, error) {
	var entries []Entry

	paths := make([]string, 0, len(config.Secrets))
	for path := range config.Secrets {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		keys := config.Secrets[path]

		dataKeys := make([]string, 0, len(keys))
		for k := range keys {
			dataKeys = append(dataKeys, k)
		}
		sort.Strings(dataKeys)

		for _, dataKey := range dataKeys {
			settings := keys[dataKey]
			if err := validateSettings(path, dataKey, settings); err != nil {
				return nil, err
			}

			entries = append(entries, Entry{
				Path:     path,
				DataKey:  dataKey,
				Settings: settings,
			})
		}
	}

	return entries, nil
}

func validateSettings(path, dataKey string, settings SecretSettings) error {
	switch settings.Type {
	case "random":
		if settings.Length < 0 {
			return fmt.Errorf("%s#%s: length must be >= 0", path, dataKey)
		}
	case "ssh":
		// valid
	case "manual":
		// valid
	case "":
		return fmt.Errorf("%s#%s: type is required", path, dataKey)
	default:
		return fmt.Errorf("%s#%s: unknown type %q", path, dataKey, settings.Type)
	}
	return nil
}
