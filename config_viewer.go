// Package configviewer implements the ConfigViewer, which is responsible for
// keeping a recent snapshot of deployed Cloud Foundry resources available to
// view. ConfigViewer structs do not present a real-time view, and instead
// refresh periodically.
package main

import (
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

// DefaultScrapeInterval is the default, recommended scrape interval for new ConfigViewer structs
const DefaultScrapeInterval = time.Second * time.Duration(15)

// ConfigViewer is used to hold the most recent config data scraped from the CF API. ConfigViewer
// structs will always show an unfiltered view of the contained resources. E.g. the v3Apps attribute
// will contain all apps, in any state and with any name in order to accurately reflect the environment
type ConfigViewer struct {
	client          *cfclient.Client
	refreshInterval time.Duration
	readWriteLock   sync.RWMutex
	v3Apps          []cfclient.V3App
	routes          []cfclient.Route
	sharedDomains   []cfclient.SharedDomain
	// Add new Cloud Foundry resource types here, then implement their refreshX and
	// getter methods, as well as update the Refresh() method.
}

// NewConfigViewer creates a new config viewer with the provided client and empty config values.
func NewConfigViewer(refreshInterval time.Duration) (viewer *ConfigViewer, err error) {
	newViewer := ConfigViewer{
		client:          NewCFClient(),
		refreshInterval: refreshInterval,
		// Avoid nil zero-values by using 0-length slices as initial values
		v3Apps:        make([]cfclient.V3App, 0),
		routes:        make([]cfclient.Route, 0),
		sharedDomains: make([]cfclient.SharedDomain, 0),
	}
	// Initialize the viewer fields with data so that a viewer can never be
	// evaluated before a 'list' call has been made to the CF API.
	newViewer.start()
	return &newViewer, nil
}

// Start the ConfigViewer, calling 'refresh()' every viewer.RefreshInterval
func (viewer *ConfigViewer) start() {
	ticker := time.NewTicker(viewer.refreshInterval)
	log.Printf("Starting ConfigViewer with refresh interval: %s", viewer.refreshInterval)
	go func() {
		viewer.Refresh()
		for range ticker.C {
			viewer.Refresh()
		}
	}()
	log.Println("cfScheduler creation complete")
}

// 'Getter' methods for the ConfigViewer resources. Do not return the
// underlying slice, as that would allow external modification of the
// viewers data.

// V3Apps returns a copy of the 'viewer's apps slice.
func (viewer *ConfigViewer) V3Apps() []cfclient.V3App {
	viewer.readWriteLock.RLock()
	defer viewer.readWriteLock.RUnlock()

	appList := make([]cfclient.V3App, len(viewer.v3Apps))
	copy(appList, viewer.v3Apps)
	return appList
}

// Routes returns a copy of the 'viewer's routes slice.
func (viewer *ConfigViewer) Routes() []cfclient.Route {
	viewer.readWriteLock.RLock()
	defer viewer.readWriteLock.RUnlock()

	routeList := make([]cfclient.Route, len(viewer.routes))
	copy(routeList, viewer.routes)
	return routeList
}

// SharedDomains returns a copy of the 'viewer's sharedDomains slice.
func (viewer *ConfigViewer) SharedDomains() []cfclient.SharedDomain {
	viewer.readWriteLock.RLock()
	defer viewer.readWriteLock.RUnlock()

	sharedDomainList := make([]cfclient.SharedDomain, len(viewer.sharedDomains))
	copy(sharedDomainList, viewer.sharedDomains)
	return sharedDomainList
}

func (viewer *ConfigViewer) RefreshInterval() time.Duration {
	duration := viewer.refreshInterval
	return duration
}

// Refresh methods. The Refresh() method is responsible for coordinating
// concurrent viewer refreshX() calls, while the refreshX methods are
// responsible for returning a single viewer resource via a provided channel.

// Refresh all ConfigViewer data by querying the CloudFoundry API
func (viewer *ConfigViewer) Refresh() {
	// Create buffered (1 element) channels to receive the new config items
	appsChan := make(chan []cfclient.V3App, 1)
	routesChan := make(chan []cfclient.Route, 1)
	sharedDomainsChan := make(chan []cfclient.SharedDomain, 1)

	// Start the refresh functions
	go viewer.refreshV3Apps(appsChan)
	go viewer.refreshRoutes(routesChan)
	go viewer.refreshSharedDomains(sharedDomainsChan)

	// Wait for refresh function return values
	apps := <-appsChan
	routes := <-routesChan
	sharedDomains := <-sharedDomainsChan

	// Update the ConfigViewer values
	viewer.readWriteLock.Lock()
	defer viewer.readWriteLock.Unlock()
	viewer.v3Apps = apps
	viewer.routes = routes
	viewer.sharedDomains = sharedDomains
}

// The refreshX methods will each be run as a goroutine and should each accept
// a channel that will accept a slice of the resource being refreshed

// refreshV3Apps sends the latest V3Apps to the provided channel
func (viewer *ConfigViewer) refreshV3Apps(appsChan chan<- []cfclient.V3App) {
	apps, err := viewer.client.ListV3AppsByQuery(url.Values{})
	if err != nil {
		log.Printf("ERROR in app refresh: %s. Skipping update", err)
		failedAppUpdates.Inc()
		appsChan <- viewer.V3Apps() // Send a duplicate copy of the current value as a no-op
		return
	}

	successfulAppUpdates.Inc()
	appsChan <- apps
}

// refreshRoutes sends the latest Routes to the provided channel
func (viewer *ConfigViewer) refreshRoutes(routesChan chan<- []cfclient.Route) {
	routes, err := viewer.client.ListRoutes()
	if err != nil {
		log.Printf("ERROR in routes refresh: %s. Skipping update", err)
		failedRouteUpdates.Inc()
		routesChan <- viewer.Routes() // Send a duplicate copy of the current value as a no-op
		return
	}
	successfulRouteUpdates.Inc()
	routesChan <- routes
}

// refreshSharedDomains sends the latest SharedDomains to the provided channel
func (viewer *ConfigViewer) refreshSharedDomains(sharedDomainsChan chan<- []cfclient.SharedDomain) {
	sharedDomains, err := viewer.client.ListSharedDomains()
	if err != nil {
		log.Printf("ERROR in shared domains refresh: %s. Skipping update", err)
		failedSharedDomainsUpdates.Inc()
		sharedDomainsChan <- viewer.SharedDomains() // Send a duplicate copy of the current value as a no-op
	}
	successfulSharedDomainsUpdates.Inc()
	sharedDomainsChan <- sharedDomains
}
