# Watchtower
Watchtower is a drift-detection app for Cloud Foundry. It can be run anywhere
that it will be able to reach the Cloud Controller API, meaning it doesn't
have to run as a Cloud Foundry App, although that is the most likely use-case.

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

## Exported Application Metrics
The following table includes all application-specific prometheus metrics that are exported

| Metric | Type | Description |
| --- | --- | --- |
| `watchtower_app_updates_failed_total` | Counter | Number of times the config refresh for V3Apps has failed for any reason |
| `watchtower_app_updates_success_total` | Counter | Number of times the config refresh for V3Apps has succeeded |
| `watchtower_route_updates_failed_total` | Counter | Number of times the config refresh for Routes has failed for any reason |
| `watchtower_route_updates_success_total` | Counter | Number of times the config refresh for Routes has succeeded |
| `watchtower_shared_domain_updates_failed_total` | Counter | Number of times the config refresh for Shared Domains has failed for any reason |
| `watchtower_shared_domain_success_failed_total` | Counter | Number of times the config refresh for Shared Domains has succeeded |
| `watchtower_unknown_apps_total` | Gauge | Number of Apps deployed that are not in the allowed config file |
