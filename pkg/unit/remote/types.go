package remote

import "time"

type TunnelStatus string

const (
	TunnelStatusDisconnected TunnelStatus = "disconnected"
	TunnelStatusConnecting   TunnelStatus = "connecting"
	TunnelStatusConnected    TunnelStatus = "connected"
	TunnelStatusError        TunnelStatus = "error"
)

type TunnelProvider string

const (
	TunnelProviderFRP        TunnelProvider = "frp"
	TunnelProviderCloudflare TunnelProvider = "cloudflare"
	TunnelProviderTailscale  TunnelProvider = "tailscale"
)

type TunnelConfig struct {
	Provider  TunnelProvider `json:"provider"`
	Server    string         `json:"server,omitempty"`
	Token     string         `json:"token,omitempty"`
	ExposeAPI bool           `json:"expose_api,omitempty"`
	ExposeMCP bool           `json:"expose_mcp,omitempty"`
}

type TunnelInfo struct {
	ID        string         `json:"id"`
	Status    TunnelStatus   `json:"status"`
	Provider  TunnelProvider `json:"provider"`
	PublicURL string         `json:"public_url,omitempty"`
	StartedAt time.Time      `json:"started_at,omitempty"`
}

type AuditRecord struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	ExitCode  int       `json:"exit_code"`
	Timestamp time.Time `json:"timestamp"`
	Duration  int       `json:"duration_ms"`
}

type EnableResult struct {
	TunnelID  string `json:"tunnel_id"`
	PublicURL string `json:"public_url"`
}

type DisableResult struct {
	Success bool `json:"success"`
}

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type StatusResult struct {
	Enabled   bool           `json:"enabled"`
	Provider  TunnelProvider `json:"provider,omitempty"`
	PublicURL string         `json:"public_url,omitempty"`
	Uptime    int64          `json:"uptime_seconds"`
}

type AuditFilter struct {
	Since time.Time `json:"since,omitempty"`
	Limit int       `json:"limit,omitempty"`
}

type AuditResult struct {
	Records []AuditRecord `json:"records"`
}
