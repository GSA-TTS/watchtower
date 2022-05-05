package main

import (
	"os"
	"testing"
)

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
