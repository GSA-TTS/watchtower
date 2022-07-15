![Watchtower logo](.github/watchtower-blue-white.jpeg)

# Watchtower

[![Go Report Card](https://goreportcard.com/badge/github.com/18f/watchtower)](https://goreportcard.com/report/github.com/18f/watchtower)
[![Maintainability](https://api.codeclimate.com/v1/badges/248e347811c1f1108868/maintainability)](https://codeclimate.com/github/18F/watchtower/maintainability)
![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)

Watchtower is a drift-detection app for Cloud Foundry. It can be run anywhere
that it will be able to reach the Cloud Controller API, meaning it doesn't
have to run as a Cloud Foundry App, although that is the most likely use-case.
Watchtower does what any real-life watchtower would do -- it observes an area
(in this case, a Cloud Foundry environment) and if something happens that isn't
supposed to, it will communicate that to the authorities (a Prometheus server).

## Features
* Detect unknown resources deployed to Cloud Foundry
* Detect missing resources *not* deployed to Cloud Foundry, but should be
* Detect SSH access misconfigurations for apps and spaces

### Supported Resource Types
* Apps
* Routes
* Spaces

## How it works
Watchtower reads in a `config.yaml` file that contains an allowed list of Cloud
Foundry resources. It will scrape the CF API and detect any resources that are
not in the allowed list. It is expected that Watchtower is being monitored by a
Prometheus instance, as "Unknown" resources are reflected in the exported `/metrics`
endpoint as a Prometheus Gauge. For example, if 2 apps were found that were not
in the provided `config.yaml` allow list, the `watchtower_unknown_apps_total`
Gauge would be set to `2`.

Resources are checked on an opt-in model, meaning if you
provide any app in the `config.yaml`, then all deployed apps must match the allow
list.

## Running Watchtower
Watchtower can be run from anywhere that is able to hit your cloud foundry api.
To run, either download a pre-compiled binary from the [releases](https://github.com/18F/watchtower/releases)
page, or compile the go source yourself using `go build`.

### CLI Arguments

| Argument | Description |
| --- | --- |
| `-config` | Path to the configuration file |
| `-help` | Print the Watchtower usage message. |

### Environment Variables
The following environment variables are required for watchtower to interact with
Cloud Foundry:

| Environment variable name | Description |
| --- | --- |
| `CF_USER` | The username of the Cloud Foundry User account for Watchtower to authenticate with. |
| `CF_PASS` | The password of the Cloud Foundry User account for Watchtower to authenticate with. |

### Service Account and Permissions
The user/pass provided to watchtower should be a service account with access to
space auditor permissions. For environments that span multiple spaces such as a
`dev-web` and `dev-data` that are both parts of a larger "dev" environment, the
user provided to Watchtower should have auditor permissions on all spaces that
contain resources you wish to monitor. Keep in mind that once you give auditor
permissions for a space, `list` operations for a resource will now include all
resources of that type from the included space, and thus need to be reflected
in the config file provided to Watchtower to avoid false positives due to
"unknown" resources showing up.

### Using a Forward Proxy
Running watchtower behind a forward proxy is as simple as setting the
`HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

### Deploying as a cloud.gov app

To deploy watchtower as a cloud.gov app using the [example manifest.yml file](manifest.yml), run:

`cf push --var cf_user=<SPACE_AUDITOR_USER> --var cf_pass=<SPACE_AUDITOR_PASSWORD> --var watchtower_app_name=<WATCHTOWER_APP_NAME>`

## Watchtower Config
Generic placeholder definitions:
* `<boolean>`: a boolean that can take the values `true` or `false`
* `<string>`: a regular string
* `<int>`: an integer value
* `<duration>`: a duration that can be parsed with go's [time.ParseDuration()](https://pkg.go.dev/time#ParseDuration)
* `<secret>`: a regular string that is a secret, such as a password

### Environment Variable Expansion
Watchtower will replace ${var} or $var in the provided config according to the
values of the current environment variables. References to undefined variables
are replaced by the empty string.

### Watchtower Config
```yaml
global:
  # The port that Watchtower will bind to for serving requests.
  port: <int> | default = 0

  # The interval Watchtower will use to refresh its internal view of the cloud
  # foundry environment as well as any exported metrics. This value cannot be
  # less than 10s.
  refresh_interval: <duration> | default = 5m

  # The full URL of the Cloud Foundry Cloud Controller that Watchtower should
  # interact with. Using the CF CLI, this value can be found with `cf api`.
  cloud_controller_url: <string> | default = ""
apps:
  # Whether to enable monitoring of CF Apps. Enabled=false will result in
  # app-related metrics being the zero-value of the metric type.
  [ enabled: <boolean> | default = false ]

  # List of CF Apps to monitor
  resources:
    [ - <cf_app_config> ... ]

spaces:
  # Whether to enable monitoring of CF Spaces. Enabled=false will result in
  # space-related metrics being the zero-value of the metric type.
  [ enabled: <boolean> | default = false ]

  # List of CF Spaces to monitor. Since it's not guaranteed that Watchtower has
  # access to all the spaces listed in the config, watchtower will list all
  # spaces it has access to and monitor any spaces with names matching config
  # entries found here. E.g. listing dev, test, and prod spaces here, but only
  # giving Watchtower auditor permissions on the dev space would result in
  # monitoring only the dev space.
  resources:
    [ - <cf_space_config> ... ]
```

### `<cf_app_config>`
```yaml
name: <string>
# Whether the app will be marked as "missing" if it is not observed. Apps
# marked as optional will never be marked as missing or unknown.
[optional: <bool> | default = false]

# Whether ssh should be allowed to the app or not. The default (false) expects
# that ssh is allowed to the app instance.
[ssh_disabled: <bool> | default = false]

# Watchtower considers routes to be a part of an apps definition. The routes
# section can be omitted, and will be interpreted as "app should have no routes"
routes:
  # Route strings must be of the form <hostname>.<domain> and not deviate. The
  # following would be a valid route: my-cool-app.app.cloudfoundry
  # where the hostname would be interpreted to be "my-cool-app" and the domain
  # as "app.cloudfoundry".
  [ - <string> ... ]
```

### `<cf_space_config>`
```yaml
name: <string>
allow_ssh: <boolean> | default = false
```

## Endpoints

| Endpoint | Description |
| --- | --- |
| `/metrics` | Prometheus-style metrics endpoint containing all Watchtower metrics |
| `/config` | The current Watchtower config |
| `/health` | Health monitoring endping. Non-200 response indicates an unhealthy Watchtower node |

## Exported Application Metrics
The following table includes all application-specific prometheus metrics that are exported

| Metric | Type | Description |
| --- | --- | --- |
| `watchtower_unknown_apps_total`               | Gauge | Number of Apps deployed that are not in the allowed config file (config.yaml) |
| `watchtower_missing_apps_total`               | Gauge | Number of Apps in the provided config file that are not deployed |
| `watchtower_unknown_app_routes_total`         | Gauge | Number of Routes deployed that are not in the allowed config file (config.yaml) |
| `watchtower_missing_app_routes_total`         | Gauge | Number of Routes in the provided config file that are not deployed |
| `watchtower_ssh_space_misconfiguration_total` | Gauge | Number of Spaces that have misconfigured SSH access settings |
| `watchtower_ssh_app_misconfiguration_total`   | Gauge | Number of Apps that have misconfigured SSH access settings |
| `watchtower_app_checks_failed_total`          | Counter | Number of times the config refresh for V3Apps has failed for any reason |
| `watchtower_app_checks_success_total`         | Counter | Number of times the config refresh for V3Apps has succeeded |
| `watchtower_space_checks_failed_total`        | Counter | Number of times the config check for Spaces has failed for any reason |
| `watchtower_space_checks_success_total`       | Counter | Number of times the config check for Spaces has succeeded |
| `watchtower_route_checks_failed_total`        | Counter | Number of times the config refresh for Routes has failed for any reason |
| `watchtower_route_checks_success_total`       | Counter | Number of times the config refresh for Routes has succeeded |
| `watchtower_app_ssh_checks_failed_total`      | Counter | Number of times the config refresh for Routes has failed for any reason |
| `watchtower_app_ssh_checks_success_total`     | Counter | Number of times the config refresh for Routes has succeeded |
