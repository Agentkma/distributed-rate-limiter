package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseServerConfigFromArgs_DefaultPort(t *testing.T) {
	cfg := parseServerConfigFromArgs([]string{})
	if cfg.port != cliDefaultPort {
		t.Fatalf("expected default port %q, got %q", cliDefaultPort, cfg.port)
	}
}

func TestParseServerConfigFromArgs_CustomPort(t *testing.T) {
	cfg := parseServerConfigFromArgs([]string{"--port", "8001"})
	if cfg.port != "8001" {
		t.Fatalf("expected custom port %q, got %q", "8001", cfg.port)
	}
}

func TestResolveClientAddress_IPv4(t *testing.T) {
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "127.0.0.1:54321"

	got := resolveClientAddress(req)
	if got != "127.0.0.1" {
		t.Fatalf("expected %q, got %q", "127.0.0.1", got)
	}
}

func TestResolveClientAddress_IPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "[::1]:54321"

	got := resolveClientAddress(req)
	if got != "::1" {
		t.Fatalf("expected %q, got %q", "::1", got)
	}
}

func TestResolveClientAddress_EmptyAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = ""

	got := resolveClientAddress(req)
	if got != "unknown" {
		t.Fatalf("expected %q, got %q", "unknown", got)
	}
}

func TestResolveClientAddress_MalformedAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "not-a-host-port"

	got := resolveClientAddress(req)
	if got != "not-a-host-port" {
		t.Fatalf("expected malformed address to pass through, got %q", got)
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
