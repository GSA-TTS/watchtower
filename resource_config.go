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

// YAMLConfig is the desired resource configuration to monitor with Watchtower
type YAMLConfig struct {
	AppConfig   AppConfig   `yaml:"apps"`
	SpaceConfig SpaceConfig `yaml:"spaces"`
}

// AppConfig represents the Watchtower app_config config file section.
type AppConfig struct {
	Enabled      bool       `yaml:"enabled"`
	Apps         []AppEntry `yaml:"cf_apps"`
	optionalApps map[string]AppEntry
}

// AppEntry represents the Watchtower
type AppEntry struct {
	Name     string `yaml:"name"`
	Optional bool   `yaml:"optional"`
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
