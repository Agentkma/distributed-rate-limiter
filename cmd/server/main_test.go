package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Agentkma/distributed-rate-limiter/internal/ratelimiter"
	"github.com/redis/go-redis/v9"
)

// stubStore implements ratelimiter.Store for handler tests.
type stubStore struct {
	allow bool
}

var _ ratelimiter.Store = (*stubStore)(nil)

func (s *stubStore) Incr(_ context.Context, _ string) (int64, error) {
	if s.allow {
		return 1, nil
	}
	return 4, nil // over windowRequestLimit of 3
}

func (s *stubStore) Expire(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}

type stubRedisPinger struct {
	err error
}

func (s stubRedisPinger) Ping(_ context.Context) *redis.StatusCmd {
	if s.err != nil {
		return redis.NewStatusResult("", s.err)
	}

	return redis.NewStatusResult("PONG", nil)
}

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

func TestMakeAPIHandler(t *testing.T) {
	tests := []struct {
		name       string
		allow      bool
		wantStatus int
		wantBody   string
	}{
		{"allowed", true, http.StatusOK, "OK - served by :8001"},
		{"rate limited", false, http.StatusTooManyRequests, "Too Many Requests"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &stubStore{allow: tt.allow}
			handler := makeAPIHandler("8001", store)
			req := httptest.NewRequest("GET", "/api", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()
			handler(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want to contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestNewHTTPServer(t *testing.T) {
	cfg := serverConfig{port: "9999"}
	srv := newHTTPServer(cfg)
	if srv.Addr != ":9999" {
		t.Errorf("server.Addr = %q, want %q", srv.Addr, ":9999")
	}
}

func TestRedisStartUpCheck_FailedLogsServerAddr(t *testing.T) {
	client := stubRedisPinger{err: errors.New("redis unavailable")}

	output := captureLogOutput(t, func() {
		redisStartUpCheck(client, ":9999")
	})

	if !strings.Contains(output, "redis startup check failed on :9999:") {
		t.Fatalf("expected failed startup log with server addr, got %q", output)
	}
	if !strings.Contains(output, "continuing fail-open") {
		t.Fatalf("expected fail-open note in failed startup log, got %q", output)
	}
}

func TestRedisStartUpCheck_PassedLogsServerAddr(t *testing.T) {
	client := stubRedisPinger{}

	output := captureLogOutput(t, func() {
		redisStartUpCheck(client, ":8001")
	})

	if !strings.Contains(output, "redis startup check passed on :8001") {
		t.Fatalf("expected passed startup log with server addr, got %q", output)
	}
}

func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()

	log.SetOutput(&buffer)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	fn()

	return buffer.String()
}
