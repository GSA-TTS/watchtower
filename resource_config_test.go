package main

import (
	"reflect"
	"testing"
)

// LoadResourceConfig tests
func TestLoadResourceConfig(t *testing.T) {
	confData := `---
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

	conf := LoadResourceConfig([]byte(confData))

	expected := ResourceConfig{
		Apps: []AppConfig{{"my-cool-app"}, {"another-app"}},
		Spaces: []SpaceConfig{
			{Name: "dev", AllowSSH: true},
			{Name: "test", AllowSSH: true},
			{Name: "prod", AllowSSH: false}},
	}
	if !reflect.DeepEqual(conf, expected) {
		t.Fatalf("ResourceConfig did not match expected value. Found: %+v", conf)
	}
}
