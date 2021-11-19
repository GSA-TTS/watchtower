package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// ResourceConfig is the desired resource configuration to monitor with Watchtower
type ResourceConfig struct {
	Apps []App `yaml:"apps"`
}

// App represents a Cloud Foundry App resource
type App struct {
	Name string `yaml:"name"`
}

// LoadResourceConfig reads config.yaml and parses it into a ResourceConfig
func LoadResourceConfig() ResourceConfig {
	yamlData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Could not read config.yaml: %s", err)
	}

	var resourceConfig ResourceConfig
	if err = yaml.Unmarshal(yamlData, &resourceConfig); err != nil {
		log.Fatalf("Could not parse config.yaml: %s", err)
	}

	return resourceConfig
}
