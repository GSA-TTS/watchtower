package main

import (
	"log"
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
	cache    CFResourceCache
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

	resourceCache := NewCFResourceCache()

	detector := Detector{
		client:   NewCFClient(),
		cache:    resourceCache,
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
		detector.cache.Refresh()
		detector.Validate()
	}
}

func (detector *Detector) enabledValidationFunctions() []func(*sync.WaitGroup) {
	validationFunctions := []func(*sync.WaitGroup){}

	if detector.config.Data.AppConfig.Enabled {
		validationFunctions = append(validationFunctions, detector.validateApps)
		validationFunctions = append(validationFunctions, detector.validateAppRoutes)
	}

	if detector.config.Data.SpaceConfig.Enabled {
		validationFunctions = append(validationFunctions, detector.validateSpaces)
	}

	return validationFunctions
}

// Validate queries the CF API and validates responses against the Watchtower config.
// Results of (non-)compliance are exported as prometheus metrics via the /metrics endpoint.
func (detector *Detector) Validate() {
	// Parallelize calls to validateX using goroutines and a sync.WaitGroup
	var waitgroup sync.WaitGroup

	validationFunctions := detector.enabledValidationFunctions()

	waitgroup.Add(len(validationFunctions))

	for _, function := range validationFunctions {
		go function(&waitgroup)
	}

	waitgroup.Wait()
}

// ValidateAppRoutes performs CF App Route resource validation
func (detector *Detector) validateAppRoutes(wg *sync.WaitGroup) {
	defer wg.Done()

	var cache = &detector.cache

	if !cache.isValid() {
		log.Println("Invalid cache detected. Skipping routes check.")
		failedRouteChecks.Inc()
		return
	}

	var missingRoutes []string
	for _, app := range detector.config.Apps {
		for _, route := range app.Routes {
			_, ok := detector.cache.findRouteByURL(route.Host(), route.Domain())
			if !ok {
				missingRoutes = append(missingRoutes, app.Name+":"+route.Host()+"."+route.Domain())
			}
		}
	}

	var unknownRoutes []string
	for _, mapping := range cache.RouteMappings.routeMappings {
		app, route, domainName, err := cache.getMappingResources(mapping.Guid)
		if err != nil {
			continue
		}

		// configApp is the AppEntry for this V3App
		configApp, ok := detector.config.Apps[app.Name]
		if !ok {
			// The app is an 'unknown' app. There is a route mapped to it, but it is not found in the config.
			continue
		}

		var routeURL = route.Host + "." + domainName
		if !configApp.ContainsRoute(routeURL) {
			unknownRoutes = append(unknownRoutes, app.Name+":"+routeURL)
		}
	}

	if len(unknownRoutes) != 0 {
		log.Printf("Unknown Routes Detected: %s", unknownRoutes)
	}
	if len(missingRoutes) != 0 {
		log.Printf("Missing Routes Detected: %s", missingRoutes)
	}
	totalUnknownRoutes.Set(float64(len(unknownRoutes)))
	totalMissingRoutes.Set(float64(len(missingRoutes)))
	successfulRouteChecks.Inc()
}

// ValidateApps performs CF App resource validation
func (detector *Detector) validateApps(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.cache.Apps.Valid {
		log.Println("Invalid app cache detected. Skipping check.")
		failedAppChecks.Inc()
		return
	}

	var unknownApps []string
	for name, _ := range detector.cache.Apps.nameMap {
		if _, ok := detector.config.Apps[name]; !ok {
			unknownApps = append(unknownApps, name)
		}
	}

	var missingApps []string
	for name, expectedApp := range detector.config.Apps {
		if _, ok := detector.cache.Apps.nameMap[name]; !ok && !expectedApp.Optional {
			missingApps = append(missingApps, name)
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

	if !detector.cache.Spaces.Valid {
		log.Println("Invalid space cache detected. Skipping check.")
		failedSpaceChecks.Inc()
		return
	}

	var spaceSSHViolations float64

	for name, space := range detector.cache.Spaces.nameMap {
		if spaceEntry, ok := detector.config.Spaces[name]; ok && space.AllowSSH != spaceEntry.AllowSSH {
			log.Printf("Misconfigured SSH access detected for space: %s. SSH access enabled: %v", name, space.AllowSSH)
			spaceSSHViolations++
		}
	}
	totalSpaceSSHViolations.Set(spaceSSHViolations)
	successfulSpaceChecks.Inc()
}
