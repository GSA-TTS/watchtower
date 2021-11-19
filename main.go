package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "watchtower"

var (
	failedAppUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_updates",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for V3Apps has failed for any reason",
	})
	failedRouteUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "route_updates",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for Routes has failed for any reason",
	})
	failedSharedDomainsUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "shared_domain_updates",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for Shared Domains has failed for any reason",
	})

	successfulAppUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_updates",
		Name:      "success_total",
		Help:      "Number of times the config refresh for V3Apps has succeeded",
	})
	successfulRouteUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "route_updates",
		Name:      "success_total",
		Help:      "Number of times the config refresh for Routes has succeeded",
	})
	successfulSharedDomainsUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "shared_domain_updates",
		Name:      "success_total",
		Help:      "Number of times the config refresh for Shared Domains has succeeded",
	})

	totalUnknownApps = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "unknown",
		Name:      "apps_total",
		Help:      "Number of Apps deployed that are not in the allowed config file (config.yaml)",
	})
)

func main() {
	NewDetector()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+ReadPortFromEnv(), nil))
}
