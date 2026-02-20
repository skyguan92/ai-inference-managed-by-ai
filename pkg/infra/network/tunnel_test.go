package network

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- NewFRPTunnel ---

func TestNewFRPTunnel_NotNil(t *testing.T) {
	tunnel := NewFRPTunnel("/tmp/frp-test")
	if tunnel == nil {
		t.Fatal("expected non-nil FRPTunnel")
	}
}

func TestNewFRPTunnel_ConfigPath(t *testing.T) {
	tunnel := NewFRPTunnel("/custom/config/path")
	if tunnel.configPath != "/custom/config/path" {
		t.Errorf("expected configPath '/custom/config/path', got '%s'", tunnel.configPath)
	}
}

func TestNewFRPTunnel_InitialState(t *testing.T) {
	tunnel := NewFRPTunnel("/tmp")
	if tunnel.enabled {
		t.Error("expected enabled=false for new tunnel")
	}
	if tunnel.tunnelID != "" {
		t.Errorf("expected empty tunnelID, got '%s'", tunnel.tunnelID)
	}
	if tunnel.publicURL != "" {
		t.Errorf("expected empty publicURL, got '%s'", tunnel.publicURL)
	}
}

// --- Name ---

func TestFRPTunnel_Name(t *testing.T) {
	tunnel := NewFRPTunnel("")
	if tunnel.Name() != "frp" {
		t.Errorf("expected name 'frp', got '%s'", tunnel.Name())
	}
}

// --- Status ---

func TestFRPTunnel_Status_Disabled(t *testing.T) {
	tunnel := NewFRPTunnel("")
	status, err := tunnel.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Enabled {
		t.Error("expected Enabled=false for new tunnel")
	}
	if status.URL != "" {
		t.Errorf("expected empty URL, got '%s'", status.URL)
	}
	if status.Uptime != 0 {
		t.Errorf("expected zero Uptime, got %v", status.Uptime)
	}
}

func TestFRPTunnel_Status_Enabled(t *testing.T) {
	tunnel := NewFRPTunnel("")
	// Simulate enabled state via direct field access (same package, white-box test)
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.publicURL = "example.com:8080"
	tunnel.startTime = time.Now().Add(-5 * time.Second)
	tunnel.mu.Unlock()

	status, err := tunnel.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Enabled {
		t.Error("expected Enabled=true")
	}
	if status.URL != "example.com:8080" {
		t.Errorf("expected URL 'example.com:8080', got '%s'", status.URL)
	}
	if status.Uptime < 4*time.Second {
		t.Errorf("expected Uptime >= 4s, got %v", status.Uptime)
	}
}

func TestFRPTunnel_Status_EnabledWithZeroStartTime(t *testing.T) {
	tunnel := NewFRPTunnel("")
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.publicURL = "example.com:9000"
	// startTime remains zero value
	tunnel.mu.Unlock()

	status, err := tunnel.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Enabled {
		t.Error("expected Enabled=true")
	}
	// When startTime is zero, Uptime should not be set
	if status.Uptime != 0 {
		t.Errorf("expected zero Uptime when startTime is zero, got %v", status.Uptime)
	}
}

// --- Enable: idempotent when already enabled ---

func TestFRPTunnel_Enable_AlreadyEnabled_Idempotent(t *testing.T) {
	tunnel := NewFRPTunnel("")
	// Pre-set enabled state
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.tunnelID = "tunnel-existing"
	tunnel.publicURL = "existing.example.com:8080"
	tunnel.mu.Unlock()

	// Enable should return existing values without calling findFRPCBin or touching the filesystem
	gotID, gotURL, err := tunnel.Enable(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("expected no error on idempotent Enable, got: %v", err)
	}
	if gotID != "tunnel-existing" {
		t.Errorf("expected tunnelID 'tunnel-existing', got '%s'", gotID)
	}
	if gotURL != "existing.example.com:8080" {
		t.Fatalf("expected publicURL 'existing.example.com:8080', got '%s'", gotURL)
	}
}

// --- Enable: error path — frpc binary not found ---

