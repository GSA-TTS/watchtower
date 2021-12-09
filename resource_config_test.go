package main

import (
	"reflect"
	"testing"
)

const basicConfig = `---
apps:
  enabled: true
  cf_apps:
    - name: my-cool-app
    - name: another-app
spaces:
  enabled: true
  cf_spaces:
    - name: dev
      allow_ssh: true
    - name: test
      allow_ssh: true
    - name: prod
      allow_ssh: false`

var basicWatchtowerConfig = WatchtowerConfig{
	AppConfig: AppConfig{
		Enabled: true,
		Apps: []AppEntry{
			{"my-cool-app"},
			{"another-app"},
		},
	},
	SpaceConfig: SpaceConfig{
		Enabled: true,
		Spaces: []SpaceEntry{
			{Name: "dev", AllowSSH: true},
			{Name: "test", AllowSSH: true},
			{Name: "prod", AllowSSH: false},
		},
	},
}

func TestLoadResourceConfigBasicConfig(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	if !reflect.DeepEqual(conf, basicWatchtowerConfig) {
		t.Fatalf("ResourceConfig did not match expected value. Found: %+v", conf)
	}
}

func TestLoadResourceConfigEnvVar(t *testing.T) {
	confData := `---
apps:
  enabled: true
  cf_apps:
    - name: ${TEST_APP_1_NAME}
    - name: $TEST_APP_2_NAME
spaces:
  enabled: true
  cf_spaces:
    - name: dev
      allow_ssh: true
    - name: test
      allow_ssh: true
    - name: prod
      allow_ssh: false`

	t.Setenv("TEST_APP_1_NAME", "my-cool-app")
	t.Setenv("TEST_APP_2_NAME", "another-app")

	conf := LoadResourceConfig([]byte(confData))

	if !reflect.DeepEqual(conf, basicWatchtowerConfig) {
		t.Fatalf("ResourceConfig did not match expected value. Found: %+v", conf)
	}
}
