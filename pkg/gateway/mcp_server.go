package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type MCPServer struct {
	adapter *MCPAdapter
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewMCPServer(adapter *MCPAdapter, stdin io.Reader, stdout, stderr io.Writer) *MCPServer {
	return &MCPServer{
		adapter: adapter,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}
}

func (s *MCPServer) Serve(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	defer cancel()

	scanner := bufio.NewScanner(s.stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					log.Printf("stdin scanner error: %v", err)
					return err
				}
				return nil
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			s.wg.Add(1)
			go func(data []byte) {
				defer s.wg.Done()
				s.handleLine(ctx, data)
			}(append([]byte(nil), line...))
		}
	}
}

func (s *MCPServer) handleLine(ctx context.Context, line []byte) {
	var req MCPRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.writeResponse(&MCPResponse{
			JSONRPC: JSONRPC,
			Error: &MCPError{
				Code:    MCPErrorCodeParseError,
				Message: "parse error: " + err.Error(),
			},
		})
		return
	}

	resp := s.adapter.HandleRequest(ctx, &req)
	if resp == nil {
		return
	}

	s.writeResponse(resp)
}

func (s *MCPServer) writeResponse(resp *MCPResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("failed to marshal response: %v", err)
		return
	}

	s.stdout.Write(data)
	s.stdout.Write([]byte("\n"))
}

func (s *MCPServer) Shutdown() {
	s.mu.Lock()
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

type MCPSSEServer struct {
	adapter  *MCPAdapter
	server   *http.Server
	sessions map[string]*sseSession
	mu       sync.RWMutex
}

type sseSession struct {
	id      string
	events  chan []byte
	closeCh chan struct{}
	adapter *MCPAdapter
}

func NewMCPSSEServer(adapter *MCPAdapter, addr string) *MCPSSEServer {
	s := &MCPSSEServer{
		adapter:  adapter,
		sessions: make(map[string]*sseSession),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

func (s *MCPSSEServer) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(ctx)
	}
}

func (s *MCPSSEServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for _, session := range s.sessions {
		close(session.closeCh)
	}
	s.sessions = make(map[string]*sseSession)
	s.mu.Unlock()

	return s.server.Shutdown(ctx)
}

func (s *MCPSSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sessionID := generateSessionID()
	session := &sseSession{
		id:      sessionID,
		events:  make(chan []byte, 100),
		closeCh: make(chan struct{}),
		adapter: s.adapter,
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
	}()

	fmt.Fprintf(w, "event: endpoint\ndata: /message?session=%s\n\n", sessionID)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-session.closeCh:
			return
		case data := <-session.events:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}

func (s *MCPSSEServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "session required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp := session.adapter.HandleRequest(r.Context(), &req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	select {
	case session.events <- data:
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "session buffer full", http.StatusServiceUnavailable)
	}
}

func (s *MCPSSEServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func generateSessionID() string {
	return "sess_" + randomHex(16)
}

func (s *MCPServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
