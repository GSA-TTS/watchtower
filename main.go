package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "watchtower"

// Configuration Flags
var configPath = flag.String("config", "config.yaml", "Path to configuration file.")
var validationInterval = flag.Int("interval", int(DetectionInterval.Seconds()), "The interval (in seconds) that Watchtower will run validation checks and update exported metrics")

// Global Settings
var client = NewCFClient()
var clientCreatedAt = time.Now()
var clientAgeLimitHours = 8.0
var configString = ""
var bindPort = "8080"

var (
	// Counters for failed/successful validation checks
	failedAppChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_checks",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for V3Apps has failed for any reason",
	})
	successfulAppChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_checks",
		Name:      "success_total",
		Help:      "Number of times the config refresh for V3Apps has succeeded",
	})
	failedSpaceChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "space_checks",
		Name:      "failed_total",
		Help:      "Number of times the config check for Spaces has failed for any reason",
	})
	successfulSpaceChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "space_checks",
		Name:      "success_total",
		Help:      "Number of times the config check for Spaces has succeeded",
	})
	failedRouteChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "route_checks",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for Routes has failed for any reason",
	})
	successfulRouteChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "route_checks",
		Name:      "success_total",
		Help:      "Number of times the config refresh for Routes has succeeded",
	})

	// Counters for unknown/missing/misconfigured resources
	totalUnknownApps = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "unknown",
		Name:      "apps_total",
		Help:      "Number of Apps deployed that are not in the allowed config file (config.yaml)",
	})
	totalMissingApps = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "missing",
		Name:      "apps_total",
		Help:      "Number of Apps in the provided config file that are not deployed",
	})

	totalSpaceSSHViolations = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "ssh",
		Name:      "space_misconfiguration_total",
		Help:      "Number of Spaces that have misconfigured SSH access settings",
	})

	totalUnknownRoutes = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "unknown",
		Name:      "app_routes_total",
		Help:      "Number of Routes deployed that are not in the allowed config file (config.yaml)",
	})
	totalMissingRoutes = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "missing",
		Name:      "app_routes_total",
		Help:      "Number of Routes in the provided config file that are not deployed",
	})
)

// configHandler shows the currently loaded config file
func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, configString)
}

// healthHandler attempts to determine the health of Watchtower by checking whether the http client can
// successfully hit the CloudController API, and whether metrics are successfully being served.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Healthy"

	// Check responses for errors
	watchtowerResp, watchtowerErr := http.Get("http://localhost:" + bindPort + "/metrics")
	if watchtowerErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp["message"] = watchtowerErr.Error()
	} else if _, clientErr := client.GetInfo(); clientErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp["message"] = clientErr.Error()
	}

	// Clean up and write the response
	if watchtowerErr == nil {
		// There was no error in the call to /metrics, so the response body must be closed
		err := watchtowerResp.Body.Close()
		if err != nil {
			log.Fatalf("Error closing response body. Err: %s", err)
		}
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Fatalf("Error writing response. Err: %s", err)
	}
	return
}

func main() {
	flag.Parse()
	NewDetector(configPath, *validationInterval)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/health", healthHandler)
	bindPort = ReadPortFromEnv()
	log.Fatal(http.ListenAndServe(":"+bindPort, nil))
}
