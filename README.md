![Watchtower logo](.github/watchtower-blue-white.jpeg)

# Watchtower

[![Go Report Card](https://goreportcard.com/badge/github.com/18f/watchtower)](https://goreportcard.com/report/github.com/18f/watchtower)
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
* Detect SSH access misconfigurations

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
| `-interval` | The interval (in seconds) that Watchtower will run validation checks and update exported metrics |

### Environment Variables
The following environment variables are required for watchtower to interact with
Cloud Foundry:

| Environment variable name | Description |
| --- | --- |
| `CF_API` | The full URL of the Cloud Foundry API that Watchtower should interact with. Using the [CF CLI](https://docs.cloudfoundry.org/cf-cli/install-go-cli.html), this value can be found with `cf api`. |
| `CF_USER` | The username of the Cloud Foundry User account for Watchtower to authenticate with. |
| `CF_PASS` | The password of the Cloud Foundry User account for Watchtower to authenticate with. |
| `PORT` | The port for watchtower to listen on. Default is 8080. |

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

## Watchtower Config
Generic placeholder definitions:
* `<boolean>`: a boolean that can take the values `true` or `false`
* `<string>`: a regular string
* `<secret>`: a regular string that is a secret, such as a password

### Environment Variable Expansion
Watchtower will replace ${var} or $var in the provided config according to the
values of the current environment variables. References to undefined variables
are replaced by the empty string.

### Global
```yaml
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

### Health Checks
When using Watchtower's `/health` endpoint to check the health of a node, make sure the timeout for the health-check is
set to at least 3 seconds. Why? Watchtower makes a `Get` request to the cloud controller API during health checks, so setting
the timeout for the Watchtower health-check too low may cause it to timeout before Watchtower has a response from the cloud controller.

## Exported Application Metrics
The following table includes all application-specific prometheus metrics that are exported

| Metric | Type | Description |
| --- | --- | --- |
| `watchtower_app_checks_failed_total` | Counter | Number of times the config check for V3Apps has failed for any reason |
| `watchtower_app_checks_success_total` | Counter | Number of times the config check for V3Apps has succeeded |
| `watchtower_space_checks_failed_total` | Counter | Number of times the config check for Spaces has failed for any reason |
| `watchtower_space_checks_success_total` | Counter | Number of times the config check for Spaces has succeeded |
| `watchtower_unknown_apps_total` | Gauge | Number of Apps deployed that are not in the allowed config file |
| `watchtower_missing_apps_total` | Gauge | Number of Apps in the allowed config file that are not deployed to Cloud Foundry |
| `watchtower_ssh_space_misconfiguration_total` | Gauge | Number of spaces visible to Watchtower with SSH access that differs from the value in the config |
| `watchtower_unknown_app_route_total` | Gauge | Number of App Routes visible to Watchtower that differ from the values in the config |
| `watchtower_missing_app_route_total` | Gauge | Number of App Routes in the allowed config file that are not deployed to Cloud Foundry |
