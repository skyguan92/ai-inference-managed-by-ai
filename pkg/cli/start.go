package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/metrics"
	"github.com/spf13/cobra"
)

func NewStartCommand(root *RootCommand) *cobra.Command {
	var (
		port    int
		addr    string
		tlsCert string
		tlsKey  string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the HTTP server",
		Long: `Start the AIMA HTTP API server.

The server provides a REST API for executing atomic units
and managing AI inference infrastructure.`,
		Example: `  # Start server with default settings
  aima start

  # Start on a different port
  aima start --port 8080

  # Start with TLS
  aima start --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), root, addr, port, tlsCert, tlsKey)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Server port (default from config)")
	cmd.Flags().StringVar(&addr, "addr", "", "Server listen address (default from config)")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate file")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS key file")

	return cmd
}

func runStart(ctx context.Context, root *RootCommand, addr string, port int, tlsCert, tlsKey string) error {
	cfg := root.Config()
	gw := root.Gateway()

	listenAddr := cfg.API.ListenAddr
	if addr != "" {
		listenAddr = addr
	}

	reqMetrics := metrics.NewRequestMetrics()
	rawCollector := metrics.NewCollector()
	sysCollector := metrics.NewCachedCollector(rawCollector, 15*time.Second)
	sysCollector.Start(ctx)
	defer sysCollector.Stop()

	router := gateway.NewRouter(gw)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/execute", instrumentHandler(handleExecute(gw), reqMetrics))
	mux.HandleFunc("/api/v2/health", instrumentHandler(handleHealth(gw), reqMetrics))
	mux.HandleFunc("/api/v2/metrics", handlePrometheusMetrics(reqMetrics, sysCollector))
	mux.Handle("/api/v2/", router)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	cert := tlsCert
	if cert == "" {
		cert = cfg.API.TLSCert
	}
	key := tlsKey
	if key == "" {
		key = cfg.API.TLSKey
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("AIMA server starting", "addr", listenAddr)

		var err error
		if cert != "" && key != "" {
			err = server.ListenAndServeTLS(cert, key)
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
	case <-ctx.Done():
		slog.Info("context cancelled, shutting down")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		slog.Info("received signal, shutting down gracefully", "signal", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("AIMA server stopped")
	return nil
}

func handleExecute(gw *gateway.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit request body to 10 MB to prevent memory exhaustion.
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		var req gateway.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "failed to decode request body")
			return
		}

		resp := gw.Handle(r.Context(), &req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleHealth(gw *gateway.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   cliVersion,
		})
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// instrumentHandler wraps an HTTP handler to record request metrics.
func instrumentHandler(next http.HandlerFunc, rm *metrics.RequestMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)
		isError := rw.statusCode >= 500
		rm.Record(time.Since(start), isError)
	}
}

// responseWriter captures the HTTP status code written by a handler.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

// handlePrometheusMetrics returns an HTTP handler that serves metrics in
// Prometheus text exposition format (version 0.0.4).
func handlePrometheusMetrics(rm *metrics.RequestMetrics, sc metrics.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var sb strings.Builder

		// --- HTTP request metrics ---
		snap := rm.Snapshot()

		sb.WriteString("# HELP aima_http_requests_total Total number of HTTP requests processed.\n")
		sb.WriteString("# TYPE aima_http_requests_total counter\n")
		fmt.Fprintf(&sb, "aima_http_requests_total %d\n", snap.TotalRequests)

		sb.WriteString("# HELP aima_http_errors_total Total number of HTTP requests that resulted in a 5xx error.\n")
		sb.WriteString("# TYPE aima_http_errors_total counter\n")
		fmt.Fprintf(&sb, "aima_http_errors_total %d\n", snap.TotalErrors)

		// Expose the accumulated duration sum so Prometheus can compute rates and averages.
		sb.WriteString("# HELP aima_http_request_duration_ms_sum Total accumulated HTTP request latency in milliseconds.\n")
		sb.WriteString("# TYPE aima_http_request_duration_ms_sum counter\n")
		fmt.Fprintf(&sb, "aima_http_request_duration_ms_sum %d\n", snap.TotalLatencyMs)

		// --- System metrics (best-effort; available on Linux and Windows) ---
		sysMetrics, err := sc.Collect(r.Context())
		if err == nil {
			sb.WriteString("# HELP aima_system_cpu_usage_percent Current CPU usage percentage (0–100).\n")
			sb.WriteString("# TYPE aima_system_cpu_usage_percent gauge\n")
			fmt.Fprintf(&sb, "aima_system_cpu_usage_percent %.3f\n", sysMetrics.CPU)

			sb.WriteString("# HELP aima_system_memory_used_bytes Memory used in bytes.\n")
			sb.WriteString("# TYPE aima_system_memory_used_bytes gauge\n")
			fmt.Fprintf(&sb, "aima_system_memory_used_bytes %d\n", sysMetrics.Memory.Used)

			sb.WriteString("# HELP aima_system_memory_total_bytes Total memory in bytes.\n")
			sb.WriteString("# TYPE aima_system_memory_total_bytes gauge\n")
			fmt.Fprintf(&sb, "aima_system_memory_total_bytes %d\n", sysMetrics.Memory.Total)

			sb.WriteString("# HELP aima_system_memory_usage_percent Memory usage percentage (0–100).\n")
			sb.WriteString("# TYPE aima_system_memory_usage_percent gauge\n")
			fmt.Fprintf(&sb, "aima_system_memory_usage_percent %.3f\n", sysMetrics.Memory.Percent)

			sb.WriteString("# HELP aima_system_disk_used_bytes Disk space used in bytes.\n")
			sb.WriteString("# TYPE aima_system_disk_used_bytes gauge\n")
			fmt.Fprintf(&sb, "aima_system_disk_used_bytes %d\n", sysMetrics.Disk.Used)

			sb.WriteString("# HELP aima_system_disk_total_bytes Total disk space in bytes.\n")
			sb.WriteString("# TYPE aima_system_disk_total_bytes gauge\n")
			fmt.Fprintf(&sb, "aima_system_disk_total_bytes %d\n", sysMetrics.Disk.Total)

			sb.WriteString("# HELP aima_system_disk_usage_percent Disk usage percentage (0–100).\n")
			sb.WriteString("# TYPE aima_system_disk_usage_percent gauge\n")
			fmt.Fprintf(&sb, "aima_system_disk_usage_percent %.3f\n", sysMetrics.Disk.Percent)

			sb.WriteString("# HELP aima_system_network_bytes_sent_total Total bytes sent over the network.\n")
			sb.WriteString("# TYPE aima_system_network_bytes_sent_total counter\n")
			fmt.Fprintf(&sb, "aima_system_network_bytes_sent_total %d\n", sysMetrics.Network.BytesSent)

			sb.WriteString("# HELP aima_system_network_bytes_recv_total Total bytes received over the network.\n")
			sb.WriteString("# TYPE aima_system_network_bytes_recv_total counter\n")
			fmt.Fprintf(&sb, "aima_system_network_bytes_recv_total %d\n", sysMetrics.Network.BytesRecv)
		} else {
			slog.Debug("system metrics unavailable", "error", err)
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sb.String()))
	}
}

