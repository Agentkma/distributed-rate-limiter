package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Agentkma/distributed-rate-limiter/internal/ratelimiter"
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
	cliArgs := os.Args[1:]
	return parseServerConfigFromArgs(cliArgs)
}

func parseServerConfigFromArgs(args []string) serverConfig {
	flagSet := flag.NewFlagSet("server", flag.ContinueOnError)
	cliPortValue := flagSet.String(cliPortFlagName, cliDefaultPort, cliPortFlagDescription)
	_ = flagSet.Parse(args)

	return serverConfig{port: *cliPortValue}
}

func newHTTPServer(config serverConfig) *http.Server {
	// mux is Go built in router
	mux := http.NewServeMux()
	registerRoutes(mux, config)

	return &http.Server{
		Addr:              ":" + config.port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func registerRoutes(mux *http.ServeMux, config serverConfig) {
	mux.HandleFunc("/api", makeAPIHandler(config.port))
}

func runHTTPServer(server *http.Server) {
	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func makeAPIHandler(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientAddress := resolveClientAddress(r)
		if ratelimiter.Allow(clientAddress) {
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
