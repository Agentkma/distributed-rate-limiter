package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Agentkma/distributed-rate-limiter/internal/ratelimiter"
	"github.com/Agentkma/distributed-rate-limiter/internal/redisclient"
	"github.com/redis/go-redis/v9"
)

type serverConfig struct {
	port string
}

const (
	cliPortFlagName        = "port"
	cliDefaultPort         = "8080"
	cliPortFlagDescription = "port to listen on"
)

func main() {
	config := parseServerConfig()
	server := newHTTPServer(config)
	runHTTPServer(server)
}

func parseServerConfig() serverConfig {
	// Skip the first argument (program name/binary path) when parsing CLI args
	const cliArgsStartIndex = 1
	cliArgs := os.Args[cliArgsStartIndex:]
	return parseServerConfigFromArgs(cliArgs)
}

func parseServerConfigFromArgs(args []string) serverConfig {
	flagSet := flag.NewFlagSet("server", flag.ContinueOnError)
	cliPortValue := flagSet.String(cliPortFlagName, cliDefaultPort, cliPortFlagDescription)
	// Intentionally ignore flag parse errors: controlled runner args + safe default port 
	// keep local demo startup fail-open.
	_ = flagSet.Parse(args)

	return serverConfig{port: *cliPortValue}
}

func newHTTPServer(config serverConfig) *http.Server {
	// mux is Go built in request router
	mux := http.NewServeMux()
	registerRoutes(mux, config)

	return &http.Server{
		Addr:              bindAddr(config),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func registerRoutes(mux *http.ServeMux, config serverConfig) {
	client := redisclient.GetClient()
	redisStartUpCheck(client, bindAddr(config))
	store := ratelimiter.NewStore(client)
	mux.HandleFunc("/api", makeAPIHandler(config.port, store))
}

func bindAddr(config serverConfig) string {
	return ":" + config.port
}

type redisPinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

func redisStartUpCheck(client redisPinger, serverAddr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis startup check failed on %s: %v (continuing fail-open)", serverAddr, err)
		return
	}

	log.Printf("redis startup check passed on %s", serverAddr)
}

func runHTTPServer(server *http.Server) {
	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func makeAPIHandler(port string, store ratelimiter.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientAddress := resolveClientAddress(r)
		if ratelimiter.Allow(store, clientAddress) {
			respondSuccess(w, port)
			return
		}

		respondTooManyRequests(w)
	}
}

func resolveClientAddress(r *http.Request) string {
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	if host == "" {
		return "unknown"
	}

	return host
}

func respondTooManyRequests(w http.ResponseWriter) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

func respondSuccess(w http.ResponseWriter, port string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "OK - served by :%s\n", port)
}
