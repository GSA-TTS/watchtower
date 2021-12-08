package main

import (
	"reflect"
	"testing"
)

const basicConfig = `---
apps:
  - name: my-cool-app
  - name: another-app
spaces:
  - name: dev
    allow_ssh: true
  - name: test
    allow_ssh: true
  - name: prod
    allow_ssh: false`

var basicResourceConfig = ResourceConfig{
	Apps: []AppConfig{{"my-cool-app"}, {"another-app"}},
	Spaces: []SpaceConfig{
		{Name: "dev", AllowSSH: true},
		{Name: "test", AllowSSH: true},
		{Name: "prod", AllowSSH: false}},
}

func TestLoadResourceConfigBasicConfig(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	if !reflect.DeepEqual(conf, basicResourceConfig) {
		t.Fatalf("ResourceConfig did not match expected value. Found: %+v", conf)
	}
}

func TestLoadResourceConfigEnvVar(t *testing.T) {
	confData := `---
apps:
  - name: ${TEST_APP_1_NAME}
  - name: $TEST_APP_2_NAME
spaces:
  - name: dev
    allow_ssh: true
  - name: test
    allow_ssh: true
  - name: prod
    allow_ssh: false`

	t.Setenv("TEST_APP_1_NAME", "my-cool-app")
	t.Setenv("TEST_APP_2_NAME", "another-app")

	conf := LoadResourceConfig([]byte(confData))

	if !reflect.DeepEqual(conf, basicResourceConfig) {
		t.Fatalf("ResourceConfig did not match expected value. Found: %+v", conf)
	}
}
