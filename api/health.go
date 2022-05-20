package api

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
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
func getEndpointHealth(url string) healthStatus {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed GET to healcheck url: %s with error: %v", url, err)
		return healthStatus{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Printf("Failure gathering details on failed health check request: %v", err)
			return healthStatus{
				StatusCode: http.StatusInternalServerError,
				Message:    "Unhealthy",
			}
		}
		log.Printf("Health check failure: %v", dump)
		return healthStatus{
			StatusCode: http.StatusInternalServerError,
			Message:    "Unhealthy",
		}
	}

	if err := resp.Body.Close(); err != nil {
		log.Fatalf("failed closing response body: %v", err)
	}

	return healthyStatus
}

// monitorHealth can be run as a goroutine to periodically update watchtowerHealth
func monitorHealth() {
	for range time.Tick(time.Second * 30) {
		status := getEndpointHealth("http://localhost:" + fmt.Sprint(bindPort) + "/metrics")
		if status != healthyStatus {
			watchtowerHealth.Set(status)
			continue
		}

		status = getEndpointHealth(cloudControllerInfoEndpoint)
		watchtowerHealth.Set(status)
	}
}
