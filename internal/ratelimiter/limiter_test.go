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
	tests := []struct {
		name          string
		clientAddress string
		wantPrefix    string
	}{
		{"IPv4", "127.0.0.1", "rate:127.0.0.1:"},
		{"IPv6", "::1", "rate:::1:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := buildRateLimitKey(tt.clientAddress)
			if !strings.HasPrefix(key, tt.wantPrefix) {
				t.Errorf("buildRateLimitKey(%q) = %q, want prefix %q", tt.clientAddress, key, tt.wantPrefix)
			}
		})
	}
}

func TestIsFirstRequestForWindow(t *testing.T) {
	tests := []struct {
		name  string
		count int64
		want  bool
	}{
		{"first request", 1, true},
		{"second request", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFirstRequestForWindow(tt.count); got != tt.want {
				t.Errorf("isFirstRequestForWindow(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}

func TestIsWithinLimit(t *testing.T) {
	tests := []struct {
		name  string
		count int64
		want  bool
	}{
		{"first request", 1, true},
		{"at limit", windowRequestLimit, true},
		{"over limit", windowRequestLimit + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithinLimit(tt.count); got != tt.want {
				t.Errorf("isWithinLimit(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}
