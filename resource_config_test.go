package main

import (
	"testing"
	"time"
)

const basicConfig = `---
global:
  port: 8443
  interval: 15s
apps:
  enabled: true
  resources:
    - name: my-cool-app
    - name: optional-app-example
      optional: true
    - name: app-with-route
      routes:
        - app-hostname.app.cloudfoundry
    - name: optional-app-with-routes
      optional: true
      routes:
        - hostname1.first.domain
        - hostname2.first.domain
        - hostname3.second.domain
spaces:
  enabled: true
  resources:
    - name: dev
      allow_ssh: true
    - name: test
      allow_ssh: true
    - name: prod
      allow_ssh: false`

// TestAppsEnabled ensures that the 'enabled' option within the 'apps' block is set correctly.
func TestAppsEnabled(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	if conf.Data.AppConfig.Enabled != true {
		t.Fatal("Apps enabled was incorrect")
	}
}

// TestNumberOfApps ensures that the correct number of apps are found within the given config.
func TestNumberOfApps(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	apps := conf.Data.AppConfig.Apps
	if len(apps) != 4 {
		t.Fatalf("Number of apps found was incorrect. Found: %d Details: %+v", len(apps), apps)
	}
}

// TestAppNames tests the app names that are found for the given config.
func TestAppNames(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	apps := conf.Data.AppConfig.Apps

	if app0Name, expected := apps[0].Name, "my-cool-app"; app0Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, app0Name)
	}
	if app1Name, expected := apps[1].Name, "optional-app-example"; app1Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, app1Name)
	}
	if app2Name, expected := apps[2].Name, "app-with-route"; app2Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, app2Name)
	}
	if app3Name, expected := apps[3].Name, "optional-app-with-routes"; app3Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, app3Name)
	}
}

// TestOptionalApp tests the 'optional' setting within the 'resources' block of 'apps'.
func TestOptionalApp(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	apps := conf.Data.AppConfig.Apps

	if app0 := apps[0]; app0.Optional != false {
		t.Fatalf("%s optional incorrect", app0.Name)
	}
	if app1 := apps[1]; app1.Optional != true {
		t.Fatalf("%s optional incorrect", app1.Name)
	}
	if app2 := apps[2]; app2.Optional != false {
		t.Fatalf("%s optional incorrect", app2.Name)
	}
	if app3 := apps[3]; app3.Optional != true {
		t.Fatalf("%s optional incorrect", app3.Name)
	}
}

// TestNumberOfAppRoutes tests that the correct number of routes are found for the given config.
func TestNumberOfAppRoutes(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	apps := conf.Data.AppConfig.Apps

	// Validate route lengths for each app
	if app1, app1Routes := apps[0].Name, apps[0].Routes; len(app1Routes) != 0 {
		t.Fatalf("Incorrect number of routes for %s. Details: %+v", app1, app1Routes)
	}
	if app2, app2Routes := apps[1].Name, apps[1].Routes; len(app2Routes) != 0 {
		t.Fatalf("Incorrect number of routes for %s. Details: %+v", app2, app2Routes)
	}
	if app3, app3Routes := apps[2].Name, apps[2].Routes; len(app3Routes) != 1 {
		t.Fatalf("Incorrect number of routes for %s. Details: %+v", app3, app3Routes)
	}
	if app4, app4Routes := apps[3].Name, apps[3].Routes; len(app4Routes) != 3 {
		t.Fatalf("Incorrect number of routes for %s. Details: %+v", app4, app4Routes)
	}
}

// TestAppRoutes tests that the correct route (hostname+domain) are found for the given config.
func TestAppRoutes(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	apps := conf.Data.AppConfig.Apps

	if apps[2].Routes[0] != "app-hostname.app.cloudfoundry" {
		t.Fatalf("Incorrect route for app %s, found %s", apps[2].Name, apps[2].Routes[0])
	}
	if apps[3].Routes[0] != "hostname1.first.domain" {
		t.Fatalf("Incorrect route1 for app %s, found %s", apps[4].Name, apps[4].Routes[0])
	}
	if apps[3].Routes[1] != "hostname2.first.domain" {
		t.Fatalf("Incorrect route2 for app %s, found %s", apps[4].Name, apps[4].Routes[1])
	}
	if apps[3].Routes[2] != "hostname3.second.domain" {
		t.Fatalf("Incorrect route3 for app %s, found %s", apps[4].Name, apps[4].Routes[1])
	}
}

// TestRouteHost tests that the RouteEntry.Host() method pulls the correct hostname from the app routes.
func TestRouteHost(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))
	apps := conf.Data.AppConfig.Apps
	app3, app4 := apps[2], apps[3]

	if host := app3.Routes[0].Host(); host != "app-hostname" {
		t.Fatalf("%s routes[0].Host incorrect. Found: %+v", app3.Name, host)
	}
	if host := app4.Routes[0].Host(); host != "hostname1" {
		t.Fatalf("%s routes[0].Host incorrect. Found: %+v", app4.Name, host)
	}
	if host := app4.Routes[1].Host(); host != "hostname2" {
		t.Fatalf("%s routes[1].Host incorrect. Found: %+v", app4.Name, host)
	}
	if host := app4.Routes[2].Host(); host != "hostname3" {
		t.Fatalf("%s routes[2].Host incorrect. Found: %+v", app4.Name, host)
	}
}

