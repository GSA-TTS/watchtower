package main

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/18F/watchtower/config"
	"go.uber.org/zap"
)

// Detector is used to find drift between the deployed Cloud Foundry resources
// and those in the provided config allow list.
type Detector struct {
	cache  CFResourceCache
	config config.Config
	logger *zap.SugaredLogger
}

// NewDetector starts and returns a new default Detector
func NewDetector(config *config.Config, logger *zap.SugaredLogger) (Detector, error) {
	if config == nil {
		return Detector{}, errors.New("detector cannot be created with nil config")
	}
	if logger == nil {
		return Detector{}, errors.New("Detector cannot be created with nil logger")
	}
	logger = logger.Named("detector")

	resourceCache, err := NewCFResourceCache(config.Data.GlobalConfig.CloudControllerURL, logger)
	if err != nil {
		logger.Error("drift detector failed to create resource cache", "error", err.Error())
		return Detector{}, err
	}
	detector := Detector{
		cache:  resourceCache,
		config: *config,
		logger: logger,
	}

	// Call .Validate() before returning the detector so that exported metrics aren't
	// evaluated at their zero-values before the .start() goroutine can can .Validate().
	// This will prevent an external monitoring system from seeing spurious resets to
	// zero after watchtower restarts.
	detector.Validate()
	go detector.start()
	return detector, nil
}

// Start the Detector, calling .Validate every DetectionInterval
func (detector *Detector) start() {
	interval := detector.config.Data.GlobalConfig.RefreshInterval
	ticker := time.NewTicker(interval)
	detector.logger.Infow("starting detector", "refresh interval", interval.String())

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
		validationFunctions = append(validationFunctions, detector.validateAppSSH)
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
//nolint:gocognit
func (detector *Detector) getMissingRoutes() []string {
	var missingRoutes []string
	for name, app := range detector.config.Apps {
		_, appExists := detector.cache.Apps.nameMap[name]
		if (app.Optional && appExists) || !app.Optional {
			for _, route := range app.Routes {
				_, ok := detector.cache.findRouteByURL(route.Host(), route.Domain())
				if !ok {
					missingRoutes = append(missingRoutes, app.Name+":"+route.Host()+"."+route.Domain())
				}
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
		detector.logger.Warn("invalid cache detected. skipping routes check.")
		failedRouteChecks.Inc()
		return
	}

	missingRoutes := detector.getMissingRoutes()
	unknownRoutes := detector.getUnknownRoutes()

	if len(unknownRoutes) != 0 {
		sort.Strings(unknownRoutes)
		detector.logger.Infow("unknown routes detected", "unknown routes", unknownRoutes)
	}
	if len(missingRoutes) != 0 {
		sort.Strings(missingRoutes)
		detector.logger.Infow("missing routes detected", "missing routes", missingRoutes)
	}
	totalUnknownRoutes.Set(float64(len(unknownRoutes)))
	totalMissingRoutes.Set(float64(len(missingRoutes)))
	successfulRouteChecks.Inc()
}

// ValidateApps performs CF App resource validation
func (detector *Detector) validateApps(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.cache.Apps.Valid {
		detector.logger.Warn("invalid app cache detected. skipping check.")
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
		detector.logger.Infow("unknown apps detected", "unknown apps", unknownApps)
	}
	if len(missingApps) != 0 {
		sort.Strings(missingApps)
		detector.logger.Infow("missing apps detected", "missing apps", missingApps)
	}
	totalUnknownApps.Set(float64(len(unknownApps)))
	totalMissingApps.Set(float64(len(missingApps)))
	successfulAppChecks.Inc()
}

func (detector *Detector) validateAppSSH(wg *sync.WaitGroup) {
	defer wg.Done()

	var appSSHViolations []string

	if !detector.cache.Apps.Valid {
		detector.logger.Warn("invalid app cache detected. skipping ssh check.")
		failedAppSSHChecks.Inc()
		return
	}

	for name, expectedApp := range detector.config.Apps {
		// only mark violations if the app was found to be deployed AND "should ssh be disabled?" == "was ssh enabled?"
		if enabled, ok := detector.cache.Apps.sshMap[name]; ok && expectedApp.SSHDisabled == enabled {
			appSSHViolations = append(appSSHViolations, name)
		}
	}

	if len(appSSHViolations) != 0 {
		sort.Strings(appSSHViolations)
		detector.logger.Infow("misconfigured app ssh detected", "apps", appSSHViolations)
	}
	totalAppSSHViolations.Set(float64(len(appSSHViolations)))
	successfulAppSSHChecks.Inc()
}

// validateSpaces verifies spaces that Watchtower has read access to against
// the provided config. If watchtower does not have permissions to a space, it
// will be skipped.
func (detector *Detector) validateSpaces(wg *sync.WaitGroup) {
	defer wg.Done()

	if !detector.cache.Spaces.Valid {
		detector.logger.Warn("invalid space cache detected. skipping check.")
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
