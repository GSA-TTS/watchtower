package main

import (
	"testing"
	"time"
)

// Test the NewConfigViewer constructor the way it is used in main.go
func TestNoChangeUnderlyingDuration(t *testing.T) {
	viewer, err := NewConfigViewer(time.Minute)
	if err != nil {
		t.FailNow()
	}
	refreshInterval := viewer.RefreshInterval()
	refreshInterval++
	if refreshInterval == viewer.RefreshInterval() {
		t.Fail()
	}
}
