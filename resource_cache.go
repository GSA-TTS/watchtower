package main

import (
	"errors"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

var client = NewCFClient()
var clientCreatedAt = time.Now()
var clientAgeLimitHours = 8.0

// CFResourceCache will contain the most recently scraped resource information
// about the Cloud Foundry environment being monitored. Various resource types
// can be searched for by their unique identifiers using provided lookup functions.
type CFResourceCache struct {
	Apps          AppCache
	Routes        RouteCache
	RouteMappings RouteMappingCache
	Domains       DomainCache
	SharedDomains SharedDomainCache
	Spaces        SpaceCache
}

// NewCFResourceCache returns a new, populated CFResourceCache
func NewCFResourceCache() CFResourceCache {
	var cache = CFResourceCache{}
	cache.Refresh()
	return cache
}

// Refresh the current resource cache
func (cache *CFResourceCache) Refresh() {
	// Ensure the client is still valid (refresh token expires periodically)
	if time.Since(clientCreatedAt).Hours() > clientAgeLimitHours {
		client = NewCFClient()
		log.Println("Successfully refreshed CF HTTP Client")
	}
	// Parallelize calls to refreshXCache using goroutines and a sync.WaitGroup
	var waitgroup sync.WaitGroup
	var numRefreshFuncions = 6
	waitgroup.Add(numRefreshFuncions)

	go cache.Apps.refresh(&waitgroup)
	go cache.Routes.refresh(&waitgroup)
	go cache.RouteMappings.refresh(&waitgroup)
	go cache.Domains.refresh(&waitgroup)
	go cache.SharedDomains.refresh(&waitgroup)
	go cache.Spaces.refresh(&waitgroup)

	waitgroup.Wait()
}

// isValid() returns 'true' if all sub-caches are valid, and 'false' otherwise
func (cache *CFResourceCache) isValid() bool {
	return cache.Apps.Valid &&
		cache.Routes.Valid &&
		cache.RouteMappings.Valid &&
		cache.Domains.Valid &&
		cache.SharedDomains.Valid &&
		cache.Spaces.Valid
}

// findRouteByURL returns a CF Route based on the Host+Domain, abstracting away the CF concept of shared vs private domains.
func (cache *CFResourceCache) findRouteByURL(host, domain string) (cfclient.Route, bool) {
	for _, route := range cache.Routes.routes {
		if route.Host == host {
			cfSharedDomain, ok1 := cache.SharedDomains.guidMap[route.DomainGuid]
			cfPrivateDomain, ok2 := cache.Domains.guidMap[route.DomainGuid]
			if !ok1 && !ok2 {
				log.Printf("Domain lookup failed for GUID: %s", route.DomainGuid)
				continue
			}
			if cfSharedDomain.Name == domain || cfPrivateDomain.Name == domain {
				return route, true
			}
		}
	}

	// The route with the specified URL could not be found
	return cfclient.Route{}, false
}

func (cache *CFResourceCache) findDomainNameByGUID(guid string) (string, bool) {
	if domain, ok := cache.SharedDomains.guidMap[guid]; ok {
		return domain.Name, true
	}
	if domain, ok := cache.Domains.guidMap[guid]; ok {
		return domain.Name, true
	}
	return "", false
}

// getMappingResources returns the app, route, and domain name associated with the given route mapping GUID.
func (cache *CFResourceCache) getMappingResources(mappingGUID string) (cfclient.V3App, cfclient.Route, string, error) {
	routeMapping, ok := cache.RouteMappings.guidMap[mappingGUID]
	if !ok {
		var errString = "RouteMapping with GUID " + mappingGUID + " not found in cache"
		return cfclient.V3App{}, cfclient.Route{}, "", errors.New(errString)
	}
	route, ok := cache.Routes.guidMap[routeMapping.RouteGUID]
	if !ok {
		var errString = "Route with GUID " + routeMapping.RouteGUID + " not found in cache"
		return cfclient.V3App{}, cfclient.Route{}, "", errors.New(errString)
	}

	domainName, ok := cache.findDomainNameByGUID(route.DomainGuid)
	if !ok {
		var errString = "Domain with GUID " + route.DomainGuid + " not found in cache"
		return cfclient.V3App{}, cfclient.Route{}, "", errors.New(errString)
	}

	app, ok := cache.Apps.guidMap[routeMapping.AppGUID]
	if !ok {
		var errString = "App with GUID " + routeMapping.AppGUID + " not found in cache"
		return cfclient.V3App{}, cfclient.Route{}, "", errors.New(errString)
	}

	return app, route, domainName, nil
}

// AppCache holds the most recently scraped CF App information
type AppCache struct {
	// AppCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid   bool
	apps    []cfclient.V3App
	guidMap map[string]cfclient.V3App
	nameMap map[string]cfclient.V3App
}

func (cache *AppCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the app data from cloud.gov
	resourceList, err := client.ListV3AppsByQuery(url.Values{})
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF Apps: %s", err)
		return
	}

	// Convert the app data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.V3App)
	nameMap := make(map[string]cfclient.V3App)

	for _, elem := range resourceList {
		nameMap[elem.Name] = elem
		guidMap[elem.GUID] = elem
	}

	cache.apps = resourceList
	cache.guidMap = guidMap
	cache.nameMap = nameMap
	cache.Valid = true
}

