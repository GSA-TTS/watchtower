package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/18F/watchtower/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var bindPort uint16
var cloudControllerInfoEndpoint = ""
var logger *zap.SugaredLogger

// healthHandler attempts to determine the health of Watchtower by checking whether the http client can
// successfully hit the CloudController API, and whether metrics are successfully being served.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	currentHealth := watchtowerHealth.Get()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(currentHealth.StatusCode)

	jsonResp, err := json.Marshal(currentHealth)
	if err != nil {
		logger.Errorw("JSON marshal failure during health check",
			"error", err.Error(),
		)
	}
	if _, err := w.Write(jsonResp); err != nil {
		logger.Errorw("failed writing response to /health request",
			"error", err.Error(),
		)
	}
}

func registerEndpoints(conf *config.Config) {
	// Set global api variables
	bindPort = conf.Data.GlobalConfig.HTTPBindPort
	cloudControllerInfoEndpoint = conf.Data.GlobalConfig.CloudControllerURL + "/v2/info"

	// Register Watchtower API endpoints

	http.HandleFunc("/health", healthHandler)

	yamlBytes, err := yaml.Marshal(conf.Data)
	if err != nil {
		logger.Fatalf("Failed marshalling config to yaml for /config endpoint: %v", err)
	}
	http.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(yamlBytes); err != nil {
			logger.Errorw("failed writing response to /config request",
				"error", err.Error(),
			)
		}
	})

	http.Handle("/metrics", promhttp.Handler())
}

// Serve registers the Watchtower endpoints to the http DefaultServeMux, begins
// listening for incoming connections, and monitoring health of the app.
func Serve(conf *config.Config, zapLogger *zap.SugaredLogger) error {
	if zapLogger == nil {
		return errors.New("cannot call api.Serve with nil logger")
	}

	logger = zapLogger.Named("api")
	registerEndpoints(conf)
	go monitorHealth(logger)
	logger.Infow("start listening for connections",
		"address", "0.0.0.0"+":"+fmt.Sprint(bindPort),
	)

	err := http.ListenAndServe(":"+fmt.Sprint(bindPort), nil)
	logger.Fatal(err)

	return nil
}
