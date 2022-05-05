package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const minRefreshInterval = time.Second * 10

// Config is the top-level configuration type for watchtower. The Config type
// should be the primary method of reading the expected state of a cloudfoundry
// environment.
type Config struct {
	Data   YAMLConfig
	Apps   map[string]AppEntry   // AppName -> AppEntry
	Spaces map[string]SpaceEntry // SpaceName -> SpaceEntry
}

// Config file definition begins here

// YAMLConfig represents top-level keys
type YAMLConfig struct {
	GlobalConfig GlobalConfig `yaml:"global"`
	AppConfig    AppConfig    `yaml:"apps"`
	SpaceConfig  SpaceConfig  `yaml:"spaces"`
}

// GlobalConfig represents allowed values under the 'global' key
type GlobalConfig struct {
	HTTPBindPort       uint16        `yaml:"port"`
	RefreshInterval    time.Duration `yaml:"interval"`
	CloudControllerURL string        `yaml:"cloud_controller_url"`
}

// AppConfig represents allowed values under the 'apps' key
type AppConfig struct {
	Enabled bool       `yaml:"enabled"`
	Apps    []AppEntry `yaml:"resources"`
}

// AppEntry represents allowed values under the 'apps:resources' key
type AppEntry struct {
	Name     string       `yaml:"name"`
	Optional bool         `yaml:"optional"`
	Routes   []RouteEntry `yaml:"routes"`
}

// ContainsRoute returns true if the AppEntry contains the specified route, false otherwise
func (a *AppEntry) ContainsRoute(route string) bool {
	for _, routeEntry := range a.Routes {
		if string(routeEntry) == route {
			return true
		}
	}
	return false
}

// SpaceConfig represents the Watchtower 'spaces' config file section.
type SpaceConfig struct {
	Enabled bool         `yaml:"enabled"`
	Spaces  []SpaceEntry `yaml:"resources"`
}

// SpaceEntry represents allowed values under the 'spaces:resources' key
type SpaceEntry struct {
	Name     string `yaml:"name"`
	AllowSSH bool   `yaml:"allow_ssh"`
}

// RouteEntry represents the allowed values for each entry under 'routes' within 'apps'
type RouteEntry string

const cFMaxRouteTokens = 2

// Host extracts the hostname from the given Route
func (r *RouteEntry) Host() string {
	return strings.SplitN(string(*r), ".", cFMaxRouteTokens)[0]
}

// Domain extracts the domain from the given Route
func (r *RouteEntry) Domain() string {
	return strings.SplitN(string(*r), ".", cFMaxRouteTokens)[1]
}

// loadData reads a []byte and parses it into a Config.
func loadData(dataSource []byte) (Config, error) {
	if dataSource == nil {
		return Config{}, errors.New("Cannot load nil config data")
	}

	// Support environent variables in the config file.
	expandedString := os.ExpandEnv(string(dataSource))
	dataSource = []byte(expandedString)

	var yamlConfig YAMLConfig
	if err := yaml.UnmarshalStrict(dataSource, &yamlConfig); err != nil {
		return Config{}, err
	}

	if yamlConfig.GlobalConfig.HTTPBindPort == 0 {
		return Config{}, errors.New("port 0 is reserved and cannot be used")
	}
	if yamlConfig.GlobalConfig.RefreshInterval < minRefreshInterval {
		return Config{}, errors.New("Refresh interval cannot be less than " + minRefreshInterval.String())
	}
	if _, err := url.ParseRequestURI(yamlConfig.GlobalConfig.CloudControllerURL); err != nil {
		return Config{}, errors.New("cloud controller URL could not be parsed")
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

	return conf, nil
}

// Load reads the named file and returns a Config.
func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	return loadData(data)
}