// RouteCache holds the most recently scraped CF Route information
type RouteCache struct {
	// RouteCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid   bool
	routes  []cfclient.Route
	guidMap map[string]cfclient.Route
}

func (cache *RouteCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the route data from cloud.gov
	resourceList, err := client.ListRoutes()
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF Routes: %s", err)
		return
	}

	// Convert the route data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.Route)

	for _, elem := range resourceList {
		guidMap[elem.Guid] = elem
	}

	cache.routes = resourceList
	cache.guidMap = guidMap
	cache.Valid = true
}

// RouteMappingCache holds the most recently scraped CF Route mapping information
type RouteMappingCache struct {
	// RouteMappingCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid         bool
	routeMappings []cfclient.RouteMapping
	guidMap       map[string]cfclient.RouteMapping
}

func (cache *RouteMappingCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the route mapping data from cloud.gov
	resourceListPtr, err := client.ListRouteMappings()
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF Route Mappings: %s", err)
		return
	}
	var resourceList []cfclient.RouteMapping
	for _, elem := range resourceListPtr {
		resourceList = append(resourceList, *elem)
	}

	// Convert the route data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.RouteMapping)

	for _, elem := range resourceList {
		guidMap[elem.Guid] = elem
	}

	cache.routeMappings = resourceList
	cache.guidMap = guidMap
	cache.Valid = true
}

// SharedDomainCache holds the most recently scraped CF SharedDomain information
type SharedDomainCache struct {
	// SharedDomainCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid   bool
	domains []cfclient.SharedDomain
	guidMap map[string]cfclient.SharedDomain
	nameMap map[string]cfclient.SharedDomain
}

func (cache *SharedDomainCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the domain data from cloud.gov
	resourceList, err := client.ListSharedDomains()
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF SharedDomains: %s", err)
		return
	}

	// Convert the domain data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.SharedDomain)
	nameMap := make(map[string]cfclient.SharedDomain)

	for _, elem := range resourceList {
		guidMap[elem.Guid] = elem
		nameMap[elem.Name] = elem
	}

	cache.domains = resourceList
	cache.guidMap = guidMap
	cache.nameMap = nameMap
	cache.Valid = true
}

// DomainCache holds the most recently scraped CF Domain information
type DomainCache struct {
	// DomainCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid   bool
	domains []cfclient.Domain
	guidMap map[string]cfclient.Domain
	nameMap map[string]cfclient.Domain
}

func (cache *DomainCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the domain data from cloud.gov
	resourceList, err := client.ListDomains()
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF Domains: %s", err)
		return
	}

	// Convert the domain data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.Domain)
	nameMap := make(map[string]cfclient.Domain)

	for _, elem := range resourceList {
		guidMap[elem.Guid] = elem
		nameMap[elem.Name] = elem
	}

	cache.domains = resourceList
	cache.guidMap = guidMap
	cache.nameMap = nameMap
	cache.Valid = true
}

// SpaceCache holds the most recently scraped CF Space information
type SpaceCache struct {
	// SpaceCache.Valid will be 'true' when the cache was successfully refreshed and 'false' if the last refresh failed.
	Valid   bool
	spaces  []cfclient.Space
	guidMap map[string]cfclient.Space
	nameMap map[string]cfclient.Space
}

func (cache *SpaceCache) refresh(wg *sync.WaitGroup) {
	defer wg.Done()

	// Retrieve the space data from cloud.gov
	resourceList, err := client.ListSpacesByQuery(url.Values{})
	if err != nil {
		cache.Valid = false
		log.Printf("Failed refreshing CF Spaces: %s", err)
		return
	}

	// Convert the space data to a map so that lookups can be performed without iterating over the data every time
	guidMap := make(map[string]cfclient.Space)
	nameMap := make(map[string]cfclient.Space)

	for _, elem := range resourceList {
		nameMap[elem.Name] = elem
		guidMap[elem.Guid] = elem
	}

	cache.spaces = resourceList
	cache.guidMap = guidMap
	cache.nameMap = nameMap
	cache.Valid = true
}
