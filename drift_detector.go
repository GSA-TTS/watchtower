package main

import (
	"log"
	"time"
)

// Detector is used to find drift between the deployed Cloud Foundry resources
// and those in the provided config allow list.
type Detector struct {
	viewer *ConfigViewer
	config ResourceConfig
}

// NewDetector starts and returns a new default Detector
func NewDetector() Detector {
	viewer, err := NewConfigViewer(DefaultScrapeInterval)
	if err != nil {
		log.Fatalf("Detector failed to initialize ConfigViewer: %s", err)
	}
	resourceConfig := LoadResourceConfig()
	detector := Detector{viewer, resourceConfig}
	go detector.start()
	return detector
}

func (detector *Detector) start() {
	interval := detector.viewer.RefreshInterval()

	// Offset the refresh and validation cycles by 1/2 viewer refresh interval
	detectionCycleOffset := interval / 2
	time.Sleep(detectionCycleOffset)
	ticker := time.NewTicker(interval)
	log.Printf("Starting Detector with refresh interval: %fs", interval.Seconds())
	for range ticker.C {
		detector.Validate()
	}
}

func (detector *Detector) Validate() {
	detector.validateApps()
}

func (detector *Detector) validateApps() {
	deployedApps := detector.viewer.V3Apps()
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
