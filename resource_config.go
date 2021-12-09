package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// WatchtowerConfig is the desired resource configuration to monitor with Watchtower
type WatchtowerConfig struct {
	AppConfig   AppConfig   `yaml:"apps"`
	SpaceConfig SpaceConfig `yaml:"spaces"`
}

// AppConfig represents the Watchtower app_config config file section.
type AppConfig struct {
	Enabled bool       `yaml:"enabled"`
	Apps    []AppEntry `yaml:"cf_apps"`
}

// AppEntry represents the Watchtower
type AppEntry struct {
	Name string `yaml:"name"`
}

type SpaceConfig struct {
	Enabled bool         `yaml:"enabled"`
	Spaces  []SpaceEntry `yaml:"cf_spaces"`
}

// SpaceEntry represents a Cloud Foundry Space config entry
type SpaceEntry struct {
	Name     string `yaml:"name"`
	AllowSSH bool   `yaml:"allow_ssh"`
}

// LoadResourceConfig reads config.yaml and parses it into a ResourceConfig. If
// dataSource is nil, it will attempt to read from `config.yaml` in the current
// directory.
func LoadResourceConfig(dataSource []byte) WatchtowerConfig {
	if dataSource == nil {
		log.Printf("Reading config from config.yaml...")
		var err error
		dataSource, err = os.ReadFile("config.yaml")
		if err != nil {
			log.Fatalf("Could not read config.yaml: %s", err)
		}
	}

	// Expand env vars
	expandedString := os.ExpandEnv(string(dataSource))
	dataSource = []byte(expandedString)

	var watchtowerConfig WatchtowerConfig
	if err := yaml.Unmarshal(dataSource, &watchtowerConfig); err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	return watchtowerConfig
}
