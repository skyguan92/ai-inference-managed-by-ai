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
	"syscall"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/execute", handleExecute(gw))
	mux.HandleFunc("/api/v2/health", handleHealth(gw))

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

func handleHealth(gw *gateway.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   cliVersion,
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
