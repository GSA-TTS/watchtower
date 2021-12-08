package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// ResourceConfig is the desired resource configuration to monitor with Watchtower
type ResourceConfig struct {
	Apps   []AppConfig   `yaml:"apps"`
	Spaces []SpaceConfig `yaml:"spaces"`
}

// AppConfig represents a Cloud Foundry App config entry
type AppConfig struct {
	Name string `yaml:"name"`
}

// SpaceConfig represents a Cloud Foundry Space config entry
type SpaceConfig struct {
	Name     string `yaml:"name"`
	AllowSSH bool   `yaml:"allow_ssh"`
}

// LoadResourceConfig reads config.yaml and parses it into a ResourceConfig. If
// dataSource is nil, it will attempt to read from `config.yaml` in the current
// directory.
func LoadResourceConfig(dataSource []byte) ResourceConfig {
	if dataSource == nil {
		log.Printf("Reading config from config.yaml...")
		var err error
		dataSource, err = os.ReadFile("config.yaml")
		if err != nil {
			log.Fatalf("Could not read config.yaml: %s", err)
		}
	}

	var resourceConfig ResourceConfig
	if err := yaml.Unmarshal(dataSource, &resourceConfig); err != nil {
		log.Fatalf("Could not parse config.yaml: %s", err)
	}

	return resourceConfig
}
