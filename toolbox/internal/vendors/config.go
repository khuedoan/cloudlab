package vendors

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Items map[string]Vendor `yaml:"vendors"`
}

type Vendor struct {
	Kind                string   `yaml:"kind"`
	RepoURL             string   `yaml:"repo_url,omitempty"`
	Ref                 string   `yaml:"ref,omitempty"`
	Chart               string   `yaml:"chart,omitempty"`
	Versions            []string `yaml:"versions"`
	Source              string   `yaml:"source,omitempty"`
}

type VendorEntry struct {
	Name string
	Vendor
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &config, nil
}

func ParseAndValidate(config *Config) ([]VendorEntry, error) {
	names := make([]string, 0, len(config.Items))
	for name := range config.Items {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]VendorEntry, 0, len(names))

	for _, name := range names {
		vendor := config.Items[name]

		if err := validateDestination(name); err != nil {
			return nil, err
		}

		vendor.Kind = strings.ToLower(vendor.Kind)

		if len(vendor.Versions) == 0 {
			return nil, fmt.Errorf("vendors.%s: versions is required", name)
		}

		for _, version := range vendor.Versions {
			if version == "" {
				return nil, fmt.Errorf("vendors.%s: versions cannot be empty", name)
			}
		}

		switch vendor.Kind {
		case "chart":
			if vendor.Ref != "" {
				if vendor.RepoURL != "" || vendor.Chart != "" {
					return nil, fmt.Errorf("vendors.%s: use either ref or repo_url/chart", name)
				}
			} else {
				if vendor.RepoURL == "" || vendor.Chart == "" {
					return nil, fmt.Errorf("vendors.%s: repo_url and chart are both required", name)
				}
			}

		case "image":
			if vendor.Source == "" {
				return nil, fmt.Errorf("vendors.%s: source is required", name)
			}

		default:
			if vendor.Kind == "" {
				return nil, fmt.Errorf("vendors.%s: kind is required (chart|image)", name)
			}
			return nil, fmt.Errorf("vendors.%s: invalid kind %q", name, vendor.Kind)
		}

		entries = append(entries, VendorEntry{Name: name, Vendor: vendor})
	}

	return entries, nil
}

func validateDestination(destination string) error {
	if destination == "" {
		return fmt.Errorf("vendors: destination key is required")
	}
	if strings.Contains(destination, "://") {
		return fmt.Errorf("vendors.%s: destination must be relative to the internal registry", destination)
	}
	if strings.HasPrefix(destination, "/") {
		return fmt.Errorf("vendors.%s: destination must not start with /", destination)
	}
	return nil
}
