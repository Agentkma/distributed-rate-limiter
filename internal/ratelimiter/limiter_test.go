package ratelimiter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockStore implements Store for testing without a real Redis connection.
type mockStore struct {
	incrCount int64
	incrErr   error
	expireErr error
}

func (m *mockStore) Incr(_ context.Context, _ string) (int64, error) {
	m.incrCount++
	return m.incrCount, m.incrErr
}

func (m *mockStore) Expire(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return m.expireErr == nil, m.expireErr
}

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

func TestAllow(t *testing.T) {
	tests := []struct {
		name      string
		initCount int64 // counter value before the request hits
		incrErr   error
		expireErr error
		want      bool
	}{
		{"first request allowed", 0, nil, nil, true},
		{"second request allowed", 1, nil, nil, true},
		{"at limit allowed", windowRequestLimit - 1, nil, nil, true},
		{"over limit denied", windowRequestLimit, nil, nil, false},
		{"incr error fail-open", 0, errors.New("redis down"), nil, true},
		{"expire error fail-open", 0, nil, errors.New("redis down"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{incrCount: tt.initCount, incrErr: tt.incrErr, expireErr: tt.expireErr}
			got := Allow(store, "127.0.0.1")
			if got != tt.want {
				t.Errorf("Allow() = %v, want %v", got, tt.want)
			}
		})
	}
}
