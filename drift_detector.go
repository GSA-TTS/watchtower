package main

import (
	"log"
	"net/url"
	"os"
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
	config WatchtowerConfig
}

// NewDetector starts and returns a new default Detector
func NewDetector(configFile *string) Detector {
	log.Printf("Config file path: %s", *configFile)
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Unable to read config file. %s", err)
	}
	resourceConfig := LoadResourceConfig(data)

	// If secret values are ever added to config, they should be masked in configString.
	configString = string(data)
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
	log.Printf("Starting Detector with refresh interval: %ds", int64(interval.Seconds()))
	for range ticker.C {
		detector.Validate()
	}
}

// Validate queries the CF API and validates responses against the Watchtower config.
// Results of (non-)compliance are exported as prometheus metrics via the /metrics endpoint.
func (detector *Detector) Validate() {
	// Parallelize calls to validateX using goroutines and a sync.WaitGroup
	var waitgroup sync.WaitGroup

	validationFunctions := []func(*sync.WaitGroup){
		detector.validateApps,
		detector.validateSpaces,
	}

	waitgroup.Add(len(validationFunctions))

	for _, function := range validationFunctions {
		go function(&waitgroup)
	}

	waitgroup.Wait()
}

// ValidateApps performs CF App resource validation
func (detector *Detector) validateApps(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.config.AppConfig.Enabled {
		return
	}

	deployedApps, err := detector.client.ListV3AppsByQuery(url.Values{})
	if err != nil {
		log.Printf("ERROR in ListApps() request: %s. Skipping check.", err)
		failedAppChecks.Inc()
		return
	}
	deployedAppEntries := toAppEntries(deployedApps)
	unknownApps := appDifference(deployedAppEntries, detector.config.AppConfig.Apps)
	missingApps := appDifference(detector.config.AppConfig.Apps, deployedAppEntries)

	log.Printf("Unknown Apps Detected: %s", unknownApps)
	log.Printf("Missing Apps Detected: %s", missingApps)
	totalUnknownApps.Set(float64(len(unknownApps)))
	totalMissingApps.Set(float64(len(missingApps)))
	successfulAppChecks.Inc()
}

// validateSpaces verifies spaces that Watchtower has read access to against
// the provided config. If watchtower does not have permissions to a space, it
// will be skipped.
func (detector *Detector) validateSpaces(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.config.SpaceConfig.Enabled {
		return
	}

	visibleSpaces, err := detector.client.ListSpacesByQuery(url.Values{})
	if err != nil {
		log.Printf("ERROR in ListSpaces request: %s. Skipping check.", err)
		failedSpaceChecks.Inc()
		return
	}

	for _, space := range visibleSpaces {
		for _, spaceEntry := range detector.config.SpaceConfig.Spaces {
			if space.Name == spaceEntry.Name && space.AllowSSH != spaceEntry.AllowSSH {
				log.Printf("Misconfigured SSH access detected for space: %s. SSH access enabled: %v", space.Name, space.AllowSSH)
				totalSpaceSSHViolations.Inc()
			}
		}
	}
	successfulSpaceChecks.Inc()
}

// App difference returns a slice of unknown app names.
// Given two AppEntry slices, returns the set difference of the two
// slices (a - b). This can be logically used as follows:
// unknownApps = (the set of deployed apps) - (the set of valid apps) OR
// missingApps = (the set of valid apps) - (the set of deployed apps)
func appDifference(a, b []AppEntry) (diff []string) {
	appMap := make(map[string]bool, len(b))

	for _, elem := range b {
		appMap[elem.Name] = true
	}

	for _, elem := range a {
		if _, ok := appMap[elem.Name]; !ok {
			diff = append(diff, elem.Name)
		}
	}
	return
}

// toAppEntries converts a slice of cfclient.V3App to a slice of AppEntry
func toAppEntries(v3Apps []cfclient.V3App) (entries []AppEntry) {
	for _, app := range v3Apps {
		entries = append(entries, AppEntry{Name: app.Name})
	}
	return
}
