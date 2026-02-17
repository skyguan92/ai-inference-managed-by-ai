package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/remote"
)

type RemoteService struct {
	registry *unit.Registry
	store    remote.RemoteStore
	provider remote.RemoteProvider
	bus      *eventbus.InMemoryEventBus
}

func NewRemoteService(registry *unit.Registry, store remote.RemoteStore, provider remote.RemoteProvider, bus *eventbus.InMemoryEventBus) *RemoteService {
	return &RemoteService{
		registry: registry,
		store:    store,
		provider: provider,
		bus:      bus,
	}
}

type EnableWithVerifyResult struct {
	TunnelID  string
	PublicURL string
	Verified  bool
	Provider  remote.TunnelProvider
}

func (s *RemoteService) EnableWithVerify(ctx context.Context, provider remote.TunnelProvider, config map[string]any) (*EnableWithVerifyResult, error) {
	enableCmd := s.registry.GetCommand("remote.enable")
	if enableCmd == nil {
		return nil, fmt.Errorf("remote.enable command not found")
	}

	input := map[string]any{
		"provider": string(provider),
	}
	if config != nil {
		input["config"] = config
	}

	result, err := enableCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("enable tunnel: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected enable result type")
	}

	tunnelID := getString(resultMap, "tunnel_id")
	publicURL := getString(resultMap, "public_url")

	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	verified := s.verifyTunnelConnection(verifyCtx, publicURL)

	s.publishEvent(ctx, "remote.enabled_and_verified", map[string]any{
		"tunnel_id":  tunnelID,
		"public_url": publicURL,
		"provider":   string(provider),
		"verified":   verified,
	})

	return &EnableWithVerifyResult{
		TunnelID:  tunnelID,
		PublicURL: publicURL,
		Verified:  verified,
		Provider:  provider,
	}, nil
}

func (s *RemoteService) verifyTunnelConnection(ctx context.Context, publicURL string) bool {
	if publicURL == "" {
		return false
	}

	statusQuery := s.registry.GetQuery("remote.status")
	if statusQuery == nil {
		return false
	}

	result, err := statusQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return false
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return false
	}

	return getBool(resultMap, "enabled")
}

type DisableWithCleanupResult struct {
	Success       bool
	ClearedAudit  bool
	TunnelID      string
	AuditRecorded bool
}

func (s *RemoteService) DisableWithCleanup(ctx context.Context) (*DisableWithCleanupResult, error) {
	tunnel, err := s.store.GetTunnel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tunnel: %w", err)
	}

	tunnelID := ""
	if tunnel != nil {
		tunnelID = tunnel.ID
	}

	disableCmd := s.registry.GetCommand("remote.disable")
	if disableCmd == nil {
		return nil, fmt.Errorf("remote.disable command not found")
	}

	_, err = disableCmd.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("disable tunnel: %w", err)
	}

	s.publishEvent(ctx, "remote.disabled_with_cleanup", map[string]any{
		"tunnel_id": tunnelID,
	})

	return &DisableWithCleanupResult{
		Success:       true,
		ClearedAudit:  false,
		TunnelID:      tunnelID,
		AuditRecorded: true,
	}, nil
}

type ExecWithTimeoutResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Timeout  bool
	Duration time.Duration
}

func (s *RemoteService) ExecWithTimeout(ctx context.Context, command string, timeoutSeconds int) (*ExecWithTimeoutResult, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	if timeoutSeconds > 3600 {
		timeoutSeconds = 3600
	}

	execCmd := s.registry.GetCommand("remote.exec")
	if execCmd == nil {
		return nil, fmt.Errorf("remote.exec command not found")
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds+5)*time.Second)
	defer cancel()

	startTime := time.Now()

	resultChan := make(chan struct {
		result map[string]any
		err    error
	})

	go func() {
		result, err := execCmd.Execute(timeoutCtx, map[string]any{
			"command": command,
			"timeout": timeoutSeconds,
		})
		var resultMap map[string]any
		if result != nil {
			resultMap, _ = result.(map[string]any)
		}
		select {
		case <-timeoutCtx.Done():
			return
		case resultChan <- struct {
			result map[string]any
			err    error
		}{resultMap, err}:
		}
	}()

	select {
	case <-timeoutCtx.Done():
		s.publishEvent(ctx, "remote.exec_timeout", map[string]any{
			"command": command,
			"timeout": timeoutSeconds,
		})
		return &ExecWithTimeoutResult{
			Timeout:  true,
			Duration: time.Since(startTime),
		}, fmt.Errorf("command execution timeout after %d seconds", timeoutSeconds)
	case res := <-resultChan:
		if res.err != nil {
			return nil, fmt.Errorf("execute command: %w", res.err)
		}

		s.publishEvent(ctx, "remote.exec_completed", map[string]any{
			"command":    command,
			"exit_code":  getInt(res.result, "exit_code"),
			"durationms": time.Since(startTime).Milliseconds(),
		})

		return &ExecWithTimeoutResult{
			Stdout:   getString(res.result, "stdout"),
			Stderr:   getString(res.result, "stderr"),
			ExitCode: getInt(res.result, "exit_code"),
			Timeout:  false,
			Duration: time.Since(startTime),
		}, nil
	}
}

