package main

import (
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
	var numRefreshFuncions = 5
	waitgroup.Add(numRefreshFuncions)

	go cache.Apps.refresh(&waitgroup)
	go cache.Routes.refresh(&waitgroup)
	go cache.RouteMappings.refresh(&waitgroup)
	go cache.Domains.refresh(&waitgroup)
	go cache.Spaces.refresh(&waitgroup)

	waitgroup.Wait()
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

// LookupByGUID returns the requested V3App, and a bool indicating whether it was found in the cache.
func (cache *AppCache) LookupByGUID(guid string) (cfclient.V3App, bool) {
	elem, ok := cache.guidMap[guid]
	return elem, ok
}

// LookupByName returns the requested V3App, and a bool indicating whether it was found in the cache.
func (cache *AppCache) LookupByName(name string) (cfclient.V3App, bool) {
	elem, ok := cache.guidMap[name]
	return elem, ok
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

// LookupByGUID returns the requested Domain, and a bool indicating whether it was found in the cache.
func (cache *DomainCache) LookupByGUID(guid string) (cfclient.Domain, bool) {
	elem, ok := cache.guidMap[guid]
	return elem, ok
}

// LookupByName returns the requested Domain, and a bool indicating whether it was found in the cache.
func (cache *DomainCache) LookupByName(name string) (cfclient.Domain, bool) {
	elem, ok := cache.nameMap[name]
	return elem, ok
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

// LookupByGUID returns the requested Space, and a bool indicating whether it was found in the cache.
func (cache *SpaceCache) LookupByGUID(guid string) (cfclient.Space, bool) {
	elem, ok := cache.guidMap[guid]
	return elem, ok
}

// LookupByName returns the requested Space, and a bool indicating whether it was found in the cache.
func (cache *SpaceCache) LookupByName(name string) (cfclient.Space, bool) {
	elem, ok := cache.guidMap[name]
	return elem, ok
}