// TestRouteDomain tests that the RouteEntry.Domain() method pulls the correct domain from the app routes.
func TestRouteDomain(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))
	apps := conf.Data.AppConfig.Apps
	app3, app4 := apps[2], apps[3]

	if domain := app3.Routes[0].Domain(); domain != "app.cloudfoundry" {
		t.Fatalf("%s routes[0].Domain incorrect. Found: %+v", app3.Name, domain)
	}
	if domain := app4.Routes[0].Domain(); domain != "first.domain" {
		t.Fatalf("%s routes[0].Domain incorrect. Found: %+v", app4.Name, domain)
	}
	if domain := app4.Routes[1].Domain(); domain != "first.domain" {
		t.Fatalf("%s routes[1].Domain incorrect. Found: %+v", app4.Name, domain)
	}
	if domain := app4.Routes[2].Domain(); domain != "second.domain" {
		t.Fatalf("%s routes[2].Domain incorrect. Found: %+v", app4.Name, domain)
	}
}

// TestConfigEnvVar tests that environment variables within the given config resolve correctly.
func TestConfigEnvVar(t *testing.T) {
	confData := `---
apps:
  enabled: true
  resources:
    - name: ${TEST_APP_1_NAME}
    - name: $TEST_APP_2_NAME
      optional: $TEST_APP_2_OPTIONAL`

	t.Setenv("TEST_APP_1_NAME", "my-cool-app")
	t.Setenv("TEST_APP_2_NAME", "another-app")
	t.Setenv("TEST_APP_2_OPTIONAL", "true")

	conf := LoadResourceConfig([]byte(confData))

	apps := conf.Data.AppConfig.Apps
	if len(apps) != 2 {
		t.Fatalf("Number of apps found was incorrect. Found: %d Details: %+v", len(apps), apps)
	}

	app1 := conf.Data.AppConfig.Apps[0]
	if app1.Name != "my-cool-app" {
		t.Fatalf("Incorrect app1 name when substituting env vars. Found: %s", app1.Name)
	}
	app2 := conf.Data.AppConfig.Apps[1]
	if app2.Name != "another-app" {
		t.Fatalf("Incorrect app2 name when substituting env vars. Found: %s", app2.Name)
	}
	if app2.Optional != true {
		t.Fatal("Incorrect app2 optional when substituting env vars")
	}
}

// TestSpaceNames tests that the correct space names are found with the given config.
func TestSpaceNames(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	spaces := conf.Data.SpaceConfig.Spaces
	if len(spaces) != 3 {
		t.Fatalf("Number of spaces found was incorrect. Found: %d Details: %+v", len(spaces), spaces)
	}

	if space0Name, expected := spaces[0].Name, "dev"; space0Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, space0Name)
	}
	if space1Name, expected := spaces[1].Name, "test"; space1Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, space1Name)
	}
	if space2Name, expected := spaces[2].Name, "prod"; space2Name != expected {
		t.Fatalf("%s name incorrect. Found: %s", expected, space2Name)
	}
}

// TestSpaceSSH tests that the correct values for allow_ssh are found for the given config.
func TestSpaceSSH(t *testing.T) {
	conf := LoadResourceConfig([]byte(basicConfig))

	spaces := conf.Data.SpaceConfig.Spaces
	if len(spaces) != 3 {
		t.Fatalf("Number of spaces found was incorrect. Found: %d Details: %+v", len(spaces), spaces)
	}

	if space0, expected := spaces[0], true; space0.AllowSSH != expected {
		t.Fatalf("Space %s allowssh incorrect", space0.Name)
	}
	if space1, expected := spaces[1], true; space1.AllowSSH != expected {
		t.Fatalf("Space %s allowssh incorrect", space1.Name)
	}
	if space2, expected := spaces[2], false; space2.AllowSSH != expected {
		t.Fatalf("Space %s allowssh incorrect", space2.Name)
	}
}

// TestGlobalPort tests that the value of 'port' is set correctly within 'global'
func TestGlobalPort(t *testing.T) {
	// Default config
	conf := LoadResourceConfig([]byte(basicConfig))
	port := conf.Data.GlobalConfig.HTTPBindPort
	if port != 8443 {
		t.Fatalf("Port was not read correctly from config. Found: %v", port)
	}

	// Custom 8080
	confData := `---
global:
  port: 8080`

	conf = LoadResourceConfig([]byte(confData))
	port = conf.Data.GlobalConfig.HTTPBindPort
	if port != 8080 {
		t.Fatalf("Port was not read correctly from config. Found: %v", port)
	}

	// No value specified
	confData = `---
global:`

	conf = LoadResourceConfig([]byte(confData))
	port = conf.Data.GlobalConfig.HTTPBindPort
	if port != 0 {
		t.Fatalf("Port was not read correctly from config. Found: %v", port)
	}
}

// TestGlobalInterval tests that the value of 'interval' is set correctly within 'global'
func TestGlobalInterval(t *testing.T) {
	// Default config
	conf := LoadResourceConfig([]byte(basicConfig))
	interval := conf.Data.GlobalConfig.RefreshInterval
	if interval != time.Second*15 {
		t.Fatalf("Interval was not read correctly from config. Found: %v", interval)
	}

	// Custom 2h interval
	confData := `---
global:
  interval: 2h`

	conf = LoadResourceConfig([]byte(confData))
	interval = conf.Data.GlobalConfig.RefreshInterval
	if interval != time.Hour*2 {
		t.Fatalf("Interval was not read correctly from config. Found: %v", interval)
	}

	// No value specified
	confData = `---
global:`

	conf = LoadResourceConfig([]byte(confData))
	interval = conf.Data.GlobalConfig.RefreshInterval
	if interval != 0 {
		t.Fatalf("Interval was not read correctly from config. Found: %v", interval)
	}
}
