package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"go.uber.org/zap"
)

// healthStatus structs capture the current health of the Watchtower app
type healthStatus struct {
	StatusCode int    `yaml:"status"`
	Message    string `yaml:"message"`
}

// health structs provide a concurrency-safe way of accessing the current healthStatus
type health struct {
	status healthStatus
	mut    sync.RWMutex
}

func (h *health) Get() healthStatus {
	h.mut.RLock()
	status := h.status
	h.mut.RUnlock()
	return status
}

func (h *health) Set(status healthStatus) {
	h.mut.Lock()
	h.status = status
	h.mut.Unlock()
}

var watchtowerHealth = health{
	status: healthStatus{
		StatusCode: http.StatusOK,
		Message:    "Healthy",
	},
	mut: sync.RWMutex{},
}

var healthyStatus = healthStatus{StatusCode: http.StatusOK, Message: "Healthy"}

// checkEndpoint makes a GET request to the requested URL and automatically sets
// watchtowerHealth should the request fail. Returns the request response.
func getEndpointHealth(url string, logger *zap.SugaredLogger) healthStatus {
	resp, err := http.Get(url)
	if err != nil {
		logger.Warnw("failed get request to healcheck endpoint", "url", url, "error", err)
		return healthStatus{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			logger.Warnw("failed gathering details on failed health check request", "error", err)
			return healthStatus{
				StatusCode: http.StatusInternalServerError,
				Message:    "Unhealthy",
			}
		}
		logger.Warnw("health check failure", "response", dump)
		return healthStatus{
			StatusCode: http.StatusInternalServerError,
			Message:    "Unhealthy",
		}
	}

	if err := resp.Body.Close(); err != nil {
		logger.Fatalw("failed closing response body", "error", err)
	}

	return healthyStatus
}

// monitorHealth can be run as a goroutine to periodically update watchtowerHealth
func monitorHealth(logger *zap.SugaredLogger) {
	const healthCheckInterval = time.Second * 30

	for range time.Tick(healthCheckInterval) {
		status := getEndpointHealth("http://localhost:"+fmt.Sprint(bindPort)+"/metrics", logger)
		if status != healthyStatus {
			watchtowerHealth.Set(status)
			continue
		}

		status = getEndpointHealth(cloudControllerInfoEndpoint, logger)
		watchtowerHealth.Set(status)
	}
}
