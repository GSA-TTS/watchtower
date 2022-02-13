package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration type for watchtower. The Config type
// should be the primary method of reading the expected state of a cloudfoundry
// environment.
type Config struct {
	Data   YAMLConfig
	Apps   map[string]AppEntry
	Spaces map[string]SpaceEntry
}

// Config file definition begins here

// Top-level keys
type YAMLConfig struct {
	AppConfig   AppConfig   `yaml:"apps"`
	SpaceConfig SpaceConfig `yaml:"spaces"`
}

// Allowed values under 'apps' (a top-level key)
type AppConfig struct {
	Enabled bool       `yaml:"enabled"`
	Apps    []AppEntry `yaml:"resources"`
}

// Allowed values under 'resources' section of 'apps'
type AppEntry struct {
	Name     string       `yaml:"name"`
	Optional bool         `yaml:"optional"`
	Routes   []RouteEntry `yaml:"routes"`
}

// SpaceConfig represents the Watchtower 'spaces' config file section.
type SpaceConfig struct {
	Enabled bool         `yaml:"enabled"`
	Spaces  []SpaceEntry `yaml:"resources"`
}

// Allowed values under 'resources' section of 'spaces'
type SpaceEntry struct {
	Name     string `yaml:"name"`
	AllowSSH bool   `yaml:"allow_ssh"`
}

// RouteEntry represents the allowed values for each entry under 'routes' within 'apps'
type RouteEntry struct {
	Host   string `yaml:"host"`
	Domain string `yaml:"domain"`
}

// LoadResourceConfig reads config.yaml and parses it into a ResourceConfig. If
// dataSource is nil, it will attempt to read from `config.yaml` in the current
// directory.
func LoadResourceConfig(dataSource []byte) Config {
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

	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(dataSource, &yamlConfig); err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	var conf Config
	conf.Data = yamlConfig
	conf.Apps = make(map[string]AppEntry)
	conf.Spaces = make(map[string]SpaceEntry)
	for _, app := range conf.Data.AppConfig.Apps {
		conf.Apps[app.Name] = app
	}

	for _, space := range conf.Data.SpaceConfig.Spaces {
		conf.Spaces[space.Name] = space
	}

	return conf
}
