package main

import (
	"math"
	"os"
	"strconv"
	"testing"
)

// Panic helper. For tests that expect a panic, this function can be used in a
// 'defer' to ensure a called function panic'ed as expected.
func testPanic(t *testing.T) {
	if r := recover(); r == nil {
		t.Errorf("The code did not panic")
	}
}

// getEnv tests
func TestGetEnvKeyExists(t *testing.T) {
	envKey := "SOME_KEY"
	expected := "expectedValue"
	t.Setenv(envKey, expected)
	someVal := getEnv(envKey, "defaultValue")
	if someVal != expected {
		t.Fatalf("Incorrect value '%s' found for %s. Expected '%s'", someVal, envKey, expected)
	}
}

func TestGetEnvKeyEmpty(t *testing.T) {
	envKey := "EMPTY_KEY"
	expected := "expectedValue"
	t.Setenv(envKey, "")
	someVal := getEnv(envKey, expected)
	if someVal != expected {
		t.Fatalf("Incorrect value '%s' found for %s. Expected '%s'", someVal, envKey, expected)
	}
}

func TestGetEnvKeyMissing(t *testing.T) {
	envKey := "MISSING_KEY"
	expected := "expectedValue"
	original := os.Getenv(envKey)
	if err := os.Unsetenv(envKey); err != nil {
		t.Fatal("Failed to unset environment var for missing key test")
	}
	defer os.Setenv(envKey, original)
	someVal := getEnv(envKey, expected)
	if someVal != expected {
		t.Fatalf("Incorrect value '%s' found for %s. Expected '%s'", someVal, envKey, expected)
	}
}

// readAPIFromEnv tests
func TestReadAPIMissingEnv(t *testing.T) {
	defer testPanic(t)
	envKey := "CF_API"
	original := os.Getenv(envKey)
	if err := os.Unsetenv(envKey); err != nil {
		t.Fatalf("Error unsetting CF_API for missing env test: %s", err)
	}
	defer os.Setenv(envKey, original)
	readAPIFromEnv()
}

func TestReadAPIIncorrectURL(t *testing.T) {
	defer testPanic(t)

	// Ensure the call panics
	t.Setenv("CF_API", "not a url")
	apiURL := readAPIFromEnv()
	t.Logf("API url returned: %s", apiURL)
}

func TestReadAPICustomUrl(t *testing.T) {
	expected := "https://google.com" // Not a CF API, but a valid URL
	t.Setenv("CF_API", expected)
	if actual := readAPIFromEnv(); expected != actual {
		t.Fatalf("Incorrect value '%s' API. Expected: '%s'", actual, expected)
	}
}

// ReadPortFromEnv tests
func TestReadPort8000(t *testing.T) {
	expected := "8000"
	t.Setenv("PORT", expected)
	if actual := ReadPortFromEnv(); expected != actual {
		t.Fatalf("Incorrect value '%s' for port. Expected: '%s'", actual, expected)
	}
}

func TestReadPortOutsideValidPortRange(t *testing.T) {
	t.Setenv("PORT", "99999")
	if actual, expected := ReadPortFromEnv(), DefaultPort; expected != actual {
		t.Fatalf("Incorrect value '%s' for port. Expected: '%s'", actual, expected)
	}
}

func TestReadPortKnownInvalidPorts(t *testing.T) {
	t.Setenv("PORT", "0")
	if actual, expected := ReadPortFromEnv(), DefaultPort; expected != actual {
		t.Fatalf("Incorrect value '%s' for port. Expected: '%s'", actual, expected)
	}
	t.Setenv("PORT", strconv.Itoa(math.MaxUint16))
	if actual, expected := ReadPortFromEnv(), DefaultPort; expected != actual {
		t.Fatalf("Incorrect value '%s' for port. Expected: '%s'", actual, expected)
	}
}

// newCFClient tests
func TestInvalidClient(t *testing.T) {
	defer testPanic(t)
	t.Setenv("CF_API", "https://google.com")
	NewCFClient()
}