func TestFRPTunnel_Enable_BinaryNotFound_ReturnsError(t *testing.T) {
	// findFRPCBin will fail in CI/test environments without frpc installed.
	// We need a real config path to pass the WriteFile step first.
	tmpDir := t.TempDir()
	tunnel := NewFRPTunnel(tmpDir)

	_, _, err := tunnel.Enable(context.Background(), map[string]any{
		"server_addr": "127.0.0.1",
		"server_port": "7000",
		"local_port":  8080,
	})

	// If frpc is not installed (typical CI), we expect an error.
	// If frpc IS installed, the test would block for 2 seconds — skip gracefully.
	if err == nil {
		t.Log("frpc binary found — skipping binary-not-found test (frpc is installed)")
		// Clean up: disable to restore state
		tunnel.mu.Lock()
		tunnel.enabled = false
		tunnel.mu.Unlock()
		return
	}

	if !strings.Contains(err.Error(), "frpc") {
		t.Errorf("expected error to mention 'frpc', got: %v", err)
	}
}

// --- Disable: no-op when not enabled ---

func TestFRPTunnel_Disable_NotEnabled_NoOp(t *testing.T) {
	tunnel := NewFRPTunnel("")
	// tunnel.enabled is false by default

	err := tunnel.Disable(context.Background())
	if err != nil {
		t.Fatalf("expected no error on Disable when not enabled, got: %v", err)
	}

	// State should remain unchanged
	if tunnel.enabled {
		t.Error("expected enabled=false after no-op Disable")
	}
}

// --- Disable: enabled state → resets fields (frpc not found path) ---

func TestFRPTunnel_Disable_ResetsState_WhenBinaryNotFound(t *testing.T) {
	tunnel := NewFRPTunnel("")
	// Simulate an enabled tunnel
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.tunnelID = "tunnel-123"
	tunnel.publicURL = "myhost.com:7000"
	tunnel.startTime = time.Now()
	tunnel.mu.Unlock()

	// Disable will try findFRPCBin; in CI frpc won't be found → returns error
	// without clearing state
	err := tunnel.Disable(context.Background())

	if err != nil {
		// Binary not found — state should NOT have been reset
		if !tunnel.enabled {
			t.Error("expected tunnel to remain enabled when Disable fails (frpc not found)")
		}
		return
	}

	// Binary found — state should have been cleared
	tunnel.mu.RLock()
	defer tunnel.mu.RUnlock()
	if tunnel.enabled {
		t.Error("expected enabled=false after successful Disable")
	}
	if tunnel.tunnelID != "" {
		t.Errorf("expected empty tunnelID after Disable, got '%s'", tunnel.tunnelID)
	}
	if tunnel.publicURL != "" {
		t.Errorf("expected empty publicURL after Disable, got '%s'", tunnel.publicURL)
	}
	if !tunnel.startTime.IsZero() {
		t.Error("expected zero startTime after Disable")
	}
}

// --- generateFRPCConfig ---

func TestGenerateFRPCConfig_BasicFields(t *testing.T) {
	result := generateFRPCConfig("192.168.1.1", "7000", "", "tcp", 8080, 8080)

	if !strings.Contains(result, "server_addr = 192.168.1.1") {
		t.Errorf("expected server_addr in config, got:\n%s", result)
	}
	if !strings.Contains(result, "server_port = 7000") {
		t.Errorf("expected server_port in config, got:\n%s", result)
	}
	if !strings.Contains(result, "protocol = tcp") {
		t.Errorf("expected protocol=tcp in config, got:\n%s", result)
	}
	if !strings.Contains(result, "local_port = 8080") {
		t.Errorf("expected local_port=8080 in config, got:\n%s", result)
	}
	if !strings.Contains(result, "remote_port = 8080") {
		t.Errorf("expected remote_port=8080 in config, got:\n%s", result)
	}
}

func TestGenerateFRPCConfig_WithSubdomain(t *testing.T) {
	result := generateFRPCConfig("myserver.com", "7000", "myapp", "http", 3000, 80)

	if !strings.Contains(result, "subdomain = myapp") {
		t.Errorf("expected subdomain in config when set, got:\n%s", result)
	}
	if !strings.Contains(result, "local_port = 3000") {
		t.Errorf("expected local_port=3000 in config, got:\n%s", result)
	}
	if !strings.Contains(result, "remote_port = 80") {
		t.Errorf("expected remote_port=80 in config, got:\n%s", result)
	}
}

