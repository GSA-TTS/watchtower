package main

import (
	"flag"
	"fmt"

	"github.com/18F/watchtower/api"
	"github.com/18F/watchtower/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const namespace = "watchtower"

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
	failedAppSSHChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_ssh_checks",
		Name:      "failed_total",
		Help:      "Number of times the config refresh for Routes has failed for any reason",
	})
	successfulAppSSHChecks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "app_ssh_checks",
		Name:      "success_total",
		Help:      "Number of times the config refresh for Routes has succeeded",
	})

	// Gauges for unknown/missing/misconfigured resources
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

	totalSpaceSSHViolations = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "ssh",
		Name:      "space_misconfiguration_total",
		Help:      "Number of Spaces that have misconfigured SSH access settings",
	})
	totalAppSSHViolations = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "ssh",
		Name:      "app_misconfiguration_total",
		Help:      "Number of Apps that have misconfigured SSH access settings",
	})
)

func main() {
	zaplogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	logger := zaplogger.Sugar().Named("main")
	defer func() {
		err := logger.Sync()
		if err != nil {
			println("logger failed to flush buffered log entries. some logs may have been lost.")
			fmt.Printf("%s", err)
		}
	}()

	help := flag.Bool("help", false, "Print usage instructions.")
	configPath := flag.String("config", "config.yaml", "Path to configuration file.")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}

	config, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalw("failed configuration loading", "error", err.Error())
	}

	_, err = NewDetector(&config, logger)
	if err != nil {
		logger.Fatalw("failed creating drift detector", "error", err.Error())
	}

	err = api.Serve(&config, logger)
	if err != nil {
		logger.Fatalw("failed serving api", "error", err.Error())
	}
}
