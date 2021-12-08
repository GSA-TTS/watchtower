package main

import (
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

// DetectionInterval is the default, recommended scrape interval for Watchtower
const DetectionInterval = time.Minute * time.Duration(5)

// Detector is used to find drift between the deployed Cloud Foundry resources
// and those in the provided config allow list.
type Detector struct {
	client *cfclient.Client
	config ResourceConfig
}

// NewDetector starts and returns a new default Detector
func NewDetector() Detector {
	resourceConfig := LoadResourceConfig(nil)
	detector := Detector{NewCFClient(), resourceConfig}

	// Call .Validate() before returning the detector so that exported metrics aren't
	// evaluated at their zero-values before the .start() goroutine can can .Validate().
	// This will prevent an external monitoring system from seeing spurious resets to
	// zero after watchtower restarts.
	detector.Validate()
	go detector.start()
	return detector
}

// Start the Detector, calling .Validate every DetectionInterval
func (detector *Detector) start() {
	interval := DetectionInterval
	ticker := time.NewTicker(interval)
	log.Printf("Starting Detector with refresh interval: %fs", interval.Seconds())
	for range ticker.C {
		detector.Validate()
	}
}

// Validate queries the CF API and validates responses against the Watchtower config.
// Results of (non-)compliance are exported as prometheus metrics via the /metrics endpoint.
func (detector *Detector) Validate() {
	// Parallelize calls to validateX using goroutines and a sync.WaitGroup
	var waitgroup sync.WaitGroup

	waitgroup.Add(1)
	go detector.validateApps(&waitgroup)

	waitgroup.Wait()
}

// ValidateApps performs CF App resource validation
func (detector *Detector) validateApps(wg *sync.WaitGroup) {
	defer wg.Done()

	if detector.config.AppConfig.Enabled == false {
		return
	}

	deployedApps, err := detector.client.ListV3AppsByQuery(url.Values{})
	if err != nil {
		log.Printf("ERROR in app refresh: %s. Skipping check.", err)
		failedAppUpdates.Inc()
		return
	}
	unknownApps := 0
OUTER:
	for _, deployedApp := range deployedApps {
		for _, allowedApp := range detector.config.Apps {
			if allowedApp.Name == deployedApp.Name {
				continue OUTER
			}
		}
		log.Printf("Unknown App Detected: %s", deployedApp.Name)
		unknownApps++
	}
	totalUnknownApps.Set(float64(unknownApps))
}
