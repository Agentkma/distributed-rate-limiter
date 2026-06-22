package ratelimiter

import (
	"strings"
	"testing"
)

func TestCurrentMinuteBucketFormat(t *testing.T) {
	bucket := currentMinuteBucket()
	if len(bucket) != 12 {
		t.Fatalf("expected 12-char bucket format YYYYMMDDHHMM, got %q", bucket)
	}
	for _, char := range bucket {
		if char < '0' || char > '9' {
			t.Fatalf("expected numeric bucket, got %q", bucket)
		}
	}
}

func TestBuildRateLimitKey(t *testing.T) {
	key := buildRateLimitKey("127.0.0.1")
	if !strings.HasPrefix(key, "rate:127.0.0.1:") {
		t.Fatalf("expected key prefix %q, got %q", "rate:127.0.0.1:", key)
	}
}

func TestIsFirstRequestInWindow(t *testing.T) {
	if !isFirstRequestInWindow(1) {
		t.Fatal("expected count=1 to be first request in window")
	}
	if isFirstRequestInWindow(2) {
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
