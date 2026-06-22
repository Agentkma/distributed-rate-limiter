package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseServerConfigFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPort string
	}{
		{"default port", []string{}, cliDefaultPort},
		{"custom port", []string{"--port", "8001"}, "8001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parseServerConfigFromArgs(tt.args)
			if cfg.port != tt.wantPort {
				t.Errorf("parseServerConfigFromArgs(%v).port = %q, want %q", tt.args, cfg.port, tt.wantPort)
			}
		})
	}
}

func TestResolveClientAddress(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{"IPv4", "127.0.0.1:54321", "127.0.0.1"},
		{"IPv6", "[::1]:54321", "::1"},
		{"empty address", "", "unknown"},
		{"malformed address", "not-a-host-port", "not-a-host-port"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api", nil)
			req.RemoteAddr = tt.remoteAddr
			got := resolveClientAddress(req)
			if got != tt.want {
				t.Errorf("resolveClientAddress(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

func TestRespondTooManyRequests(t *testing.T) {
	rr := httptest.NewRecorder()

	respondTooManyRequests(rr)

	if rr.Code != 429 {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Too Many Requests") {
		t.Fatalf("expected response body to contain Too Many Requests, got %q", rr.Body.String())
	}
}

func TestRespondSuccess(t *testing.T) {
	rr := httptest.NewRecorder()

	respondSuccess(rr, "8001")

	if rr.Code != 200 {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("expected content type %q, got %q", "text/plain; charset=utf-8", got)
	}
	if got := rr.Body.String(); got != "OK - served by :8001\n" {
		t.Fatalf("expected body %q, got %q", "OK - served by :8001\n", got)
	}
}
