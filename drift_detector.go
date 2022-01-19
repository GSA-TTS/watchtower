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
	client   *cfclient.Client
	config   Config
	interval int
}

// NewDetector starts and returns a new default Detector
func NewDetector(configFile *string, validationInterval int) Detector {
	log.Printf("Config file path: %s", *configFile)
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Unable to read config file. %s", err)
	}
	resourceConfig := LoadResourceConfig(data)

	// If secret values are ever added to config, they should be masked in configString.
	configString = string(data)

	detector := Detector{
		client:   NewCFClient(),
		config:   resourceConfig,
		interval: validationInterval,
	}

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
	interval := time.Second * time.Duration(detector.interval)
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

func (detector *Detector) getDeployedApps() (map[string]cfclient.V3App, error) {
	// Retrieve the app data from cloud.gov
	deployedApps, err := detector.client.ListV3AppsByQuery(url.Values{})
	if err != nil {
		return nil, err
	}

	// Convert the app data to a map so that lookups can be performed without iterating over the data every time
	deployedAppMap := make(map[string]cfclient.V3App)
	for _, app := range deployedApps {
		deployedAppMap[app.Name] = app
	}
	return deployedAppMap, nil
}

// ValidateApps performs CF App resource validation
func (detector *Detector) validateApps(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.config.Data.AppConfig.Enabled {
		return
	}

	deployedApps, err := detector.getDeployedApps()
	if err != nil {
		log.Printf("ERROR in ListApps() request: %s. Skipping check.", err)
		failedAppChecks.Inc()
		return
	}

	var unknownApps []string
	for _, deployedApp := range deployedApps {
		if _, ok := detector.config.Apps[deployedApp.Name]; !ok {
			unknownApps = append(unknownApps, deployedApp.Name)
		}
	}

	var missingApps []string
	for _, expectedApp := range detector.config.Apps {
		if _, ok := deployedApps[expectedApp.Name]; !ok && !expectedApp.Optional {
			missingApps = append(missingApps, expectedApp.Name)
		}
	}

	if len(unknownApps) != 0 {
		log.Printf("Unknown Apps Detected: %s", unknownApps)
	}
	if len(missingApps) != 0 {
		log.Printf("Missing Apps Detected: %s", missingApps)
	}
	totalUnknownApps.Set(float64(len(unknownApps)))
	totalMissingApps.Set(float64(len(missingApps)))
	successfulAppChecks.Inc()
}

// validateSpaces verifies spaces that Watchtower has read access to against
// the provided config. If watchtower does not have permissions to a space, it
// will be skipped.
func (detector *Detector) validateSpaces(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.config.Data.SpaceConfig.Enabled {
		return
	}

	visibleSpaces, err := detector.client.ListSpacesByQuery(url.Values{})
	if err != nil {
		log.Printf("ERROR in ListSpaces request: %s. Skipping check.", err)
		failedSpaceChecks.Inc()
		return
	}

	var spaceSSHViolations float64

	for _, space := range visibleSpaces {
		if spaceEntry, ok := detector.config.Spaces[space.Name]; ok && space.AllowSSH != spaceEntry.AllowSSH {
			log.Printf("Misconfigured SSH access detected for space: %s. SSH access enabled: %v", space.Name, space.AllowSSH)
			spaceSSHViolations++
		}
	}
	totalSpaceSSHViolations.Set(spaceSSHViolations)
	successfulSpaceChecks.Inc()
}
