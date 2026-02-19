package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway/middleware"
)

type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	EnableCORS      bool
	CORSConfig      middleware.CORSConfig
	EnableAuth      bool
	AuthConfig      middleware.AuthConfig
	Logger          *slog.Logger
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:            ":9090",
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		EnableCORS:      false,
		CORSConfig:      middleware.DefaultCORSConfig(),
		EnableAuth:      false,
		AuthConfig:      middleware.DefaultAuthConfig(),
		Logger:          nil,
	}
}

type Server struct {
	gateway *Gateway
	config  ServerConfig
	http    *http.Server
	router  *Router
	logger  *slog.Logger
}

func NewServer(gateway *Gateway, config ServerConfig) *Server {
	if config.Addr == "" {
		config.Addr = ":9090"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 60 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 10 * time.Second
	}

	router := NewRouter(gateway)

	return &Server{
		gateway: gateway,
		config:  config,
		router:  router,
		logger:  config.Logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	handler := s.buildHandler()

	mux.Handle("/api/v2/", http.StripPrefix("/api/v2", handler))

	mux.HandleFunc("/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("/health", s.handleHealth)

	s.http = &http.Server{
		Addr:         s.config.Addr,
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	if s.logger != nil {
		s.logger.Info("starting HTTP server", slog.String("addr", s.config.Addr))
	}

	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *Server) buildHandler() http.Handler {
	var handler http.Handler

	executeHandler := NewHTTPAdapter(s.gateway)
	routerHandler := s.router

	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/execute" && r.Method == http.MethodPost {
			executeHandler.ServeHTTP(w, r)
			return
		}
		routerHandler.ServeHTTP(w, r)
	})

	// Auth middleware: wire the logger and the global Enabled flag from server config.
	authCfg := s.config.AuthConfig
	authCfg.Enabled = s.config.EnableAuth
	authCfg.Logger = s.logger
	handler = middleware.Auth(authCfg)(handler)

	// CORS must run before Auth so that browser preflight OPTIONS requests
	// are answered without requiring a bearer token.
	if s.config.EnableCORS {
		handler = middleware.CORS(s.config.CORSConfig)(handler)
	}

	// Logging and Recovery are outermost so they observe every request,
	// including those rejected by Auth, and catch panics from any middleware.
	handler = middleware.Logging(s.logger)(handler)
	handler = middleware.Recovery(s.logger)(handler)

	return handler
}

func (s *Server) Stop(ctx context.Context) error {
	if s.http == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("stopping HTTP server")
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := GenerateOpenAPI(s.gateway.Registry())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(spec)
}

func (s *Server) Gateway() *Gateway {
	return s.gateway
}

func (s *Server) Config() ServerConfig {
	return s.config
}

func (s *Server) Router() *Router {
	return s.router
}
