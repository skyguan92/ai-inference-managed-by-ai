package agent

import (
	"context"
	"sync"
	"time"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// AgentStatusResource implements the asms://agent/status resource.
type AgentStatusResource struct {
	agent    *coreagent.Agent
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewAgentStatusResource(agent *coreagent.Agent) *AgentStatusResource {
	return &AgentStatusResource{agent: agent}
}

func (r *AgentStatusResource) URI() string    { return "asms://agent/status" }
func (r *AgentStatusResource) Domain() string { return "agent" }

func (r *AgentStatusResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Agent operator runtime status",
		Properties: map[string]unit.Field{
			"enabled":              {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
			"provider":             {Name: "provider", Schema: unit.Schema{Type: "string"}},
			"model":                {Name: "model", Schema: unit.Schema{Type: "string"}},
			"active_conversations": {Name: "active_conversations", Schema: unit.Schema{Type: "integer"}},
		},
	}
}

func (r *AgentStatusResource) Get(_ context.Context) (any, error) {
	if r.agent == nil {
		return AgentStatus{Enabled: false}, nil
	}
	return AgentStatus{
		Enabled:             true,
		Provider:            r.agent.LLMName(),
		Model:               r.agent.LLMModelName(),
		ActiveConversations: r.agent.ActiveConversationCount(),
	}, nil
}

func (r *AgentStatusResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)
	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()

	go func() {
		defer close(ch)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.mu.Lock()
				for i, w := range r.watchers {
					if w == ch {
						r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
						break
					}
				}
				r.mu.Unlock()
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}

// AgentConversationsResource implements asms://agent/conversations.
type AgentConversationsResource struct {
	agent    *coreagent.Agent
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewAgentConversationsResource(agent *coreagent.Agent) *AgentConversationsResource {
	return &AgentConversationsResource{agent: agent}
}

func (r *AgentConversationsResource) URI() string    { return "asms://agent/conversations" }
func (r *AgentConversationsResource) Domain() string { return "agent" }

func (r *AgentConversationsResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Active agent conversations",
		Properties: map[string]unit.Field{
			"conversations": {Name: "conversations", Schema: unit.Schema{Type: "array"}},
			"total":         {Name: "total", Schema: unit.Schema{Type: "integer"}},
		},
	}
}

func (r *AgentConversationsResource) Get(_ context.Context) (any, error) {
	if r.agent == nil {
		return map[string]any{"conversations": []any{}, "total": 0}, nil
	}

	convs := r.agent.ListConversations()
	summaries := make([]ConversationSummary, 0, len(convs))
	for _, c := range convs {
		summaries = append(summaries, ConversationSummary{
			ID:           c.ID,
			MessageCount: len(c.Messages),
			CreatedAt:    c.CreatedAt,
			UpdatedAt:    c.UpdatedAt,
		})
	}

	return map[string]any{
		"conversations": summaries,
		"total":         len(summaries),
	}, nil
}

func (r *AgentConversationsResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)
	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.mu.Lock()
				for i, w := range r.watchers {
					if w == ch {
						r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
						break
					}
				}
				r.mu.Unlock()
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}
