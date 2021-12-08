// Package cfhelper exposes convienience functions for creating Cloud Controller
// clients as well as reading the relevant environment variables.
package main

import (
	"log"
	"math"
	"net/url"
	"os"
	"strconv"

	"github.com/cloudfoundry-community/go-cfclient"
)

// DefaultPort is the Cloud Foundry default port for application traffic
const DefaultPort = ":8080"
const base10 = 10
const portMaxBitSize = 16

// Get an environment variable value. If the key is empty or does not exist,
// return fallback.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// ReadPortFromEnv Reads the PORT environment variable, which should be in the form PORT=8080
func ReadPortFromEnv() string {
	portString := getEnv("PORT", "8080")

	// Ensure the given port is within the valid range for port numbers (0-65535, or 2^16)
	portNum, err := strconv.ParseUint(portString, base10, portMaxBitSize)
	if err != nil {
		log.Printf("Error parsing env variable PORT with value %s. Defaulting to 8080", portString)
		return DefaultPort
	}
	if portNum == 0 || portNum == math.MaxUint16 {
		log.Printf("Invalid PORT value %s. Defaulting to 8080", portString)
		return DefaultPort
	}
	return portString
}

// Get the CF API URL value found in the CF_API environment variable. Ensures
// the value can be parsed by url.ParseRequestURI(api)
func readAPIFromEnv() string {
	apiString := getEnv("CF_API", "")

	// Perform basic URL validation
	apiURL, err := url.ParseRequestURI(apiString)
	if err != nil {
		log.Panicf("Could not parse CF API URL: '%s'.", apiString)
	}
	return apiURL.String()
}

// NewCFClient creates and returns a cfclient.Client. Reads CF_API, CF_USER, and
// CF_PASS environment variables as configuration values.
func NewCFClient() *cfclient.Client {
	c := &cfclient.Config{
		ApiAddress: readAPIFromEnv(),
		Username:   getEnv("CF_USER", ""),
		Password:   getEnv("CF_PASS", ""),
	}
	client, err := cfclient.NewClient(c)
	if err != nil {
		log.Panicf("Could not create cfclient. Error: %s", err)
	} else {
		log.Println("Successfully created cfclient")
	}
	return client
}