func TestGenerateFRPCConfig_NoSubdomain_HasRequiredSections(t *testing.T) {
	result := generateFRPCConfig("myserver.com", "7000", "", "tcp", 8080, 8080)

	if !strings.Contains(result, "[common]") {
		t.Errorf("expected [common] section in config, got:\n%s", result)
	}
	if !strings.Contains(result, "[tunnel]") {
		t.Errorf("expected [tunnel] section in config, got:\n%s", result)
	}
	if strings.Contains(result, "subdomain = ") {
		t.Errorf("expected no 'subdomain = ' line when subdomain is empty, got:\n%s", result)
	}
	// Regression: ensure no fmt.Sprintf EXTRA argument garbage in output
	if strings.Contains(result, "%!(EXTRA") {
		t.Errorf("config contains fmt.Sprintf garbage (extra arg bug): %s", result)
	}
}

func TestGenerateFRPCConfig_CommonSection(t *testing.T) {
	result := generateFRPCConfig("host", "port", "sub", "udp", 1234, 5678)

	if !strings.HasPrefix(strings.TrimSpace(result), "[common]") {
		t.Errorf("expected config to start with [common], got:\n%s", result)
	}
	if !strings.Contains(result, "[tunnel]") {
		t.Errorf("expected [tunnel] section, got:\n%s", result)
	}
}

func TestGenerateFRPCConfig_DifferentLocalAndRemotePorts(t *testing.T) {
	result := generateFRPCConfig("server", "7000", "", "tcp", 3000, 9000)

	if !strings.Contains(result, "local_port = 3000") {
		t.Errorf("expected local_port=3000, got:\n%s", result)
	}
	if !strings.Contains(result, "remote_port = 9000") {
		t.Errorf("expected remote_port=9000, got:\n%s", result)
	}
}

// --- getStringConfig ---

func TestGetStringConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]any
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "key exists with non-empty string",
			config:       map[string]any{"server_addr": "192.168.1.1"},
			key:          "server_addr",
			defaultValue: "default",
			want:         "192.168.1.1",
		},
		{
			name:         "key missing - returns default",
			config:       map[string]any{},
			key:          "server_addr",
			defaultValue: "0.0.0.0",
			want:         "0.0.0.0",
		},
		{
			name:         "key exists with empty string - returns default",
			config:       map[string]any{"server_addr": ""},
			key:          "server_addr",
			defaultValue: "0.0.0.0",
			want:         "0.0.0.0",
		},
		{
			name:         "key exists with wrong type - returns default",
			config:       map[string]any{"server_addr": 12345},
			key:          "server_addr",
			defaultValue: "0.0.0.0",
			want:         "0.0.0.0",
		},
		{
			name:         "nil config - returns default",
			config:       nil,
			key:          "server_addr",
			defaultValue: "fallback",
			want:         "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringConfig(tt.config, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, got)
			}
		})
	}
}

// --- getIntConfig ---

func TestGetIntConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{
			name:         "key exists as int",
			config:       map[string]any{"local_port": 9090},
			key:          "local_port",
			defaultValue: 8080,
			want:         9090,
		},
		{
			name:         "key exists as float64 (JSON numbers)",
			config:       map[string]any{"local_port": float64(9090)},
			key:          "local_port",
			defaultValue: 8080,
			want:         9090,
		},
		{
			name:         "key missing - returns default",
			config:       map[string]any{},
			key:          "local_port",
			defaultValue: 8080,
			want:         8080,
		},
		{
			name:         "key exists with wrong type - returns default",
			config:       map[string]any{"local_port": "8080"},
			key:          "local_port",
			defaultValue: 8080,
			want:         8080,
		},
		{
			name:         "nil config - returns default",
			config:       nil,
			key:          "local_port",
			defaultValue: 3000,
			want:         3000,
		},
		{
			name:         "zero value int",
			config:       map[string]any{"local_port": 0},
			key:          "local_port",
			defaultValue: 8080,
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getIntConfig(tt.config, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

// --- generateTunnelID ---

func TestGenerateTunnelID_NotEmpty(t *testing.T) {
	id := generateTunnelID()
	if id == "" {
		t.Error("expected non-empty tunnel ID")
	}
}

func TestGenerateTunnelID_HasPrefix(t *testing.T) {
	id := generateTunnelID()
	if !strings.HasPrefix(id, "tunnel-") {
		t.Errorf("expected tunnel ID to start with 'tunnel-', got '%s'", id)
	}
}

func TestGenerateTunnelID_Unique(t *testing.T) {
	id1 := generateTunnelID()
	time.Sleep(time.Millisecond)
	id2 := generateTunnelID()
	if id1 == id2 {
		t.Errorf("expected unique tunnel IDs, both were '%s'", id1)
	}
}

// --- findFRPCBin ---

func TestFindFRPCBin_ReturnsErrorWhenNotInstalled(t *testing.T) {
	// In most CI and test environments, frpc is not installed.
	// If it IS installed, we just verify the path is non-empty.
	path, err := findFRPCBin()
	if err != nil {
		if !strings.Contains(err.Error(), "frpc") {
			t.Errorf("expected error to mention 'frpc', got: %v", err)
		}
		// Expected in CI without frpc
		return
	}

	// frpc found — validate return value
	if path == "" {
		t.Error("expected non-empty path when frpc is found")
	}
}

// --- TunnelProvider interface compliance ---

func TestFRPTunnel_ImplementsTunnelProvider(t *testing.T) {
	var _ TunnelProvider = (*FRPTunnel)(nil)
}

// --- Concurrent access (sync.RWMutex) ---

func TestFRPTunnel_Status_ConcurrentReads_NoRace(t *testing.T) {
	tunnel := NewFRPTunnel("")
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.publicURL = "concurrent.example.com:8080"
	tunnel.startTime = time.Now()
	tunnel.mu.Unlock()

	const readers = 20
	var wg sync.WaitGroup
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := tunnel.Status(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !status.Enabled {
				t.Error("expected Enabled=true in concurrent read")
			}
		}()
	}
	wg.Wait()
}

func TestFRPTunnel_ConcurrentDisable_WhenNotEnabled_NoRace(t *testing.T) {
	// Multiple goroutines calling Disable on a non-enabled tunnel (no-op path)
	// should not race on the mutex.
	tunnel := NewFRPTunnel("")

	const goroutines = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := tunnel.Disable(context.Background())
			if err != nil {
				t.Errorf("unexpected error on no-op Disable: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestFRPTunnel_ConcurrentStatusAndDisable_NoRace(t *testing.T) {
	tunnel := NewFRPTunnel("")

	var wg sync.WaitGroup

	// Multiple Status readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = tunnel.Status(context.Background())
		}()
	}

	// Concurrent Disable writers (no-op when not enabled)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tunnel.Disable(context.Background())
		}()
	}

	wg.Wait()
}

func TestFRPTunnel_ConcurrentEnableIdempotent_NoRace(t *testing.T) {
	tunnel := NewFRPTunnel("")
	// Pre-set enabled state so Enable takes the fast idempotent path (no frpc calls)
	tunnel.mu.Lock()
	tunnel.enabled = true
	tunnel.tunnelID = "tunnel-race-test"
	tunnel.publicURL = "race.example.com:8080"
	tunnel.mu.Unlock()

	const goroutines = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, url, err := tunnel.Enable(context.Background(), map[string]any{})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if id != "tunnel-race-test" {
				t.Errorf("expected tunnelID 'tunnel-race-test', got '%s'", id)
			}
			if url != "race.example.com:8080" {
				t.Errorf("expected publicURL 'race.example.com:8080', got '%s'", url)
			}
		}()
	}
	wg.Wait()
}

// --- Enable: context cancellation handling ---

func TestFRPTunnel_Enable_ContextAlreadyCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	tunnel := NewFRPTunnel(tmpDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := tunnel.Enable(ctx, map[string]any{
		"server_addr": "127.0.0.1",
		"server_port": "7000",
	})

	// Either frpc binary not found (most CI) OR context already cancelled
	// Both are valid error paths; what matters is err != nil
	if err == nil {
		t.Error("expected error when context is cancelled or frpc not found")
	}
}
