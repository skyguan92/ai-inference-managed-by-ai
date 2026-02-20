package network

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	_ TunnelProvider = (*FRPTunnel)(nil)
)

type TunnelProvider interface {
	Name() string
	Enable(ctx context.Context, config map[string]any) (tunnelID, publicURL string, err error)
	Disable(ctx context.Context) error
	Status(ctx context.Context) (TunnelStatus, error)
}

type TunnelStatus struct {
	Enabled bool
	URL     string
	Uptime  time.Duration
}

type FRPTunnel struct {
	configPath string
	mu         sync.RWMutex
	tunnelID   string
	publicURL  string
	startTime  time.Time
	enabled    bool
}

func NewFRPTunnel(configPath string) *FRPTunnel {
	return &FRPTunnel{
		configPath: configPath,
	}
}

func (t *FRPTunnel) Name() string {
	return "frp"
}

func (t *FRPTunnel) Enable(ctx context.Context, config map[string]any) (string, string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.enabled {
		return t.tunnelID, t.publicURL, nil
	}

	serverAddr := getStringConfig(config, "server_addr", "0.0.0.0:7000")
	serverPort := getStringConfig(config, "server_port", "7000")
	subdomain := getStringConfig(config, "subdomain", "")
	protocol := getStringConfig(config, "protocol", "tcp")
	localPort := getIntConfig(config, "local_port", 8080)
	remotePort := getIntConfig(config, "remote_port", 0)

	if remotePort == 0 {
		remotePort = localPort
	}

	frpcContent := generateFRPCConfig(serverAddr, serverPort, subdomain, protocol, localPort, remotePort)

	frpcPath := filepath.Join(t.configPath, "frpc.ini")
	if err := os.WriteFile(frpcPath, []byte(frpcContent), 0644); err != nil {
		return "", "", fmt.Errorf("write frpc config: %w", err)
	}

	frpcBin, err := findFRPCBin()
	if err != nil {
		return "", "", err
	}

	cmd := exec.CommandContext(ctx, frpcBin, "-c", frpcPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("start frpc: %w", err)
	}

	time.Sleep(2 * time.Second)

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return "", "", ctx.Err()
	default:
	}

	tunnelID := generateTunnelID()
	publicURL := fmt.Sprintf("%s.%s:%d", subdomain, serverAddr, remotePort)
	if subdomain == "" {
		publicURL = fmt.Sprintf("%s:%d", serverAddr, remotePort)
	}

	t.tunnelID = tunnelID
	t.publicURL = publicURL
	t.startTime = time.Now()
	t.enabled = true

	return tunnelID, publicURL, nil
}

func (t *FRPTunnel) Disable(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		return nil
	}

	frpcBin, err := findFRPCBin()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, frpcBin, "stop", "tunnel")
	_ = cmd.Run()

	t.enabled = false
	t.tunnelID = ""
	t.publicURL = ""
	t.startTime = time.Time{}

	return nil
}

func (t *FRPTunnel) Status(ctx context.Context) (TunnelStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := TunnelStatus{
		Enabled: t.enabled,
		URL:     t.publicURL,
	}

	if t.enabled && !t.startTime.IsZero() {
		status.Uptime = time.Since(t.startTime)
	}

	return status, nil
}

func generateTunnelID() string {
	return fmt.Sprintf("tunnel-%d", time.Now().UnixNano())
}

func getStringConfig(config map[string]any, key, defaultValue string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultValue
}

func getIntConfig(config map[string]any, key string, defaultValue int) int {
	if v, ok := config[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return defaultValue
}

func generateFRPCConfig(serverAddr, serverPort, subdomain, protocol string, localPort, remotePort int) string {
	config := `[common]
server_addr = %s
server_port = %s
protocol = %s

[tunnel]
type = %s
local_ip = 127.0.0.1
local_port = %d
remote_port = %d
`
	if subdomain != "" {
		config += `subdomain = %s`
	}

	return fmt.Sprintf(config, serverAddr, serverPort, protocol, protocol, localPort, remotePort, subdomain)
}

func findFRPCBin() (string, error) {
	paths := []string{
		"frpc",
		"/usr/bin/frpc",
		"/usr/local/bin/frpc",
		filepath.Join(os.Getenv("HOME"), "frp", "frpc"),
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("frpc binary not found")
}
