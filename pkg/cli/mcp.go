package cli

import (
	"context"
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

const defaultMCPAddr = "127.0.0.1:9091"

func NewMCPCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
		Long: `Manage the MCP (Model Context Protocol) server.

MCP provides a standardized interface for AI agents to interact
with the AIMA infrastructure.`,
	}

	cmd.AddCommand(NewMCPServeCommand(root))
	cmd.AddCommand(NewMCPSSECommand(root))

	return cmd
}

func NewMCPServeCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server in stdio mode",
		Long: `Start the MCP server using standard input/output.

This mode is suitable for running as an MCP tool server
that communicates via stdin/stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServe(cmd.Context(), root)
		},
	}

	return cmd
}

func NewMCPSSECommand(root *RootCommand) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "sse",
		Short: "Start MCP server in SSE mode",
		Long: `Start the MCP server using Server-Sent Events (SSE).

This mode is suitable for web-based clients that connect
via HTTP and receive events through SSE.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPSSE(cmd.Context(), root, addr)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", defaultMCPAddr, "Listen address for SSE server")

	return cmd
}

func runMCPServe(ctx context.Context, root *RootCommand) error {
	gw := root.Gateway()
	adapter := gateway.NewMCPAdapter(gw)

	server := gateway.NewMCPServer(adapter, os.Stdin, os.Stdout, os.Stderr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("MCP server shutting down")
		server.Shutdown()
	}()

	return server.Serve(ctx)
}

func runMCPSSE(ctx context.Context, root *RootCommand, addr string) error {
	gw := root.Gateway()
	adapter := gateway.NewMCPAdapter(gw)

	server := gateway.NewMCPSSEServer(adapter, addr)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("MCP SSE server starting", "addr", addr)
		if err := server.Serve(ctx); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		slog.Info("context cancelled, shutting down")
	case err := <-errCh:
		return fmt.Errorf("MCP SSE server error: %w", err)
	case sig := <-quit:
		slog.Info("received signal, shutting down gracefully", "signal", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