type GetStatusResult struct {
	Enabled   bool
	Provider  remote.TunnelProvider
	PublicURL string
	Uptime    int64
	TunnelID  string
	Status    remote.TunnelStatus
	StartedAt time.Time
}

func (s *RemoteService) GetStatus(ctx context.Context) (*GetStatusResult, error) {
	statusQuery := s.registry.GetQuery("remote.status")
	if statusQuery == nil {
		return nil, fmt.Errorf("remote.status query not found")
	}

	result, err := statusQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected status result type")
	}

	statusResult := &GetStatusResult{
		Enabled:   getBool(resultMap, "enabled"),
		Provider:  remote.TunnelProvider(getString(resultMap, "provider")),
		PublicURL: getString(resultMap, "public_url"),
		Uptime:    getInt64(resultMap, "uptime_seconds"),
	}

	if statusResult.Enabled {
		tunnel, err := s.store.GetTunnel(ctx)
		if err == nil && tunnel != nil {
			statusResult.TunnelID = tunnel.ID
			statusResult.Status = tunnel.Status
			statusResult.StartedAt = tunnel.StartedAt
		}
	}

	return statusResult, nil
}

type AuditLogRecord struct {
	ID        string
	Command   string
	ExitCode  int
	Timestamp time.Time
	Duration  int
}

type GetAuditLogResult struct {
	Records []AuditLogRecord
	Total   int
}

func (s *RemoteService) GetAuditLog(ctx context.Context, since time.Time, limit int) (*GetAuditLogResult, error) {
	auditQuery := s.registry.GetQuery("remote.audit")
	if auditQuery == nil {
		return nil, fmt.Errorf("remote.audit query not found")
	}

	input := map[string]any{}
	if !since.IsZero() {
		input["since"] = since.Format(time.RFC3339)
	}
	if limit > 0 {
		input["limit"] = limit
	} else {
		input["limit"] = 100
	}

	result, err := auditQuery.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected audit result type")
	}

	records := []AuditLogRecord{}
	if items, ok := resultMap["records"].([]any); ok {
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				timestamp, _ := time.Parse(time.RFC3339, getString(m, "timestamp"))
				records = append(records, AuditLogRecord{
					ID:        getString(m, "id"),
					Command:   getString(m, "command"),
					ExitCode:  getInt(m, "exit_code"),
					Timestamp: timestamp,
					Duration:  getInt(m, "duration_ms"),
				})
			}
		}
	}

	return &GetAuditLogResult{
		Records: records,
		Total:   len(records),
	}, nil
}

func (s *RemoteService) IsEnabled(ctx context.Context) (bool, error) {
	tunnel, err := s.store.GetTunnel(ctx)
	if err != nil {
		return false, nil
	}
	return tunnel != nil && tunnel.Status == remote.TunnelStatusConnected, nil
}

func (s *RemoteService) GetPublicURL(ctx context.Context) (string, error) {
	tunnel, err := s.store.GetTunnel(ctx)
	if err != nil {
		return "", fmt.Errorf("tunnel not available: %w", err)
	}
	if tunnel == nil || tunnel.Status != remote.TunnelStatusConnected {
		return "", fmt.Errorf("tunnel not connected")
	}
	return tunnel.PublicURL, nil
}

func (s *RemoteService) publishEvent(ctx context.Context, eventType string, payload any) {
	if s.bus == nil {
		return
	}

	evt := &BaseEvent{
		eventType: eventType,
		domain:    "remote",
		payload:   payload,
	}

	_ = s.bus.Publish(evt)
}
