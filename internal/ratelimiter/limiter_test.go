package ratelimiter

import (
	"strings"
	"testing"
)

func TestCurrentMinuteWindowFormat(t *testing.T) {
	window := currentMinuteWindow()
	if len(window) != 12 {
		t.Fatalf("expected 12-char window format YYYYMMDDHHMM, got %q", window)
	}
	for _, char := range window {
		if char < '0' || char > '9' {
			t.Fatalf("expected numeric window, got %q", window)
		}
	}
}

func TestBuildRateLimitKey(t *testing.T) {
	key := buildRateLimitKey("127.0.0.1")
	if !strings.HasPrefix(key, "rate:127.0.0.1:") {
		t.Fatalf("expected key prefix %q, got %q", "rate:127.0.0.1:", key)
	}
}

func TestIsFirstRequestForWindow(t *testing.T) {
	if !isFirstRequestForWindow(1) {
		t.Fatal("expected count=1 to be first request in window")
	}
	if isFirstRequestForWindow(2) {
		t.Fatal("expected count=2 to not be first request in window")
	}
}

func TestIsWithinLimit(t *testing.T) {
	if !isWithinLimit(1) {
		t.Fatal("expected count=1 to be within limit")
	}
	if !isWithinLimit(maxRequestsPerWindow) {
		t.Fatalf("expected count=%d to be within limit", maxRequestsPerWindow)
	}
	if isWithinLimit(maxRequestsPerWindow + 1) {
		t.Fatalf("expected count=%d to exceed limit", maxRequestsPerWindow+1)
	}
}
