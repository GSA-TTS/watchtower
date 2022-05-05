package main

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/18F/watchtower/config"
)

// Detector is used to find drift between the deployed Cloud Foundry resources
// and those in the provided config allow list.
type Detector struct {
	cache  CFResourceCache
	config config.Config
}

// NewDetector starts and returns a new default Detector
func NewDetector(config *config.Config) Detector {
	resourceCache := NewCFResourceCache(config.Data.GlobalConfig.CloudControllerURL)

	detector := Detector{
		cache:  resourceCache,
		config: *config,
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
	interval := detector.config.Data.GlobalConfig.RefreshInterval
	ticker := time.NewTicker(interval)
	log.Printf("Starting Detector with refresh interval: %s", interval.String())
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

// getMissingRoutes will return a slice of strings representing missing routes in the form
// <app_name>:<app_hostname>.<app_domain>
func (detector *Detector) getMissingRoutes() []string {
	var missingRoutes []string
	for _, app := range detector.config.Apps {
		for _, route := range app.Routes {
			_, ok := detector.cache.findRouteByURL(route.Host(), route.Domain())
			if !ok {
				missingRoutes = append(missingRoutes, app.Name+":"+route.Host()+"."+route.Domain())
			}
		}
	}

	return missingRoutes
}

// getUnknownRoutes will return a slice of strings representing unknown routes in the form
// <app_name>:<app_hostname>.<app_domain>
func (detector *Detector) getUnknownRoutes() []string {
	var unknownRoutes []string
	for _, mapping := range detector.cache.RouteMappings.routeMappings {
		app, route, domainName, err := detector.cache.getMappingResources(mapping.Guid)
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

	return unknownRoutes
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

	missingRoutes := detector.getMissingRoutes()
	unknownRoutes := detector.getUnknownRoutes()

	if len(unknownRoutes) != 0 {
		sort.Strings(unknownRoutes)
		log.Printf("Unknown Routes Detected: %s", unknownRoutes)
	}
	if len(missingRoutes) != 0 {
		sort.Strings(missingRoutes)
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
	for name := range detector.cache.Apps.nameMap {
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
		sort.Strings(unknownApps)
		log.Printf("Unknown Apps Detected: %s", unknownApps)
	}
	if len(missingApps) != 0 {
		sort.Strings(missingApps)
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
