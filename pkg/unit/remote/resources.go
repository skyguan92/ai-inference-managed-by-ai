package remote

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type StatusResource struct {
	store RemoteStore
}

func NewStatusResource(store RemoteStore) *StatusResource {
	return &StatusResource{store: store}
}

func (r *StatusResource) URI() string {
	return "asms://remote/status"
}

func (r *StatusResource) Domain() string {
	return "remote"
}

func (r *StatusResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Remote tunnel status resource",
		Properties: map[string]unit.Field{
			"enabled":        {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
			"status":         {Name: "status", Schema: unit.Schema{Type: "string"}},
			"provider":       {Name: "provider", Schema: unit.Schema{Type: "string"}},
			"public_url":     {Name: "public_url", Schema: unit.Schema{Type: "string"}},
			"tunnel_id":      {Name: "tunnel_id", Schema: unit.Schema{Type: "string"}},
			"started_at":     {Name: "started_at", Schema: unit.Schema{Type: "string"}},
			"uptime_seconds": {Name: "uptime_seconds", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *StatusResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	tunnel, err := r.store.GetTunnel(ctx)
	if err != nil {
		return map[string]any{
			"enabled":        false,
			"status":         string(TunnelStatusDisconnected),
			"provider":       "",
			"public_url":     "",
			"tunnel_id":      "",
			"started_at":     "",
			"uptime_seconds": 0,
		}, nil
	}

	uptime := int64(0)
	startedAt := ""
	if !tunnel.StartedAt.IsZero() {
		uptime = int64(time.Since(tunnel.StartedAt).Seconds())
		startedAt = tunnel.StartedAt.Format(time.RFC3339)
	}

	return map[string]any{
		"enabled":        tunnel.Status == TunnelStatusConnected,
		"status":         string(tunnel.Status),
		"provider":       string(tunnel.Provider),
		"public_url":     tunnel.PublicURL,
		"tunnel_id":      tunnel.ID,
		"started_at":     startedAt,
		"uptime_seconds": uptime,
	}, nil
}

func (r *StatusResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus TunnelStatus
		var mu sync.Mutex

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				mu.Lock()
				dataMap, ok := data.(map[string]any)
				if ok {
					newStatus := TunnelStatusDisconnected
					if s, ok := dataMap["status"].(string); ok {
						newStatus = TunnelStatus(s)
					}

					if lastStatus != "" && newStatus != lastStatus {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "status_changed",
							Data:      data,
						}
					} else {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "refresh",
							Data:      data,
						}
					}
					lastStatus = newStatus
				}
				mu.Unlock()
			}
		}
	}()

	return ch, nil
}

type AuditResource struct {
	store RemoteStore
}

func NewAuditResource(store RemoteStore) *AuditResource {
	return &AuditResource{store: store}
}

func (r *AuditResource) URI() string {
	return "asms://remote/audit"
}

func (r *AuditResource) Domain() string {
	return "remote"
}

func (r *AuditResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Remote command audit log resource",
		Properties: map[string]unit.Field{
			"records": {
				Name: "records",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
							"command":     {Name: "command", Schema: unit.Schema{Type: "string"}},
							"exit_code":   {Name: "exit_code", Schema: unit.Schema{Type: "number"}},
							"timestamp":   {Name: "timestamp", Schema: unit.Schema{Type: "string"}},
							"duration_ms": {Name: "duration_ms", Schema: unit.Schema{Type: "number"}},
						},
					},
				},
			},
		},
	}
}

func (r *AuditResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	records, err := r.store.ListAuditRecords(ctx, AuditFilter{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("list audit records: %w", err)
	}

	result := make([]map[string]any, len(records))
	for i, rec := range records {
		result[i] = map[string]any{
			"id":          rec.ID,
			"command":     rec.Command,
			"exit_code":   rec.ExitCode,
			"timestamp":   rec.Timestamp.Format(time.RFC3339),
			"duration_ms": rec.Duration,
		}
	}

	return map[string]any{"records": result}, nil
}

func (r *AuditResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
				}
			}
		}
	}()

	return ch, nil
}
