package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

var (
	version   = "dev"
	buildDate = "unknown"
	gitCommit = "unknown"
)

func main() {
	var (
		configPath  string
		showVersion bool
		showHelp    bool
	)

	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if showVersion {
		printVersion()
		os.Exit(0)
	}

	args := flag.Args()
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		if err := runServer(configPath); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case "version":
		printVersion()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`AIMA - AI Inference Managed by AI

Usage:
  aima [flags]
  aima [command]

Commands:
  serve     Start the API server (default)
  version   Show version information

Flags:
  -config string
        path to config file
  -help
        show help
  -version
        show version information

Examples:
  aima                           Start server with default config
  aima serve                     Start server with default config
  aima -config /etc/aima.toml    Start server with specified config
  aima -version                  Show version`)
}

func printVersion() {
	fmt.Printf("AIMA version %s (commit: %s, built: %s)\n", version, gitCommit, buildDate)
}

func runServer(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	initLogging(cfg)

	registry := unit.NewRegistry()

	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	gw := gateway.NewGateway(registry, gateway.WithTimeout(cfg.Gateway.RequestTimeoutD))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/execute", handleExecute(gw))
	mux.HandleFunc("/api/v2/health", handleHealth())

	server := &http.Server{
		Addr:         cfg.API.ListenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("AIMA server starting on %s", cfg.API.ListenAddr)

		var err error
		if cfg.API.TLSCert != "" && cfg.API.TLSKey != "" {
			err = server.ListenAndServeTLS(cfg.API.TLSCert, cfg.API.TLSKey)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("AIMA server stopped")
	return nil
}

func initLogging(cfg *config.Config) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(os.Stderr)

	switch cfg.Logging.Level {
	case "debug":
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile | log.Lmsgprefix)
	case "error", "warn":
		log.SetFlags(log.LstdFlags)
	}
}

func handleExecute(gw *gateway.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req gateway.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "failed to decode request body")
			return
		}

		resp := gw.Handle(r.Context(), &req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
		})
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
